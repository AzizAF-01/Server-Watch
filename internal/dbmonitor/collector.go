package dbmonitor

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"serverwatch/internal/model"
	"serverwatch/internal/sshpool"
)

var (
	validDBIdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-\.]{0,127}$`)
	validHostRegex         = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-\.]{0,253}$`)
)

// ValidateDBConfig enforces SEC-05 and SEC-16 on database connection parameters.
func ValidateDBConfig(cfg *model.DatabaseConfig) error {
	if cfg == nil {
		return fmt.Errorf("konfigurasi database tidak boleh kosong")
	}
	host := strings.TrimSpace(cfg.Host)
	if host != "" && !validHostRegex.MatchString(host) {
		return fmt.Errorf("host database mengandung karakter tidak valid: '%s' (SEC-16)", host)
	}
	user := strings.TrimSpace(cfg.User)
	if user != "" && !validDBIdentifierRegex.MatchString(user) {
		return fmt.Errorf("user database mengandung karakter tidak valid: '%s' (SEC-16)", user)
	}
	dbName := strings.TrimSpace(cfg.DatabaseName)
	if dbName != "" && !validDBIdentifierRegex.MatchString(dbName) {
		return fmt.Errorf("nama database mengandung karakter tidak valid: '%s' (SEC-16)", dbName)
	}
	if cfg.Port < 0 || cfg.Port > 65535 {
		return fmt.Errorf("port database tidak valid: %d", cfg.Port)
	}
	return nil
}

type DBCollector struct {
	pool *sshpool.SSHPool
}

func NewDBCollector(pool *sshpool.SSHPool) *DBCollector {
	return &DBCollector{pool: pool}
}

