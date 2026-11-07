package dbmonitor

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"serverwatch/internal/model"
	"serverwatch/internal/sshpool"
)

type DBDetector struct {
	pool *sshpool.SSHPool
}

func NewDBDetector(pool *sshpool.SSHPool) *DBDetector {
	return &DBDetector{pool: pool}
}

// DetectDatabases probes the remote OS for active DBMS services, listening ports, and UNIX sockets
func (d *DBDetector) DetectDatabases(ctx context.Context, srv *model.Server) ([]model.DiscoveredDatabase, error) {
	probeScript := `
export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH
export SYSTEMD_PAGER=cat

echo "---SECTION_PORTS---"
(ss -tulpn 2>/dev/null || ss -tuln 2>/dev/null || netstat -tulpn 2>/dev/null || netstat -tuln 2>/dev/null || lsof -iTCP -sTCP:LISTEN -P -n 2>/dev/null) | grep -E "(:3306|:33060|:5432|:6379|mysql|mariadb|postgres|redis)" || true
grep -E ":(0CEA|1538|18EB)\b" /proc/net/tcp /proc/net/tcp6 2>/dev/null || true

echo "---SECTION_PROCS---"
ps aux 2>/dev/null | grep -E "(mysqld|mariadbd|postgres|redis-server)" | grep -v grep || true

echo "---SECTION_DOCKER---"
(command -v docker >/dev/null 2>&1 && docker ps --format "{{.Names}}\t{{.Image}}\t{{.Ports}}" 2>/dev/null) || true

echo "---SECTION_SOCKETS---"
ls -1 /var/run/mysqld/mysqld.sock /run/mysqld/mysqld.sock /var/run/postgresql/.s.PGSQL.5432 /tmp/.s.PGSQL.5432 /var/run/redis/redis-server.sock /var/run/redis/redis.sock /tmp/mysql.sock 2>/dev/null || true

echo "---SECTION_SERVICES---"
systemctl is-active mysql mariadb 2>/dev/null | grep -E "^active$" && echo "mysql:active" || true
systemctl is-active postgresql 2>/dev/null | grep -E "^active$" && echo "postgresql:active" || true
systemctl is-active redis redis-server 2>/dev/null | grep -E "^active$" && echo "redis:active" || true

echo "---SECTION_INSTALLED---"
which mysqld mariadbd postgres redis-server 2>/dev/null || true
ls -1 /usr/sbin/mysqld /usr/sbin/mariadbd /usr/lib/postgresql/*/bin/postgres /usr/bin/redis-server 2>/dev/null || true

echo "---SECTION_VERSIONS---"
(mysqld -V 2>/dev/null || mariadbd -V 2>/dev/null) || true
echo "---"
(postgres -V 2>/dev/null) || true
echo "---"
(redis-server -v 2>/dev/null) || true

echo "---SECTION_MYSQL_DBS---"
(timeout 2 mysql --connect-timeout=2 -s -N -e "SHOW DATABASES;" 2>/dev/null | grep -vE "^(information_schema|performance_schema|mysql|sys)$") || true

echo "---SECTION_PG_DBS---"
(timeout 2 psql --connect-timeout=2 -t -A -c "SELECT datname FROM pg_database WHERE NOT datistemplate AND datname != 'postgres';" 2>/dev/null) || true
`

	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	rawOut, err := d.pool.ExecuteCommand(probeCtx, srv, probeScript)
	if err != nil {
		return nil, fmt.Errorf("gagal menjalankan probe deteksi database di server: %w", err)
	}

	return d.parseProbeOutput(rawOut), nil
}

