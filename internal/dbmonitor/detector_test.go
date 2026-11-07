package dbmonitor

import (
	"testing"
)

func TestParseProbeOutput_LegacyFormat(t *testing.T) {
	d := &DBDetector{}
	sampleOutput := `===PORTS===
tcp        0      0 127.0.0.1:3306          0.0.0.0:*               LISTEN
===PROCS===
mysql      1234  0.0  2.5 123456 7890 ?      Ssl  12:00   0:01 /usr/sbin/mysqld
===SOCKETS===
/var/run/mysqld/mysqld.sock
===SERVICES===
mysql is running
===INSTALLED_BINS===
/usr/sbin/mysqld
===VERSIONS===
mysql  Ver 8.0.35-0ubuntu0.22.04.1 for Linux on x86_64 ((Ubuntu))
===LOCAL_MYSQL_DBS===
my_app_db
===LOCAL_PG_DBS===
`
	dbs := d.parseProbeOutput(sampleOutput)
	if len(dbs) == 0 {
		t.Fatalf("Expected at least 1 database, got 0")
	}
	if dbs[0].Name != "MySQL / MariaDB" || dbs[0].ServiceState != "running" {
		t.Errorf("Unexpected db result: %+v", dbs[0])
	}
}

func TestParseProbeOutput_WSLOperational(t *testing.T) {
	d := &DBDetector{}
	sampleOutput := `---SECTION_PORTS---
---SECTION_PROCS---
mysql      1234  0.0  2.5 123456 7890 ?      Ssl  12:00   0:01 /usr/sbin/mysqld
---SECTION_SOCKETS---
/var/run/mysqld/mysqld.sock
---SECTION_SERVICES---
 * /usr/bin/mysqladmin  Ver 8.0.35-0ubuntu0.22.04.1 for Linux on x86_64 ((Ubuntu))
Server is operational
---SECTION_INSTALLED---
/usr/sbin/mysqld
mysql-server
---SECTION_VERSIONS---
mysql  Ver 8.0.35-0ubuntu0.22.04.1
---SECTION_MYSQL_DBS---
db_kantor
---SECTION_PG_DBS---
`
	dbs := d.parseProbeOutput(sampleOutput)
	if len(dbs) == 0 {
		t.Fatalf("Expected MySQL to be detected from WSL status, got none")
	}
	if dbs[0].ServiceState != "running" {
		t.Errorf("Expected MySQL to be detected as running, got: %s", dbs[0].ServiceState)
	}
	if len(dbs[0].Databases) != 1 || dbs[0].Databases[0] != "db_kantor" {
		t.Errorf("Expected db_kantor in databases, got: %+v", dbs[0].Databases)
	}
}

func TestParseProbeOutput_StoppedInstalled(t *testing.T) {
	d := &DBDetector{}
	sampleOutput := `---SECTION_PORTS---
---SECTION_PROCS---
---SECTION_SOCKETS---
---SECTION_SERVICES---
mysql is stopped
---SECTION_INSTALLED---
/usr/sbin/mysqld
/etc/mysql
---SECTION_VERSIONS---
mysql  Ver 8.0.35
---SECTION_MYSQL_DBS---
---SECTION_PG_DBS---
`
	dbs := d.parseProbeOutput(sampleOutput)
	if len(dbs) == 0 {
		t.Fatalf("Expected stopped MySQL to be detected as installed, got none")
	}
	if dbs[0].ServiceState != "stopped" {
		t.Errorf("Expected state stopped, got: %s", dbs[0].ServiceState)
	}
}

func TestParseProbeOutput_DebianNoDBMSInstalled(t *testing.T) {
	d := &DBDetector{}
	sampleOutput := `---SECTION_PORTS---
---SECTION_PROCS---
---SECTION_SOCKETS---
---SECTION_SERVICES---
inactive
inactive
inactive
---SECTION_INSTALLED---
/etc/mysql
---SECTION_VERSIONS---
---
---
---SECTION_MYSQL_DBS---
---SECTION_PG_DBS---
`
	dbs := d.parseProbeOutput(sampleOutput)
	if len(dbs) != 0 {
		t.Fatalf("Expected 0 databases on clean Debian VPS, got: %+v", dbs)
	}
}