// CollectMetrics dispatches telemetry gathering to the appropriate DBMS collector
func (c *DBCollector) CollectMetrics(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) (*model.DatabaseMetrics, error) {
	if err := ValidateDBConfig(cfg); err != nil {
		return nil, err
	}
	switch cfg.Type {
	case model.DBTypeMySQL:
		return c.collectMySQL(ctx, srv, cfg, password)
	case model.DBTypePostgres:
		return c.collectPostgres(ctx, srv, cfg, password)
	case model.DBTypeRedis:
		return c.collectRedis(ctx, srv, cfg, password)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

// collectMySQL gathers metrics via mysqladmin or lightweight information_schema checks
func (c *DBCollector) collectMySQL(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) (*model.DatabaseMetrics, error) {
	now := time.Now()
	m := &model.DatabaseMetrics{
		DatabaseConfigID: cfg.ID,
		Timestamp:        now,
		Status:           "up",
	}

	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		port = 3306
	}
	user := cfg.User
	if user == "" {
		user = "root"
	}

	pwdEnv := ""
	escapedPwd := ""
	if password != "" {
		escapedPwd = strings.ReplaceAll(password, "'", "'\\''")
		pwdEnv = fmt.Sprintf("MYSQL_PWD='%s' ", escapedPwd)
	}

	// Strategy list to connect to MySQL:
	// 1. Host TCP mysqladmin with user/password
	// 2. UNIX socket mysqladmin with user/password
	// 3. Docker exec mysqladmin (if running in Docker container, passing MYSQL_PWD via -e SEC-15)
	// 4. Host mysql CLI query
	// 5. Docker exec mysql CLI query
	candidates := []string{
		fmt.Sprintf("%smysqladmin -h %s -P %d -u %s status extended-status 2>&1", pwdEnv, host, port, user),
	}

	if host == "127.0.0.1" || host == "localhost" {
		dockerPwdEnv := ""
		if password != "" {
			dockerPwdEnv = fmt.Sprintf("-e MYSQL_PWD='%s' ", escapedPwd)
		}
		candidates = append(candidates,
			fmt.Sprintf("%smysqladmin -u %s status extended-status 2>&1", pwdEnv, user),
			fmt.Sprintf("docker exec %smysql-test mysqladmin -u %s status extended-status 2>&1", dockerPwdEnv, user),
			fmt.Sprintf("sudo -n docker exec %smysql-test mysqladmin -u %s status extended-status 2>&1", dockerPwdEnv, user),
			"sudo -n mysqladmin status extended-status 2>&1",
			fmt.Sprintf("%smysql -h %s -P %d -u %s -s -N -e \"SHOW GLOBAL STATUS; SHOW GLOBAL VARIABLES;\" 2>&1", pwdEnv, host, port, user),
			fmt.Sprintf("docker exec %smysql-test mysql -u %s -s -N -e \"SHOW GLOBAL STATUS; SHOW GLOBAL VARIABLES;\" 2>&1", dockerPwdEnv, user),
			fmt.Sprintf("sudo -n docker exec %smysql-test mysql -u %s -s -N -e \"SHOW GLOBAL STATUS; SHOW GLOBAL VARIABLES;\" 2>&1", dockerPwdEnv, user),
			"sudo -n mysql -s -N -e \"SHOW GLOBAL STATUS; SHOW GLOBAL VARIABLES;\" 2>&1",
		)
	} else {
		candidates = append(candidates,
			fmt.Sprintf("%smysql -h %s -P %d -u %s -s -N -e \"SHOW GLOBAL STATUS; SHOW GLOBAL VARIABLES;\" 2>&1", pwdEnv, host, port, user),
		)
	}

	var bestOut string
	var lastErrStr string
	success := false

	for _, cmd := range candidates {
		out, err := c.pool.ExecuteCommand(ctx, srv, cmd)
		if err != nil {
			errStr := err.Error()
			// If SSH itself cannot connect or timed out, fail fast without wasting time on subsequent candidates
			if strings.Contains(errStr, "dial tcp") || strings.Contains(errStr, "i/o timeout") || strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "handshake") {
				m.Status = "down"
				return m, fmt.Errorf("server remote tidak merespons (SSH timeout): pastikan laptop target menyala dan IP tidak berubah")
			}
		}

		outTrim := strings.TrimSpace(out)
		if strings.Contains(outTrim, "Access denied") {
			lastErrStr = "Access Denied: username atau password MySQL tidak sesuai"
		} else if strings.Contains(outTrim, "Can't connect") || strings.Contains(outTrim, "Connection refused") {
			if lastErrStr == "" {
				lastErrStr = "Tidak dapat terhubung ke MySQL port 3306: pastikan container MySQL/MariaDB sedang aktif ('docker ps')"
			}
		} else if strings.Contains(outTrim, "not found") || strings.Contains(outTrim, "No such container") {
			if lastErrStr == "" {
				lastErrStr = "Client mysqladmin belum terpasang di host VPS. Solusi tercepat di VPS: 'apt install -y mariadb-client'"
			}
		}

		if err == nil && (strings.Contains(out, "Uptime") || strings.Contains(out, "Threads")) {
			bestOut = out
			success = true
			break
		}
	}

	if !success {
		m.Status = "down"
		errMsg := "koneksi ke MySQL gagal"
		if lastErrStr != "" {
			errMsg = lastErrStr
		}
		return m, fmt.Errorf("%s", errMsg)
	}

	parseMySQLStatus(bestOut, m)

	// 2. Query Database Size if specific database configured
	if cfg.DatabaseName != "" {
		sizeCmd := fmt.Sprintf("%smysql -h %s -P %d -u %s -s -N -e \"SELECT SUM(data_length + index_length) FROM information_schema.tables WHERE table_schema='%s';\" 2>/dev/null", pwdEnv, host, port, user, cfg.DatabaseName)
		sizeOut, err := c.pool.ExecuteCommand(ctx, srv, sizeCmd)
		if err == nil {
			sizeOut = strings.TrimSpace(sizeOut)
			if sz, e := strconv.ParseUint(sizeOut, 10, 64); e == nil {
				m.DatabaseSizeBytes = sz
			}
		}
	}

	return m, nil
}

