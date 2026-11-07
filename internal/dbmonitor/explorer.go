package dbmonitor

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"serverwatch/internal/model"
)

var validSchemaRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

// AnalyzeQueries inspects active queries/connections and slow query digests on the target database instance.
func (c *DBCollector) AnalyzeQueries(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) (*model.DatabaseQueryAnalysis, error) {
	if err := ValidateDBConfig(cfg); err != nil {
		return nil, err
	}
	switch cfg.Type {
	case model.DBTypeMySQL:
		return c.analyzeMySQLQueries(ctx, srv, cfg, password)
	case model.DBTypePostgres:
		return c.analyzePostgresQueries(ctx, srv, cfg, password)
	case model.DBTypeRedis:
		return c.analyzeRedisQueries(ctx, srv, cfg, password)
	default:
		return nil, fmt.Errorf("analisis query tidak didukung untuk tipe database %s", cfg.Type)
	}
}

func (c *DBCollector) analyzeMySQLQueries(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) (*model.DatabaseQueryAnalysis, error) {
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

	procSQL := `SELECT ID, COALESCE(USER,''), COALESCE(HOST,''), COALESCE(DB,''), COALESCE(COMMAND,''), COALESCE(TIME,0), COALESCE(STATE,''), REPLACE(REPLACE(COALESCE(INFO,''), '\n', ' '), '\t', ' ') FROM information_schema.PROCESSLIST WHERE COMMAND != 'Sleep' AND INFO IS NOT NULL ORDER BY TIME DESC LIMIT 25;`
	slowSQL := `SELECT COALESCE(SCHEMA_NAME,'all'), COUNT_STAR, ROUND(AVG_TIMER_WAIT/1000000000, 2), ROUND(MAX_TIMER_WAIT/1000000000, 2), SUM_ROWS_EXAMINED, REPLACE(REPLACE(COALESCE(DIGEST_TEXT,''), '\n', ' '), '\t', ' ') FROM performance_schema.events_statements_summary_by_digest WHERE DIGEST_TEXT IS NOT NULL AND (SCHEMA_NAME IS NULL OR SCHEMA_NAME NOT IN ('performance_schema','information_schema','mysql','sys')) ORDER BY AVG_TIMER_WAIT DESC LIMIT 20;`

	combinedScript := fmt.Sprintf(
		"echo '===ACTIVE==='; %smysql -h %s -P %d -u %s -s -N -e \"%s\" 2>/dev/null || sudo -n mysql -s -N -e \"%s\" 2>/dev/null; echo '===SLOW==='; %smysql -h %s -P %d -u %s -s -N -e \"%s\" 2>/dev/null || sudo -n mysql -s -N -e \"%s\" 2>/dev/null",
		pwdEnv, host, port, user, procSQL, procSQL,
		pwdEnv, host, port, user, slowSQL, slowSQL,
	)

	out, err := c.pool.ExecuteCommand(ctx, srv, combinedScript)
	if err != nil && strings.TrimSpace(out) == "" {
		return nil, fmt.Errorf("gagal mengambil data query MySQL: %w", err)
	}

	analysis := parseMySQLQueryAnalysis(cfg.ID, out)
	return analysis, nil
}

