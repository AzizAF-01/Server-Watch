package dbmonitor

import (
	"testing"

	"serverwatch/internal/model"
)

func TestParseMySQLStatus_Mysqladmin(t *testing.T) {
	out := `Uptime: 10000  Threads: 5  Questions: 20000  Slow queries: 2  Opens: 150  Flush tables: 1  Open tables: 64  Queries per second avg: 2.0
| Variable_name | Value |
| Max_used_connections | 25 |
| max_connections | 200 |
| Innodb_buffer_pool_reads | 50 |
| Innodb_buffer_pool_read_requests | 1000 |
`
	m := &model.DatabaseMetrics{Status: "up"}
	parseMySQLStatus(out, m)

	if m.UptimeSeconds != 10000 {
		t.Errorf("Expected uptime 10000, got %d", m.UptimeSeconds)
	}
	if m.ActiveConnections != 5 {
		t.Errorf("Expected active connections 5, got %d", m.ActiveConnections)
	}
	if m.MaxConnections != 200 {
		t.Errorf("Expected max connections 200, got %d", m.MaxConnections)
	}
	if m.SlowQueries != 2 {
		t.Errorf("Expected slow queries 2, got %d", m.SlowQueries)
	}
	if m.QueriesPerSec != 2.0 {
		t.Errorf("Expected QPS 2.0, got %f", m.QueriesPerSec)
	}
	if m.CacheHitRate < 94.0 || m.CacheHitRate > 96.0 {
		t.Errorf("Expected cache hit rate ~95%%, got %f", m.CacheHitRate)
	}
}

func TestParseMySQLStatus_TSV(t *testing.T) {
	out := "Uptime\t3600\nThreads_connected\t12\nmax_connections\t150\nSlow_queries\t1\nQuestions\t7200\nInnodb_buffer_pool_reads\t10\nInnodb_buffer_pool_read_requests\t100\n"
	m := &model.DatabaseMetrics{Status: "up"}
	parseMySQLStatus(out, m)

	if m.UptimeSeconds != 3600 {
		t.Errorf("Expected uptime 3600, got %d", m.UptimeSeconds)
	}
	if m.ActiveConnections != 12 {
		t.Errorf("Expected active connections 12, got %d", m.ActiveConnections)
	}
	if m.MaxConnections != 150 {
		t.Errorf("Expected max connections 150, got %d", m.MaxConnections)
	}
	if m.QueriesPerSec != 2.0 {
		t.Errorf("Expected QPS 2.0, got %f", m.QueriesPerSec)
	}
}

func TestValidateDBConfig(t *testing.T) {
	// Valid configs
	valid := &model.DatabaseConfig{
		Host:         "127.0.0.1",
		Port:         3306,
		User:         "root",
		DatabaseName: "app_db",
	}
	if err := ValidateDBConfig(valid); err != nil {
		t.Fatalf("expected valid config to pass, got err: %v", err)
	}

	// Malicious host
	badHost := &model.DatabaseConfig{
		Host:         "127.0.0.1; rm -rf /tmp",
		Port:         3306,
		User:         "root",
		DatabaseName: "db",
	}
	if err := ValidateDBConfig(badHost); err == nil {
		t.Errorf("expected error for shell injection in host, got nil")
	}

	// Malicious user
	badUser := &model.DatabaseConfig{
		Host:         "localhost",
		Port:         3306,
		User:         "root' --",
		DatabaseName: "db",
	}
	if err := ValidateDBConfig(badUser); err == nil {
		t.Errorf("expected error for injection in user, got nil")
	}

	// Malicious dbName
	badDB := &model.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		DatabaseName: "db; DROP DATABASE test;",
	}
	if err := ValidateDBConfig(badDB); err == nil {
		t.Errorf("expected error for SQL injection in dbName, got nil")
	}
}

func TestParseMySQLQueryAnalysis(t *testing.T) {
	raw := "===ACTIVE===\n" +
		"412\tapp_user\t10.0.0.5:51234\tecommerce\tQuery\t4.5\tSending data\tSELECT * FROM orders WHERE status = 'pending'\n" +
		"===SLOW===\n" +
		"ecommerce\t120\t245.50\t1890.00\t450000\tSELECT * FROM `orders` WHERE `status` = ?\n"

	analysis := parseMySQLQueryAnalysis("db-1", raw)
	if len(analysis.ActiveQueries) != 1 {
		t.Fatalf("expected 1 active query, got %d", len(analysis.ActiveQueries))
	}
	if analysis.ActiveQueries[0].PID != "412" || analysis.ActiveQueries[0].TimeSec != 4.5 {
		t.Fatalf("unexpected active query: %+v", analysis.ActiveQueries[0])
	}
	if len(analysis.SlowQueries) != 1 {
		t.Fatalf("expected 1 slow query, got %d", len(analysis.SlowQueries))
	}
	if analysis.SlowQueries[0].AvgTimeMs != 245.50 || analysis.SlowQueries[0].ExecCount != 120 {
		t.Fatalf("unexpected slow query: %+v", analysis.SlowQueries[0])
	}
}

func TestParseRedisQueryAnalysisAndTables(t *testing.T) {
	raw := "===CLIENTS===\n" +
		"id=15 addr=127.0.0.1:49210 fd=8 name= age=120 idle=3 flags=N db=0 sub=0 psub=0 multi=-1 qbuf=0 qbuf-free=0 obl=0 oll=0 omem=0 events=r cmd=get user=default\n" +
		"===SLOWLOG===\n" +
		"\"1\",\"1700000000\",\"15400\",\"KEYS\",\"*\"\n"

	res := parseRedisQueryAnalysis("redis-1", raw)
	if len(res.ActiveQueries) != 1 || res.ActiveQueries[0].PID != "15" {
		t.Fatalf("unexpected redis active queries: %+v", res.ActiveQueries)
	}
	if len(res.SlowQueries) != 1 || res.SlowQueries[0].AvgTimeMs != 15.4 {
		t.Fatalf("unexpected redis slow queries: %+v", res.SlowQueries)
	}

	tblOut := "orders\tInnoDB\t15420\t4194304\t1048576\t5242880\n"
	tables := parseTableItems(tblOut)
	if len(tables) != 1 || tables[0].TableName != "orders" || tables[0].TotalBytes != 5242880 {
		t.Fatalf("unexpected tables: %+v", tables)
	}
}