func parseMySQLStatus(out string, m *model.DatabaseMetrics) {
	lines := strings.Split(out, "\n")
	var bufferPoolReads, bufferPoolRequests float64
	var questions uint64

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Uptime:") {
			fields := strings.Fields(line)
			for i := 0; i < len(fields); i++ {
				switch fields[i] {
				case "Uptime:":
					if i+1 < len(fields) {
						if up, e := strconv.ParseUint(fields[i+1], 10, 64); e == nil {
							m.UptimeSeconds = up
						}
					}
				case "Threads:":
					if i+1 < len(fields) {
						if thr, e := strconv.Atoi(fields[i+1]); e == nil {
							m.ActiveConnections = thr
						}
					}
				case "Slow":
					if i+2 < len(fields) && fields[i+1] == "queries:" {
						if sq, e := strconv.ParseInt(fields[i+2], 10, 64); e == nil {
							m.SlowQueries = sq
						}
					}
				case "Queries":
					if i+2 < len(fields) && fields[i+1] == "per" {
						parts := strings.Split(fields[len(fields)-1], ":")
						if len(parts) > 0 {
							valStr := parts[len(parts)-1]
							if qps, e := strconv.ParseFloat(valStr, 64); e == nil {
								m.QueriesPerSec = qps
							}
						}
					}
				}
			}
		} else if strings.HasPrefix(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				key := strings.TrimSpace(parts[1])
				val := strings.TrimSpace(parts[2])
				parseMySQLVar(key, val, m, &bufferPoolReads, &bufferPoolRequests, &questions)
			}
		} else if strings.Contains(line, "\t") {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				parseMySQLVar(key, val, m, &bufferPoolReads, &bufferPoolRequests, &questions)
			}
		}
	}

	if bufferPoolRequests > 0 {
		m.CacheHitRate = math.Max(0, (1.0-(bufferPoolReads/bufferPoolRequests))*100.0)
	}
	if m.QueriesPerSec == 0 && m.UptimeSeconds > 0 && questions > 0 {
		m.QueriesPerSec = float64(questions) / float64(m.UptimeSeconds)
	}
	if m.MaxConnections == 0 {
		m.MaxConnections = 151 // MySQL standard default
	}
}

func parseMySQLVar(key, val string, m *model.DatabaseMetrics, reads, reqs *float64, questions *uint64) {
	switch strings.ToLower(key) {
	case "uptime":
		if up, e := strconv.ParseUint(val, 10, 64); e == nil && up > 0 {
			m.UptimeSeconds = up
		}
	case "threads_connected":
		if thr, e := strconv.Atoi(val); e == nil {
			m.ActiveConnections = thr
		}
	case "max_used_connections", "max_connections":
		if mc, e := strconv.Atoi(val); e == nil && mc > 0 {
			m.MaxConnections = mc
		}
	case "questions", "queries":
		if q, e := strconv.ParseUint(val, 10, 64); e == nil {
			*questions = q
		}
	case "slow_queries":
		if sq, e := strconv.ParseInt(val, 10, 64); e == nil {
			m.SlowQueries = sq
		}
	case "innodb_buffer_pool_reads":
		*reads, _ = strconv.ParseFloat(val, 64)
	case "innodb_buffer_pool_read_requests":
		*reqs, _ = strconv.ParseFloat(val, 64)
	case "version":
		m.Version = val
	}
}

// collectPostgres gathers metrics via psql / pg_stat_*
func (c *DBCollector) collectPostgres(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) (*model.DatabaseMetrics, error) {
	now := time.Now()
	m := &model.DatabaseMetrics{
		DatabaseConfigID: cfg.ID,
		Timestamp:        now,
		Status:           "up",
	}

	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		port = 5432
	}
	user := cfg.User
	if user == "" {
		user = "postgres"
	}
	dbName := cfg.DatabaseName
	if dbName == "" {
		dbName = "postgres"
	}

	pwdEnv := ""
	if password != "" {
		escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
		pwdEnv = fmt.Sprintf("PGPASSWORD='%s' ", escapedPwd)
	}

	escapedDBName := strings.ReplaceAll(dbName, "'", "''")
	sqlQuery := fmt.Sprintf(`SELECT
		COALESCE((SELECT count(*) FROM pg_stat_activity), 0),
		COALESCE((SELECT setting::int FROM pg_settings WHERE name='max_connections'), 100),
		COALESCE(numbackends, 0),
		COALESCE(xact_commit + xact_rollback, 0),
		COALESCE(blks_hit, 0),
		COALESCE(blks_read, 0),
		COALESCE(pg_database_size('%s'), 0)
	FROM pg_stat_database WHERE datname='%s';`, escapedDBName, escapedDBName)

	cmd := fmt.Sprintf("%spsql -h %s -p %d -U %s -d %s -t -A -F ',' -c \"%s\" 2>/dev/null", pwdEnv, host, port, user, dbName, sqlQuery)
	out, err := c.pool.ExecuteCommand(ctx, srv, cmd)
	if err != nil || strings.TrimSpace(out) == "" {
		m.Status = "down"
		return m, fmt.Errorf("failed to reach PostgreSQL: %v", err)
	}

	fields := strings.Split(strings.TrimSpace(out), ",")
	if len(fields) >= 7 {
		m.ActiveConnections, _ = strconv.Atoi(fields[0])
		m.MaxConnections, _ = strconv.Atoi(fields[1])
		totalTx, _ := strconv.ParseFloat(fields[3], 64)
		m.QueriesPerSec = math.Max(0, totalTx/10.0) // sample rate normalized
		blksHit, _ := strconv.ParseFloat(fields[4], 64)
		blksRead, _ := strconv.ParseFloat(fields[5], 64)
		if blksHit+blksRead > 0 {
			m.CacheHitRate = (blksHit / (blksHit + blksRead)) * 100.0
		}
		dbSize, _ := strconv.ParseUint(fields[6], 10, 64)
		m.DatabaseSizeBytes = dbSize
	}

	return m, nil
}