func parseMySQLQueryAnalysis(dbConfigID string, raw string) *model.DatabaseQueryAnalysis {
	res := &model.DatabaseQueryAnalysis{
		DatabaseConfigID: dbConfigID,
		ActiveQueries:    []model.DatabaseActiveQuery{},
		SlowQueries:      []model.DatabaseSlowQuery{},
		SourceNote:       "MySQL information_schema.PROCESSLIST & performance_schema.events_statements_summary_by_digest",
	}

	sections := strings.Split(raw, "===SLOW===")
	activeBlock := strings.TrimPrefix(sections[0], "===ACTIVE===")
	for _, line := range strings.Split(strings.TrimSpace(activeBlock), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 8)
		if len(parts) < 8 {
			continue
		}
		timeSec, _ := strconv.ParseFloat(parts[5], 64)
		res.ActiveQueries = append(res.ActiveQueries, model.DatabaseActiveQuery{
			PID:      parts[0],
			User:     parts[1],
			Host:     parts[2],
			Database: parts[3],
			Command:  parts[4],
			TimeSec:  timeSec,
			State:    parts[6],
			Query:    parts[7],
		})
	}

	if len(sections) > 1 {
		slowBlock := sections[1]
		for _, line := range strings.Split(strings.TrimSpace(slowBlock), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 6)
			if len(parts) < 6 {
				continue
			}
			execCount, _ := strconv.ParseInt(parts[1], 10, 64)
			avgMs, _ := strconv.ParseFloat(parts[2], 64)
			maxMs, _ := strconv.ParseFloat(parts[3], 64)
			rowsExamined, _ := strconv.ParseInt(parts[4], 10, 64)
			res.SlowQueries = append(res.SlowQueries, model.DatabaseSlowQuery{
				SchemaName:   parts[0],
				ExecCount:    execCount,
				AvgTimeMs:    avgMs,
				MaxTimeMs:    maxMs,
				RowsExamined: rowsExamined,
				QueryText:    parts[5],
			})
		}
	}

	return res
}

func (c *DBCollector) analyzePostgresQueries(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) (*model.DatabaseQueryAnalysis, error) {
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

	activeSQL := `SELECT pid, COALESCE(usename,''), COALESCE(host(client_addr),'local'), COALESCE(datname,''), COALESCE(state,''), COALESCE(ROUND(EXTRACT(EPOCH FROM (now() - query_start))::numeric, 2), 0), COALESCE(wait_event_type,''), REPLACE(REPLACE(COALESCE(query,''), E'\n', ' '), E'\t', ' ') FROM pg_stat_activity WHERE pid <> pg_backend_pid() AND state IS NOT NULL AND state <> 'idle' ORDER BY query_start ASC LIMIT 25;`
	slowSQL := `SELECT COALESCE(d.datname,'postgres'), s.calls, ROUND(s.mean_exec_time::numeric, 2), ROUND(s.max_exec_time::numeric, 2), s.rows, REPLACE(REPLACE(s.query, E'\n', ' '), E'\t', ' ') FROM pg_stat_statements s LEFT JOIN pg_database d ON d.oid = s.dbid ORDER BY s.mean_exec_time DESC LIMIT 20;`

	script := fmt.Sprintf(
		"echo '===ACTIVE==='; %spsql -h %s -p %d -U %s -d %s -t -A -F '\t' -c \"%s\" 2>/dev/null; echo '===SLOW==='; %spsql -h %s -p %d -U %s -d %s -t -A -F '\t' -c \"%s\" 2>/dev/null",
		pwdEnv, host, port, user, dbName, activeSQL,
		pwdEnv, host, port, user, dbName, slowSQL,
	)

	out, err := c.pool.ExecuteCommand(ctx, srv, script)
	if err != nil && strings.TrimSpace(out) == "" {
		return nil, fmt.Errorf("gagal mengambil data query PostgreSQL: %w", err)
	}

	analysis := parseMySQLQueryAnalysis(cfg.ID, out)
	analysis.SourceNote = "PostgreSQL pg_stat_activity & pg_stat_statements"
	return analysis, nil
}

func (c *DBCollector) analyzeRedisQueries(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string) (*model.DatabaseQueryAnalysis, error) {
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		port = 6379
	}
	authEnv := ""
	if password != "" {
		escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
		authEnv = fmt.Sprintf("REDISCLI_AUTH='%s' ", escapedPwd)
	}

	cmd := fmt.Sprintf("echo '===CLIENTS==='; %sredis-cli -h %s -p %d CLIENT LIST 2>/dev/null | head -n 30; echo '===SLOWLOG==='; %sredis-cli -h %s -p %d --csv SLOWLOG GET 20 2>/dev/null",
		authEnv, host, port, authEnv, host, port)

	out, err := c.pool.ExecuteCommand(ctx, srv, cmd)
	if err != nil && strings.TrimSpace(out) == "" {
		return nil, fmt.Errorf("gagal mengambil data query Redis: %w", err)
	}

	return parseRedisQueryAnalysis(cfg.ID, out), nil
}