func (d *DBDetector) parseProbeOutput(raw string) []model.DiscoveredDatabase {
	secMap := make(map[string]*strings.Builder)
	var currentSec string

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if (strings.HasPrefix(trimmed, "---SECTION_") && strings.HasSuffix(trimmed, "---")) ||
			(strings.HasPrefix(trimmed, "===") && strings.HasSuffix(trimmed, "===")) {
			// Extract section identifier: e.g. "PORTS", "PROCS", "SOCKETS"
			tag := strings.ToUpper(strings.Trim(trimmed, "-=_ "))
			tag = strings.TrimPrefix(tag, "SECTION_")
			currentSec = tag
			if _, ok := secMap[currentSec]; !ok {
				secMap[currentSec] = &strings.Builder{}
			}
			continue
		}
		if currentSec != "" {
			if b, ok := secMap[currentSec]; ok {
				b.WriteString(line)
				b.WriteString("\n")
			}
		}
	}

	getSec := func(name string) string {
		if b, ok := secMap[name]; ok {
			return b.String()
		}
		return ""
	}

	rawPorts := getSec("PORTS")
	rawProcs := getSec("PROCS")
	rawDocker := getSec("DOCKER")
	rawSockets := getSec("SOCKETS")
	rawServices := getSec("SERVICES")
	rawInstalled := getSec("INSTALLED")
	if rawInstalled == "" {
		rawInstalled = getSec("INSTALLED_BINS")
	}
	rawVersions := getSec("VERSIONS")
	rawMySQLDBs := getSec("MYSQL_DBS")
	if rawMySQLDBs == "" {
		rawMySQLDBs = getSec("LOCAL_MYSQL_DBS")
	}
	rawPgDBs := getSec("PG_DBS")
	if rawPgDBs == "" {
		rawPgDBs = getSec("LOCAL_PG_DBS")
	}

	rawPortsLower := strings.ToLower(rawPorts)
	rawProcsLower := strings.ToLower(rawProcs)
	rawDockerLower := strings.ToLower(rawDocker)
	rawSocketsLower := strings.ToLower(rawSockets)
	rawServicesLower := strings.ToLower(rawServices)
	rawServicesClean := strings.ReplaceAll(strings.ReplaceAll(rawServicesLower, "inactive", ""), "deactivating", "")
	rawInstalledLower := strings.ToLower(rawInstalled)
	rawVersionsLower := strings.ToLower(rawVersions)

	var results []model.DiscoveredDatabase

	// 1. MySQL / MariaDB Detection
	hasMySQLDocker := strings.Contains(rawDockerLower, "mysql") || strings.Contains(rawDockerLower, "mariadb")
	hasMySQLPort := strings.Contains(rawPorts, ":3306") || strings.Contains(rawPorts, ":33060") || strings.Contains(rawPorts, ":0CEA") || strings.Contains(rawPortsLower, "mysql") || strings.Contains(rawPortsLower, "mariadb")
	hasMySQLProc := strings.Contains(rawProcsLower, "mysqld") || strings.Contains(rawProcsLower, "mariadbd") || hasMySQLDocker
	hasMySQLSock := strings.Contains(rawSocketsLower, "mysql.sock") || strings.Contains(rawSocketsLower, "mysqld.sock")
	isMySQLServiceActive := hasMySQLDocker || strings.Contains(rawServicesClean, "mysql:active") || strings.Contains(rawServicesClean, "is running") || strings.Contains(rawServicesClean, "operational") || ((strings.Contains(rawServicesClean, "mysql") || strings.Contains(rawServicesClean, "mariadb")) && (strings.Contains(rawServicesClean, "active") || strings.Contains(rawServicesClean, "online") || strings.Contains(rawServicesClean, "started")))
	isMySQLInstalled := hasMySQLDocker || strings.Contains(rawInstalledLower, "mysqld") || strings.Contains(rawInstalledLower, "mariadbd") || strings.Contains(rawInstalledLower, "mysql-server") || strings.Contains(rawInstalledLower, "mariadb-server") || strings.Contains(rawVersionsLower, "mysqld") || strings.Contains(rawVersionsLower, "mariadbd") || strings.Contains(rawVersionsLower, "mysql  ver") || strings.Contains(rawVersionsLower, "mariadb  ver")

	if hasMySQLPort || hasMySQLProc || hasMySQLSock || isMySQLServiceActive || isMySQLInstalled {
		ver := extractVersion(rawVersions, "mysql", "mariadb")
		if ver == "" && hasMySQLDocker {
			ver = "Container (Docker)"
		}
		dbNames := extractDBNames(rawMySQLDBs)
		sockPath := ""
		if strings.Contains(rawSockets, "/run/mysqld/mysqld.sock") {
			sockPath = "/run/mysqld/mysqld.sock"
		} else if strings.Contains(rawSockets, "/var/run/mysqld/mysqld.sock") {
			sockPath = "/var/run/mysqld/mysqld.sock"
		}

		name := "MySQL / MariaDB"
		if strings.Contains(strings.ToLower(ver), "mariadb") || strings.Contains(rawInstalledLower, "mariadb") || strings.Contains(rawDockerLower, "mariadb") {
			name = "MariaDB (Docker)"
		} else if hasMySQLDocker {
			name = "MySQL (Docker)"
		}

		port := 3306
		if portMatch := regexp.MustCompile(`:(330[0-9]|331[0-9])\b`).FindStringSubmatch(rawPorts); len(portMatch) > 1 {
			if p, err := strconv.Atoi(portMatch[1]); err == nil && p > 0 {
				port = p
			}
		}

		state := "running"
		if !hasMySQLPort && !hasMySQLProc && !hasMySQLSock && !isMySQLServiceActive {
			state = "stopped"
		}

		results = append(results, model.DiscoveredDatabase{
			Type:         model.DBTypeMySQL,
			Name:         name,
			Host:         "127.0.0.1",
			Port:         port,
			SocketPath:   sockPath,
			ServiceState: state,
			Version:      ver,
			DefaultUser:  "root",
			Databases:    dbNames,
		})
	}

	// 2. PostgreSQL Detection
	hasPgPort := strings.Contains(rawPorts, ":5432") || strings.Contains(rawPorts, ":1538") || strings.Contains(rawPortsLower, "postgres")
	hasPgProc := strings.Contains(rawProcsLower, "postgres")
	hasPgSock := strings.Contains(rawSocketsLower, ".s.pgsql.5432")
	isPgServiceActive := strings.Contains(rawServicesClean, "postgresql:active") || (strings.Contains(rawServicesClean, "postgresql") && (strings.Contains(rawServicesClean, "is running") || strings.Contains(rawServicesClean, "active") || strings.Contains(rawServicesClean, "online") || strings.Contains(rawServicesClean, "started")))
	isPgInstalled := strings.Contains(rawInstalledLower, "bin/postgres") || strings.Contains(rawInstalledLower, "postgresql-server") || strings.Contains(rawInstalledLower, "postgresql-") || strings.Contains(rawVersionsLower, "postgres (postgresql)")

	if hasPgPort || hasPgProc || hasPgSock || isPgServiceActive || isPgInstalled {
		ver := extractVersion(rawVersions, "psql", "postgres")
		dbNames := extractDBNames(rawPgDBs)
		sockPath := ""
		if strings.Contains(rawSockets, "/var/run/postgresql/.s.PGSQL.5432") {
			sockPath = "/var/run/postgresql"
		}

		state := "running"
		if !hasPgPort && !hasPgProc && !hasPgSock && !isPgServiceActive {
			state = "stopped"
		}

		results = append(results, model.DiscoveredDatabase{
			Type:         model.DBTypePostgres,
			Name:         "PostgreSQL",
			Host:         "127.0.0.1",
			Port:         5432,
			SocketPath:   sockPath,
			ServiceState: state,
			Version:      ver,
			DefaultUser:  "postgres",
			Databases:    dbNames,
		})
	}

	// 3. Redis Detection
	hasRedisPort := strings.Contains(rawPorts, ":6379") || strings.Contains(rawPorts, ":18EB") || strings.Contains(rawPortsLower, "redis")
	hasRedisProc := strings.Contains(rawProcsLower, "redis-server")
	hasRedisSock := strings.Contains(rawSocketsLower, "redis.sock") || strings.Contains(rawSocketsLower, "redis-server.sock")
	isRedisServiceActive := strings.Contains(rawServicesClean, "redis:active") || (strings.Contains(rawServicesClean, "redis") && (strings.Contains(rawServicesClean, "is running") || strings.Contains(rawServicesClean, "active") || strings.Contains(rawServicesClean, "online") || strings.Contains(rawServicesClean, "started")))
	isRedisInstalled := strings.Contains(rawInstalledLower, "redis-server") || strings.Contains(rawVersionsLower, "redis server")

	if hasRedisPort || hasRedisProc || hasRedisSock || isRedisServiceActive || isRedisInstalled {
		ver := extractVersion(rawVersions, "redis")
		state := "running"
		if !hasRedisPort && !hasRedisProc && !hasRedisSock && !isRedisServiceActive {
			state = "stopped"
		}

		results = append(results, model.DiscoveredDatabase{
			Type:         model.DBTypeRedis,
			Name:         "Redis Cache",
			Host:         "127.0.0.1",
			Port:         6379,
			ServiceState: state,
			Version:      ver,
			DefaultUser:  "",
		})
	}

	return results
}

func extractVersion(rawVersions string, keywords ...string) string {
	lines := strings.Split(rawVersions, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		for _, kw := range keywords {
			if strings.Contains(strings.ToLower(l), kw) {
				return l
			}
		}
	}
	return ""
}

func extractDBNames(raw string) []string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	var dbs []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "-") && !strings.Contains(l, "datname") {
			dbs = append(dbs, l)
		}
	}
	return dbs
}