// collectRedis gathers metrics via redis-cli info
func (c *DBCollector) collectRedis(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) (*model.DatabaseMetrics, error) {
	now := time.Now()
	m := &model.DatabaseMetrics{
		DatabaseConfigID: cfg.ID,
		Timestamp:        now,
		Status:           "up",
	}

	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		port = 6379
	}

	// SEC-15: Pass Redis password via REDISCLI_AUTH environment variable instead of -a flag
	authEnv := ""
	if password != "" {
		escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
		authEnv = fmt.Sprintf("REDISCLI_AUTH='%s' ", escapedPwd)
	}

	cmd := fmt.Sprintf("%sredis-cli -h %s -p %d info 2>/dev/null", authEnv, host, port)
	out, err := c.pool.ExecuteCommand(ctx, srv, cmd)
	if err != nil || strings.TrimSpace(out) == "" {
		m.Status = "down"
		return m, fmt.Errorf("failed to reach Redis: %v", err)
	}

	lines := strings.Split(out, "\n")
	var hits, misses float64
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "#") || !strings.Contains(l, ":") {
			continue
		}
		parts := strings.SplitN(l, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], parts[1]
		switch key {
		case "uptime_in_seconds":
			m.UptimeSeconds, _ = strconv.ParseUint(val, 10, 64)
		case "connected_clients":
			m.ActiveConnections, _ = strconv.Atoi(val)
		case "maxclients":
			m.MaxConnections, _ = strconv.Atoi(val)
		case "used_memory":
			m.MemoryUsedBytes, _ = strconv.ParseUint(val, 10, 64)
		case "instantaneous_ops_per_sec":
			m.QueriesPerSec, _ = strconv.ParseFloat(val, 64)
		case "keyspace_hits":
			hits, _ = strconv.ParseFloat(val, 64)
		case "keyspace_misses":
			misses, _ = strconv.ParseFloat(val, 64)
		case "redis_version":
			m.Version = val
		}
	}

	if hits+misses > 0 {
		m.CacheHitRate = (hits / (hits + misses)) * 100.0
	}
	if m.MaxConnections <= 0 {
		m.MaxConnections = 10000 // Redis default
	}

	return m, nil
}

// ListDatabases retrieves all schemas/databases residing in the target instance
func (c *DBCollector) ListDatabases(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) ([]model.DatabaseSchemaItem, error) {
	if err := ValidateDBConfig(cfg); err != nil {
		return nil, err
	}
	switch cfg.Type {
	case model.DBTypeMySQL:
		return c.listMySQLDatabases(ctx, srv, cfg, password)
	case model.DBTypePostgres:
		return c.listPostgresDatabases(ctx, srv, cfg, password)
	default:
		return nil, fmt.Errorf("eksplorasi database tidak didukung untuk tipe %s", cfg.Type)
	}
}