func parseRedisQueryAnalysis(dbConfigID string, raw string) *model.DatabaseQueryAnalysis {
	res := &model.DatabaseQueryAnalysis{
		DatabaseConfigID: dbConfigID,
		ActiveQueries:    []model.DatabaseActiveQuery{},
		SlowQueries:      []model.DatabaseSlowQuery{},
		SourceNote:       "Redis CLIENT LIST & SLOWLOG GET",
	}

	sections := strings.Split(raw, "===SLOWLOG===")
	clientBlock := strings.TrimPrefix(sections[0], "===CLIENTS===")
	for _, line := range strings.Split(strings.TrimSpace(clientBlock), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kv := map[string]string{}
		for _, field := range strings.Fields(line) {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) == 2 {
				kv[parts[0]] = parts[1]
			}
		}
		if kv["id"] == "" {
			continue
		}
		ageSec, _ := strconv.ParseFloat(kv["age"], 64)
		res.ActiveQueries = append(res.ActiveQueries, model.DatabaseActiveQuery{
			PID:      kv["id"],
			User:     kv["user"],
			Host:     kv["addr"],
			Database: "db" + kv["db"],
			Command:  kv["cmd"],
			TimeSec:  ageSec,
			State:    "idle=" + kv["idle"] + "s",
			Query:    kv["cmd"],
		})
	}

	if len(sections) > 1 {
		for _, line := range strings.Split(strings.TrimSpace(sections[1]), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Split(line, ",")
			if len(parts) >= 4 {
				microSec, _ := strconv.ParseFloat(strings.Trim(parts[2], `"`), 64)
				cmdParts := []string{}
				for _, p := range parts[3:] {
					cmdParts = append(cmdParts, strings.Trim(p, `"`))
				}
				res.SlowQueries = append(res.SlowQueries, model.DatabaseSlowQuery{
					SchemaName: "redis",
					ExecCount:  1,
					AvgTimeMs:  microSec / 1000.0,
					MaxTimeMs:  microSec / 1000.0,
					QueryText:  strings.Join(cmdParts, " "),
				})
			}
		}
	}

	return res
}

// ListTables retrieves table-level size, row count, and index statistics for a specific schema (SEC-16).
func (c *DBCollector) ListTables(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string, schemaName string) ([]model.DatabaseTableItem, error) {
	if err := ValidateDBConfig(cfg); err != nil {
		return nil, err
	}
	schemaName = strings.TrimSpace(schemaName)
	if schemaName == "" {
		schemaName = cfg.DatabaseName
	}
	if schemaName == "" || !validSchemaRegex.MatchString(schemaName) {
		return nil, fmt.Errorf("nama schema/database tidak valid (SEC-16)")
	}

	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	user := cfg.User

	switch cfg.Type {
	case model.DBTypeMySQL:
		if port <= 0 {
			port = 3306
		}
		if user == "" {
			user = "root"
		}
		pwdEnv := ""
		if password != "" {
			escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
			pwdEnv = fmt.Sprintf("MYSQL_PWD='%s' ", escapedPwd)
		}
		sqlQuery := fmt.Sprintf("SELECT TABLE_NAME, COALESCE(ENGINE,'InnoDB'), COALESCE(TABLE_ROWS,0), COALESCE(DATA_LENGTH,0), COALESCE(INDEX_LENGTH,0), COALESCE(DATA_LENGTH + INDEX_LENGTH,0) FROM information_schema.tables WHERE table_schema='%s' ORDER BY (DATA_LENGTH + INDEX_LENGTH) DESC LIMIT 100;", schemaName)
		cmd := fmt.Sprintf("%smysql -h %s -P %d -u %s -s -N -e \"%s\" 2>/dev/null || sudo -n mysql -s -N -e \"%s\" 2>/dev/null", pwdEnv, host, port, user, sqlQuery, sqlQuery)
		out, err := c.pool.ExecuteCommand(ctx, srv, cmd)
		if err != nil && strings.TrimSpace(out) == "" {
			return nil, fmt.Errorf("gagal memuat daftar tabel MySQL: %w", err)
		}
		return parseTableItems(out), nil

	case model.DBTypePostgres:
		if port <= 0 {
			port = 5432
		}
		if user == "" {
			user = "postgres"
		}
		pwdEnv := ""
		if password != "" {
			escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
			pwdEnv = fmt.Sprintf("PGPASSWORD='%s' ", escapedPwd)
		}
		sqlQuery := `SELECT relname, 'heap', COALESCE(n_live_tup,0), COALESCE(pg_relation_size(relid),0), COALESCE(pg_indexes_size(relid),0), COALESCE(pg_total_relation_size(relid),0) FROM pg_stat_user_tables ORDER BY pg_total_relation_size(relid) DESC LIMIT 100;`
		cmd := fmt.Sprintf("%spsql -h %s -p %d -U %s -d %s -t -A -F '\t' -c \"%s\" 2>/dev/null", pwdEnv, host, port, user, schemaName, sqlQuery)
		out, err := c.pool.ExecuteCommand(ctx, srv, cmd)
		if err != nil && strings.TrimSpace(out) == "" {
			return nil, fmt.Errorf("gagal memuat daftar tabel PostgreSQL: %w", err)
		}
		return parseTableItems(out), nil

	default:
		return nil, fmt.Errorf("inspeksi tabel hanya didukung untuk MySQL/MariaDB dan PostgreSQL")
	}
}

