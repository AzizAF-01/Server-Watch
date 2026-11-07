package backup

import (
	"context"
	"testing"
	"time"

	"serverwatch/internal/model"
)

func TestValidateBackupInputs(t *testing.T) {
	// 1. Validate remote path
	validPaths := []string{
		"/var/backups/serverwatch",
		"/home/ubuntu/backups",
		"~/backups",
		"/data/db_backups-2026/site_01",
	}
	for _, p := range validPaths {
		if err := ValidateRemotePath(p); err != nil {
			t.Errorf("expected valid path %q, got error: %v", p, err)
		}
	}

	invalidPaths := []string{
		"/var/backups/../etc",
		"/var/backups; rm -rf /",
		"/var/backups/`id`",
		"/var/backups/|cat",
		"/var/backups/$HOME",
	}
	for _, p := range invalidPaths {
		if err := ValidateRemotePath(p); err == nil {
			t.Errorf("expected rejection for invalid path %q", p)
		}
	}

	// 2. Validate database identifier
	validIdentifiers := []string{
		"production_db",
		"app-db",
		"wordpress_01",
		"my.db",
	}
	for _, id := range validIdentifiers {
		if err := ValidateIdentifier(id); err != nil {
			t.Errorf("expected valid identifier %q, got error: %v", id, err)
		}
	}

	invalidIdentifiers := []string{
		"db; DROP TABLE users;",
		"db' OR '1'='1",
		"db `whoami`",
		"db$(calc)",
	}
	for _, id := range invalidIdentifiers {
		if err := ValidateIdentifier(id); err == nil {
			t.Errorf("expected rejection for malicious identifier %q", id)
		}
	}
}

func TestRestoreDatabaseValidation(t *testing.T) {
	engine := &BackupEngine{}

	srv := &model.Server{
		ID:   "srv-1",
		Name: "Prod Web",
		Host: "192.168.1.100",
		Port: 22,
		User: "root",
	}
	dbCfg := &model.DatabaseConfig{
		ID:           "db-1",
		ServerID:     "srv-1",
		Type:         model.DBTypeMySQL,
		Host:         "127.0.0.1",
		Port:         3306,
		User:         "root",
		DatabaseName: "prod_db",
	}
	jobLog := &model.BackupJobLog{
		ID:           101,
		ServerID:     "srv-1",
		DatabaseName: "prod_db",
		FilePath:     "/var/backups/serverwatch/backup_mysql_prod_db_20261106.sql.gz",
	}

	// If malicious targetDBName is injected, RestoreDatabase must reject it before doing anything
	maliciousDBName := "prod_db; DROP DATABASE test; --"
	err := engine.RestoreDatabase(context.Background(), srv, dbCfg, jobLog, maliciousDBName)
	if err == nil {
		t.Fatalf("expected RestoreDatabase to reject malicious DB name %q", maliciousDBName)
	}

	// Path traversal in jobLog.FilePath must also be rejected
	traversalLog := &model.BackupJobLog{
		ID:           102,
		ServerID:     "srv-1",
		DatabaseName: "prod_db",
		FilePath:     "/var/backups/../../etc/passwd",
	}
	err = engine.RestoreDatabase(context.Background(), srv, dbCfg, traversalLog, "prod_db")
	if err == nil {
		t.Fatalf("expected RestoreDatabase to reject traversal path")
	}
}

func TestBackupSchedulerIsDue(t *testing.T) {
	sched := &BackupScheduler{}
	now := time.Date(2026, 11, 7, 14, 0, 0, 0, time.UTC)

	// Past NextRun should be due
	past := now.Add(-5 * time.Minute)
	if !sched.isDue(model.BackupSchedule{NextRun: &past}, now) {
		t.Errorf("expected schedule with past NextRun to be due")
	}

	// Future NextRun should NOT be due
	future := now.Add(1 * time.Hour)
	pastRun := now.Add(-10 * time.Minute)
	if sched.isDue(model.BackupSchedule{NextRun: &future, LastRun: &pastRun}, now) {
		t.Errorf("expected schedule with future NextRun not to be due")
	}
}