func (c *DBCollector) listMySQLDatabases(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) ([]model.DatabaseSchemaItem, error) {
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		port = 3306
	}
	user := cfg.User
	if user == "" {
		user = "root"
	}

	pwdEnv := ""
	if password != "" {
		escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
		pwdEnv = fmt.Sprintf("MYSQL_PWD='%s' ", escapedPwd)
	}

	query := `SELECT s.schema_name, COUNT(t.table_name), COALESCE(SUM(t.data_length + t.index_length), 0) FROM information_schema.schemata s LEFT JOIN information_schema.tables t ON s.schema_name = t.table_schema GROUP BY s.schema_name ORDER BY s.schema_name ASC;`

	candidates := []string{
		fmt.Sprintf("%smysql -h %s -P %d -u %s -s -N -e \"%s\" 2>&1", pwdEnv, host, port, user, query),
	}
	if host == "127.0.0.1" || host == "localhost" {
		candidates = append(candidates,
			fmt.Sprintf("%smysql -u %s -s -N -e \"%s\" 2>&1", pwdEnv, user, query),
			fmt.Sprintf("sudo -n mysql -s -N -e \"%s\" 2>&1", query),
		)
	}

	var rawOut string
	var success bool

	for _, cmd := range candidates {
		out, err := c.pool.ExecuteCommand(ctx, srv, cmd)
		if err == nil && !strings.Contains(out, "Access denied") && !strings.Contains(out, "ERROR") && strings.TrimSpace(out) != "" {
			rawOut = out
			success = true
			break
		}
	}

	// Fallback to simple SHOW DATABASES if information_schema query didn't return any databases
	if !success || strings.TrimSpace(rawOut) == "" {
		fallbackCmd := fmt.Sprintf("%smysql -h %s -P %d -u %s -s -N -e \"SHOW DATABASES;\" 2>&1", pwdEnv, host, port, user)
		if out, err := c.pool.ExecuteCommand(ctx, srv, fallbackCmd); err == nil && !strings.Contains(out, "ERROR") && !strings.Contains(out, "Access denied") {
			rawOut = out
			success = true
		}
	}

	if !success && strings.TrimSpace(rawOut) == "" {
		return nil, fmt.Errorf("gagal mengambil daftar database: kredensial atau koneksi ditolak")
	}

	var items []model.DatabaseSchemaItem
	lines := strings.Split(strings.TrimSpace(rawOut), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		parts := strings.Split(l, "\t")
		if len(parts) >= 3 {
			tblCount, _ := strconv.Atoi(parts[1])
			sizeBytes, _ := strconv.ParseUint(parts[2], 10, 64)
			items = append(items, model.DatabaseSchemaItem{
				Name:       parts[0],
				TableCount: tblCount,
				SizeBytes:  sizeBytes,
			})
		} else if len(parts) == 1 {
			dbName := parts[0]
			if dbName != "" && dbName != "Database" {
				items = append(items, model.DatabaseSchemaItem{
					Name: dbName,
				})
			}
		}
	}

	return items, nil
}

func (c *DBCollector) listPostgresDatabases(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) ([]model.DatabaseSchemaItem, error) {
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		port = 5432
	}
	user := cfg.User
	if user == "" {
		user = "postgres"
	}

	pwdEnv := ""
	if password != "" {
		escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
		pwdEnv = fmt.Sprintf("PGPASSWORD='%s' ", escapedPwd)
	}

	sqlQuery := "SELECT datname, pg_database_size(datname) FROM pg_database WHERE NOT datistemplate ORDER BY datname ASC;"
	cmd := fmt.Sprintf("%spsql -h %s -p %d -U %s -d postgres -t -A -F ',' -c \"%s\" 2>/dev/null", pwdEnv, host, port, user, sqlQuery)

	out, err := c.pool.ExecuteCommand(ctx, srv, cmd)
	if err != nil {
		return nil, err
	}

	var items []model.DatabaseSchemaItem
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		parts := strings.Split(l, ",")
		if len(parts) >= 2 {
			sizeBytes, _ := strconv.ParseUint(parts[1], 10, 64)
			items = append(items, model.DatabaseSchemaItem{
				Name:      parts[0],
				SizeBytes: sizeBytes,
			})
		}
	}
	return items, nil
}