func parseTableItems(raw string) []model.DatabaseTableItem {
	var items []model.DatabaseTableItem
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 6 {
			continue
		}
		rows, _ := strconv.ParseInt(parts[2], 10, 64)
		dataB, _ := strconv.ParseUint(parts[3], 10, 64)
		idxB, _ := strconv.ParseUint(parts[4], 10, 64)
		totB, _ := strconv.ParseUint(parts[5], 10, 64)
		items = append(items, model.DatabaseTableItem{
			TableName:  parts[0],
			Engine:     parts[1],
			RowCount:   rows,
			DataBytes:  dataB,
			IndexBytes: idxB,
			TotalBytes: totB,
		})
	}
	return items
}

// KillQuery terminates a running query by numeric PID / connection ID (SEC-05 & SEC-16).
func (c *DBCollector) KillQuery(ctx context.Context, srv *model.Server, cfg *model.DatabaseConfig, password string, pid string) error {
	if err := ValidateDBConfig(cfg); err != nil {
		return err
	}
	pidNum, err := strconv.Atoi(strings.TrimSpace(pid))
	if err != nil || pidNum <= 0 {
		return fmt.Errorf("PID/ID query tidak valid: harus berupa angka positif")
	}

	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	user := cfg.User

	switch cfg.Type {
	case model.DBTypeMySQL:
		if port <= 0 {
			port = 3306
		}
		if user == "" {
			user = "root"
		}
		pwdEnv := ""
		if password != "" {
			escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
			pwdEnv = fmt.Sprintf("MYSQL_PWD='%s' ", escapedPwd)
		}
		cmd := fmt.Sprintf("%smysql -h %s -P %d -u %s -e \"KILL QUERY %d;\" 2>&1 || sudo -n mysql -e \"KILL QUERY %d;\" 2>&1", pwdEnv, host, port, user, pidNum, pidNum)
		out, err := c.pool.ExecuteCommand(ctx, srv, cmd)
		if err != nil || strings.Contains(out, "ERROR ") {
			return fmt.Errorf("gagal menghentikan query #%d: %s", pidNum, strings.TrimSpace(out))
		}
		return nil

	case model.DBTypePostgres:
		if port <= 0 {
			port = 5432
		}
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
		cmd := fmt.Sprintf("%spsql -h %s -p %d -U %s -d %s -c \"SELECT pg_cancel_backend(%d);\" 2>&1", pwdEnv, host, port, user, dbName, pidNum)
		out, err := c.pool.ExecuteCommand(ctx, srv, cmd)
		if err != nil || strings.Contains(out, "ERROR:") {
			return fmt.Errorf("gagal membatalkan backend #%d: %s", pidNum, strings.TrimSpace(out))
		}
		return nil

	default:
		return fmt.Errorf("penghentian query tidak didukung untuk tipe %s", cfg.Type)
	}
}
