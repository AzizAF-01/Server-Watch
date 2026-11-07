package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"serverwatch/internal/model"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(customPath string) (*SQLiteStorage, error) {
	dbPath := customPath
	if dbPath == "" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = "."
		}
		dir := filepath.Join(appData, "ServerWatch")
		_ = os.MkdirAll(dir, 0755)
		dbPath = filepath.Join(dir, "serverwatch.db")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// Single writer, multiple readers optimization
	db.SetMaxOpenConns(1)

	s := &SQLiteStorage{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

func (s *SQLiteStorage) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS servers (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		user TEXT NOT NULL,
		auth_type TEXT NOT NULL,
		key_path TEXT,
		jump_host_id TEXT,
		tags TEXT,
		notes TEXT,
		status TEXT NOT NULL,
		status_reason TEXT,
		last_check TIMESTAMP,
		host_key_fingerprint TEXT,
		trusted_host_key BOOLEAN,
		interval_sec INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		cpu_total REAL,
		mem_used_pct REAL,
		disk_max_pct REAL,
		load1 REAL,
		raw_json TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_metrics_server_time ON metrics(server_id, timestamp);

	CREATE TABLE IF NOT EXISTS alert_rules (
		id TEXT PRIMARY KEY,
		metric TEXT NOT NULL,
		operator TEXT NOT NULL,
		threshold REAL NOT NULL,
		duration_sec INTEGER NOT NULL,
		level TEXT NOT NULL,
		target_tags TEXT NOT NULL,
		enabled BOOLEAN NOT NULL,
		channels TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS alert_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id TEXT NOT NULL,
		server_name TEXT NOT NULL,
		rule_id TEXT NOT NULL,
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		started_at TIMESTAMP NOT NULL,
		resolved_at TIMESTAMP,
		status TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_alert_events ON alert_events(server_id, status);

	CREATE TABLE IF NOT EXISTS database_configs (
		id TEXT PRIMARY KEY,
		server_id TEXT NOT NULL,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		user TEXT NOT NULL,
		database_name TEXT NOT NULL,
		enabled BOOLEAN NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_db_configs_server ON database_configs(server_id);

	CREATE TABLE IF NOT EXISTS database_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		database_config_id TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		status TEXT NOT NULL,
		uptime_seconds INTEGER,
		active_connections INTEGER,
		max_connections INTEGER,
		queries_per_sec REAL,
		slow_queries INTEGER,
		cache_hit_rate REAL,
		memory_used_bytes INTEGER,
		database_size_bytes INTEGER,
		version TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_db_metrics_config ON database_metrics(database_config_id, timestamp);

	CREATE TABLE IF NOT EXISTS backup_schedules (
		id TEXT PRIMARY KEY,
		server_id TEXT NOT NULL,
		database_config_id TEXT NOT NULL,
		database_name TEXT NOT NULL,
		frequency TEXT NOT NULL,
		time_of_day TEXT NOT NULL,
		day_of_week INTEGER NOT NULL,
		retention_days INTEGER NOT NULL,
		storage_target TEXT NOT NULL,
		remote_path TEXT NOT NULL,
		local_path TEXT NOT NULL,
		compression TEXT NOT NULL,
		enabled BOOLEAN NOT NULL,
		last_run TIMESTAMP,
		next_run TIMESTAMP,
		created_at TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_backup_sched_server ON backup_schedules(server_id);

	CREATE TABLE IF NOT EXISTS backup_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		schedule_id TEXT,
		server_id TEXT NOT NULL,
		database_config_id TEXT NOT NULL,
		database_name TEXT NOT NULL,
		started_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		status TEXT NOT NULL,
		file_size_bytes INTEGER,
		file_path TEXT NOT NULL,
		duration_sec REAL,
		error_message TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_backup_logs ON backup_logs(database_config_id, started_at);

	CREATE TABLE IF NOT EXISTS uptime_monitors (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		target TEXT NOT NULL,
		server_id TEXT NOT NULL,
		interval_sec INTEGER NOT NULL,
		enabled BOOLEAN NOT NULL,
		status TEXT NOT NULL,
		status_code INTEGER,
		latency_ms INTEGER,
		ssl_expiry_days INTEGER,
		ssl_issuer TEXT,
		ssl_valid_until TEXT,
		last_error TEXT,
		last_check TIMESTAMP,
		created_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS app_settings (
		key TEXT PRIMARY KEY,
		value_json TEXT NOT NULL
	);
	`
	if _, err := s.db.Exec(query); err != nil {
		return err
	}

	// Seed default rules if empty
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM alert_rules").Scan(&count)
	if count == 0 {
		defaultRules := `
		INSERT INTO alert_rules (id, metric, operator, threshold, duration_sec, level, target_tags, enabled, channels) VALUES
		('rule_disk', 'disk.used', '>', 90, 300, 'critical', 'all', 1, 'desktop,telegram'),
		('rule_cpu', 'cpu.total', '>', 90, 300, 'warning', 'all', 1, 'desktop,telegram'),
		('rule_mem', 'mem.used', '>', 90, 600, 'warning', 'all', 1, 'desktop'),
		('rule_ssh', 'ssh.unreachable', '==', 1, 180, 'critical', 'all', 1, 'desktop,telegram');
		`
		_, _ = s.db.Exec(defaultRules)
	}

	return nil
}

func (s *SQLiteStorage) SaveServer(srv *model.Server) error {
	tagsJSON, _ := json.Marshal(srv.Tags)
	query := `
	INSERT INTO servers (
		id, name, host, port, user, auth_type, key_path, jump_host_id,
		tags, notes, status, status_reason, last_check, host_key_fingerprint,
		trusted_host_key, interval_sec, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		host = excluded.host,
		port = excluded.port,
		user = excluded.user,
		auth_type = excluded.auth_type,
		key_path = excluded.key_path,
		jump_host_id = excluded.jump_host_id,
		tags = excluded.tags,
		notes = excluded.notes,
		status = excluded.status,
		status_reason = excluded.status_reason,
		last_check = excluded.last_check,
		host_key_fingerprint = excluded.host_key_fingerprint,
		trusted_host_key = excluded.trusted_host_key,
		interval_sec = excluded.interval_sec,
		updated_at = excluded.updated_at;
	`
	_, err := s.db.Exec(
		query,
		srv.ID, srv.Name, srv.Host, srv.Port, srv.User, string(srv.AuthType),
		srv.KeyPath, srv.JumpHostID, string(tagsJSON), srv.Notes,
		string(srv.Status), srv.StatusReason, srv.LastCheck,
		srv.HostKeyFingerprint, srv.TrustedHostKey, srv.IntervalSec,
		srv.CreatedAt, srv.UpdatedAt,
	)
	return err
}

func (s *SQLiteStorage) ListServers() ([]model.Server, error) {
	rows, err := s.db.Query(`SELECT
		id, name, host, port, user, auth_type, key_path, jump_host_id,
		tags, notes, status, status_reason, last_check, host_key_fingerprint,
		trusted_host_key, interval_sec, created_at, updated_at
		FROM servers ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []model.Server
	for rows.Next() {
		var srv model.Server
		var authType, status, tagsStr string
		var lastCheck sql.NullTime

		err := rows.Scan(
			&srv.ID, &srv.Name, &srv.Host, &srv.Port, &srv.User, &authType,
			&srv.KeyPath, &srv.JumpHostID, &tagsStr, &srv.Notes,
			&status, &srv.StatusReason, &lastCheck, &srv.HostKeyFingerprint,
			&srv.TrustedHostKey, &srv.IntervalSec, &srv.CreatedAt, &srv.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		srv.AuthType = model.SSHAuthType(authType)
		srv.Status = model.ServerStatus(status)
		if lastCheck.Valid {
			srv.LastCheck = &lastCheck.Time
		}
		_ = json.Unmarshal([]byte(tagsStr), &srv.Tags)

		servers = append(servers, srv)
	}

	return servers, nil
}

func (s *SQLiteStorage) GetServer(id string) (*model.Server, error) {
	row := s.db.QueryRow(`SELECT
		id, name, host, port, user, auth_type, key_path, jump_host_id,
		tags, notes, status, status_reason, last_check, host_key_fingerprint,
		trusted_host_key, interval_sec, created_at, updated_at
		FROM servers WHERE id = ?`, id)

	var srv model.Server
	var authType, status, tagsStr string
	var lastCheck sql.NullTime

	err := row.Scan(
		&srv.ID, &srv.Name, &srv.Host, &srv.Port, &srv.User, &authType,
		&srv.KeyPath, &srv.JumpHostID, &tagsStr, &srv.Notes,
		&status, &srv.StatusReason, &lastCheck, &srv.HostKeyFingerprint,
		&srv.TrustedHostKey, &srv.IntervalSec, &srv.CreatedAt, &srv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	srv.AuthType = model.SSHAuthType(authType)
	srv.Status = model.ServerStatus(status)
	if lastCheck.Valid {
		srv.LastCheck = &lastCheck.Time
	}
	_ = json.Unmarshal([]byte(tagsStr), &srv.Tags)

	return &srv, nil
}

func (s *SQLiteStorage) DeleteServer(id string) error {
	_, err := s.db.Exec("DELETE FROM servers WHERE id = ?", id)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec("DELETE FROM metrics WHERE server_id = ?", id)
	return nil
}

func (s *SQLiteStorage) UpdateServerStatus(id string, status model.ServerStatus, reason string) error {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE servers SET status = ?, status_reason = ?, last_check = ?, updated_at = ? WHERE id = ?`,
		string(status), reason, now, now, id)
	return err
}

func (s *SQLiteStorage) SaveMetrics(m *model.ServerMetrics) error {
	rawJSON, err := json.Marshal(m)
	if err != nil {
		return err
	}

	var maxDisk float64
	for _, d := range m.Disks {
		if d.UsedPercent > maxDisk {
			maxDisk = d.UsedPercent
		}
	}

	query := `INSERT INTO metrics (server_id, timestamp, cpu_total, mem_used_pct, disk_max_pct, load1, raw_json)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.Exec(query, m.ServerID, m.Timestamp, m.CPU.TotalPercent, m.Memory.UsedPercent, maxDisk, m.CPU.Load1, string(rawJSON))
	return err
}

func (s *SQLiteStorage) GetLatestMetrics(serverID string) (*model.ServerMetrics, error) {
	row := s.db.QueryRow(`SELECT raw_json FROM metrics WHERE server_id = ? ORDER BY timestamp DESC LIMIT 1`, serverID)
	var rawJSON string
	if err := row.Scan(&rawJSON); err != nil {
		return nil, err
	}

	var m model.ServerMetrics
	if err := json.Unmarshal([]byte(rawJSON), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *SQLiteStorage) GetMetricsHistory(serverID string, duration time.Duration) ([]model.ServerMetrics, error) {
	since := time.Now().Add(-duration)
	rows, err := s.db.Query(`SELECT raw_json FROM metrics WHERE server_id = ? AND timestamp >= ? ORDER BY timestamp ASC`, serverID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.ServerMetrics
	for rows.Next() {
		var rawJSON string
		if err := rows.Scan(&rawJSON); err != nil {
			continue
		}
		var m model.ServerMetrics
		if err := json.Unmarshal([]byte(rawJSON), &m); err == nil {
			list = append(list, m)
		}
	}
	return list, nil
}

func (s *SQLiteStorage) CleanOldMetrics(retentionDays int) error {
	threshold := time.Now().AddDate(0, 0, -retentionDays)
	_, err := s.db.Exec("DELETE FROM metrics WHERE timestamp < ?", threshold)
	return err
}

func (s *SQLiteStorage) ListAlertRules() ([]model.AlertRule, error) {
	rows, err := s.db.Query("SELECT id, metric, operator, threshold, duration_sec, level, target_tags, enabled, channels FROM alert_rules")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.AlertRule
	for rows.Next() {
		var r model.AlertRule
		if err := rows.Scan(&r.ID, &r.Metric, &r.Operator, &r.Threshold, &r.DurationSec, &r.Level, &r.TargetTags, &r.Enabled, &r.Channels); err == nil {
			rules = append(rules, r)
		}
	}
	return rules, nil
}

func (s *SQLiteStorage) ToggleAlertRule(id string, enabled bool) error {
	_, err := s.db.Exec("UPDATE alert_rules SET enabled = ? WHERE id = ?", enabled, id)
	return err
}

func (s *SQLiteStorage) RecordAlertEvent(srvID, srvName, ruleID, level, message string) error {
	_, err := s.RecordAlertEventTransition(srvID, srvName, ruleID, level, message)
	return err
}

// RecordAlertEventTransition inserts an active alert if not already active and returns true when a new incident was created.
func (s *SQLiteStorage) RecordAlertEventTransition(srvID, srvName, ruleID, level, message string) (bool, error) {
	var existingID int
	err := s.db.QueryRow("SELECT id FROM alert_events WHERE server_id = ? AND rule_id = ? AND status = 'active'", srvID, ruleID).Scan(&existingID)
	if err == nil && existingID > 0 {
		return false, nil
	}

	query := `INSERT INTO alert_events (server_id, server_name, rule_id, level, message, started_at, status)
	VALUES (?, ?, ?, ?, ?, ?, 'active')`
	_, err = s.db.Exec(query, srvID, srvName, ruleID, level, message, time.Now())
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *SQLiteStorage) ResolveAlertEvent(srvID, ruleID string) error {
	_, err := s.ResolveAlertEventTransition(srvID, ruleID)
	return err
}

// ResolveAlertEventTransition resolves an active alert and returns true if an active incident was transitioned to resolved.
func (s *SQLiteStorage) ResolveAlertEventTransition(srvID, ruleID string) (bool, error) {
	now := time.Now()
	res, err := s.db.Exec("UPDATE alert_events SET status = 'resolved', resolved_at = ? WHERE server_id = ? AND rule_id = ? AND status = 'active'", now, srvID, ruleID)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (s *SQLiteStorage) ListAlertEvents(limit int) ([]model.AlertEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT id, server_id, server_name, rule_id, level, message, started_at, resolved_at, status
	FROM alert_events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.AlertEvent
	for rows.Next() {
		var e model.AlertEvent
		var resolved sql.NullTime
		if err := rows.Scan(&e.ID, &e.ServerID, &e.ServerName, &e.RuleID, &e.Level, &e.Message, &e.StartedAt, &resolved, &e.Status); err == nil {
			if resolved.Valid {
				e.ResolvedAt = &resolved.Time
			}
			events = append(events, e)
		}
	}
	return events, nil
}

func (s *SQLiteStorage) GetActiveAlerts() ([]model.AlertEvent, error) {
	rows, err := s.db.Query(`SELECT id, server_id, server_name, rule_id, level, message, started_at, resolved_at, status
	FROM alert_events WHERE status = 'active' ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.AlertEvent
	for rows.Next() {
		var e model.AlertEvent
		var resolved sql.NullTime
		if err := rows.Scan(&e.ID, &e.ServerID, &e.ServerName, &e.RuleID, &e.Level, &e.Message, &e.StartedAt, &resolved, &e.Status); err == nil {
			if resolved.Valid {
				e.ResolvedAt = &resolved.Time
			}
			events = append(events, e)
		}
	}
	return events, nil
}

// Database Configs
func (s *SQLiteStorage) SaveDatabaseConfig(cfg *model.DatabaseConfig) error {
	_, err := s.db.Exec(`INSERT INTO database_configs (id, server_id, type, name, host, port, user, database_name, enabled, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name=excluded.name,
		type=excluded.type,
		host=excluded.host,
		port=excluded.port,
		user=excluded.user,
		database_name=excluded.database_name,
		enabled=excluded.enabled,
		updated_at=excluded.updated_at`,
		cfg.ID, cfg.ServerID, string(cfg.Type), cfg.Name, cfg.Host, cfg.Port, cfg.User, cfg.DatabaseName, cfg.Enabled, cfg.CreatedAt, cfg.UpdatedAt)
	return err
}

func (s *SQLiteStorage) GetDatabaseConfig(id string) (*model.DatabaseConfig, error) {
	row := s.db.QueryRow(`SELECT id, server_id, type, name, host, port, user, database_name, enabled, created_at, updated_at
	FROM database_configs WHERE id = ?`, id)
	var c model.DatabaseConfig
	var dbType string
	if err := row.Scan(&c.ID, &c.ServerID, &dbType, &c.Name, &c.Host, &c.Port, &c.User, &c.DatabaseName, &c.Enabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	c.Type = model.DatabaseType(dbType)
	return &c, nil
}

func (s *SQLiteStorage) ListDatabaseConfigs(serverID string) ([]model.DatabaseConfig, error) {
	query := `SELECT id, server_id, type, name, host, port, user, database_name, enabled, created_at, updated_at FROM database_configs`
	var rows *sql.Rows
	var err error
	if serverID != "" {
		rows, err = s.db.Query(query+` WHERE server_id = ? ORDER BY name ASC`, serverID)
	} else {
		rows, err = s.db.Query(query + ` ORDER BY name ASC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.DatabaseConfig
	for rows.Next() {
		var c model.DatabaseConfig
		var dbType string
		if err := rows.Scan(&c.ID, &c.ServerID, &dbType, &c.Name, &c.Host, &c.Port, &c.User, &c.DatabaseName, &c.Enabled, &c.CreatedAt, &c.UpdatedAt); err == nil {
			c.Type = model.DatabaseType(dbType)
			list = append(list, c)
		}
	}
	return list, nil
}

func (s *SQLiteStorage) DeleteDatabaseConfig(id string) error {
	_, _ = s.db.Exec(`DELETE FROM database_metrics WHERE database_config_id = ?`, id)
	_, _ = s.db.Exec(`DELETE FROM backup_schedules WHERE database_config_id = ?`, id)
	_, _ = s.db.Exec(`DELETE FROM backup_logs WHERE database_config_id = ?`, id)
	_, err := s.db.Exec(`DELETE FROM database_configs WHERE id = ?`, id)
	return err
}

// Database Metrics
func (s *SQLiteStorage) SaveDatabaseMetrics(m *model.DatabaseMetrics) error {
	_, err := s.db.Exec(`INSERT INTO database_metrics
	(database_config_id, timestamp, status, uptime_seconds, active_connections, max_connections, queries_per_sec, slow_queries, cache_hit_rate, memory_used_bytes, database_size_bytes, version)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.DatabaseConfigID, m.Timestamp, m.Status, m.UptimeSeconds, m.ActiveConnections, m.MaxConnections, m.QueriesPerSec, m.SlowQueries, m.CacheHitRate, m.MemoryUsedBytes, m.DatabaseSizeBytes, m.Version)
	return err
}

func (s *SQLiteStorage) GetLatestDatabaseMetrics(dbConfigID string) (*model.DatabaseMetrics, error) {
	row := s.db.QueryRow(`SELECT id, database_config_id, timestamp, status, uptime_seconds, active_connections, max_connections, queries_per_sec, slow_queries, cache_hit_rate, memory_used_bytes, database_size_bytes, version
	FROM database_metrics WHERE database_config_id = ? ORDER BY id DESC LIMIT 1`, dbConfigID)
	var m model.DatabaseMetrics
	err := row.Scan(&m.ID, &m.DatabaseConfigID, &m.Timestamp, &m.Status, &m.UptimeSeconds, &m.ActiveConnections, &m.MaxConnections, &m.QueriesPerSec, &m.SlowQueries, &m.CacheHitRate, &m.MemoryUsedBytes, &m.DatabaseSizeBytes, &m.Version)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Backup Schedules
func (s *SQLiteStorage) SaveBackupSchedule(b *model.BackupSchedule) error {
	_, err := s.db.Exec(`INSERT INTO backup_schedules
	(id, server_id, database_config_id, database_name, frequency, time_of_day, day_of_week, retention_days, storage_target, remote_path, local_path, compression, enabled, last_run, next_run, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		database_name=excluded.database_name,
		frequency=excluded.frequency,
		time_of_day=excluded.time_of_day,
		day_of_week=excluded.day_of_week,
		retention_days=excluded.retention_days,
		storage_target=excluded.storage_target,
		remote_path=excluded.remote_path,
		local_path=excluded.local_path,
		compression=excluded.compression,
		enabled=excluded.enabled,
		last_run=excluded.last_run,
		next_run=excluded.next_run`,
		b.ID, b.ServerID, b.DatabaseConfigID, b.DatabaseName, b.Frequency, b.TimeOfDay, b.DayOfWeek, b.RetentionDays, b.StorageTarget, b.RemotePath, b.LocalPath, b.Compression, b.Enabled, b.LastRun, b.NextRun, b.CreatedAt)
	return err
}

func (s *SQLiteStorage) ListBackupSchedules(serverID string) ([]model.BackupSchedule, error) {
	query := `SELECT id, server_id, database_config_id, database_name, frequency, time_of_day, day_of_week, retention_days, storage_target, remote_path, local_path, compression, enabled, last_run, next_run, created_at FROM backup_schedules`
	var rows *sql.Rows
	var err error
	if serverID != "" {
		rows, err = s.db.Query(query+` WHERE server_id = ? ORDER BY created_at DESC`, serverID)
	} else {
		rows, err = s.db.Query(query + ` ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.BackupSchedule
	for rows.Next() {
		var b model.BackupSchedule
		var last, next sql.NullTime
		if err := rows.Scan(&b.ID, &b.ServerID, &b.DatabaseConfigID, &b.DatabaseName, &b.Frequency, &b.TimeOfDay, &b.DayOfWeek, &b.RetentionDays, &b.StorageTarget, &b.RemotePath, &b.LocalPath, &b.Compression, &b.Enabled, &last, &next, &b.CreatedAt); err == nil {
			if last.Valid {
				b.LastRun = &last.Time
			}
			if next.Valid {
				b.NextRun = &next.Time
			}
			list = append(list, b)
		}
	}
	return list, nil
}

func (s *SQLiteStorage) ListAllActiveBackupSchedules() ([]model.BackupSchedule, error) {
	rows, err := s.db.Query(`SELECT id, server_id, database_config_id, database_name, frequency, time_of_day, day_of_week, retention_days, storage_target, remote_path, local_path, compression, enabled, last_run, next_run, created_at
	FROM backup_schedules WHERE enabled = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.BackupSchedule
	for rows.Next() {
		var b model.BackupSchedule
		var last, next sql.NullTime
		if err := rows.Scan(&b.ID, &b.ServerID, &b.DatabaseConfigID, &b.DatabaseName, &b.Frequency, &b.TimeOfDay, &b.DayOfWeek, &b.RetentionDays, &b.StorageTarget, &b.RemotePath, &b.LocalPath, &b.Compression, &b.Enabled, &last, &next, &b.CreatedAt); err == nil {
			if last.Valid {
				b.LastRun = &last.Time
			}
			if next.Valid {
				b.NextRun = &next.Time
			}
			list = append(list, b)
		}
	}
	return list, nil
}

func (s *SQLiteStorage) UpdateBackupScheduleRun(id string, lastRun time.Time, nextRun time.Time) error {
	_, err := s.db.Exec(`UPDATE backup_schedules SET last_run = ?, next_run = ? WHERE id = ?`, lastRun, nextRun, id)
	return err
}

func (s *SQLiteStorage) DeleteBackupSchedule(id string) error {
	_, err := s.db.Exec(`DELETE FROM backup_schedules WHERE id = ?`, id)
	return err
}

// Backup Logs
func (s *SQLiteStorage) RecordBackupJobStart(log *model.BackupJobLog) (int, error) {
	res, err := s.db.Exec(`INSERT INTO backup_logs
	(schedule_id, server_id, database_config_id, database_name, started_at, status, file_size_bytes, file_path, duration_sec, error_message)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ScheduleID, log.ServerID, log.DatabaseConfigID, log.DatabaseName, log.StartedAt, log.Status, log.FileSizeBytes, log.FilePath, log.DurationSec, log.ErrorMessage)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *SQLiteStorage) UpdateBackupJobFinish(id int, status string, fileSize int64, filePath string, duration float64, errMsg string) error {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE backup_logs SET
		completed_at = ?,
		status = ?,
		file_size_bytes = ?,
		file_path = ?,
		duration_sec = ?,
		error_message = ?
	WHERE id = ?`, now, status, fileSize, filePath, duration, errMsg, id)
	return err
}

func (s *SQLiteStorage) GetBackupLog(id int) (*model.BackupJobLog, error) {
	row := s.db.QueryRow(`SELECT id, schedule_id, server_id, database_config_id, database_name, started_at, completed_at, status, file_size_bytes, file_path, duration_sec, error_message
	FROM backup_logs WHERE id = ?`, id)
	var l model.BackupJobLog
	var comp sql.NullTime
	var schedID, errMsg sql.NullString
	if err := row.Scan(&l.ID, &schedID, &l.ServerID, &l.DatabaseConfigID, &l.DatabaseName, &l.StartedAt, &comp, &l.Status, &l.FileSizeBytes, &l.FilePath, &l.DurationSec, &errMsg); err != nil {
		return nil, err
	}
	if schedID.Valid {
		l.ScheduleID = schedID.String
	}
	if comp.Valid {
		l.CompletedAt = &comp.Time
	}
	if errMsg.Valid {
		l.ErrorMessage = errMsg.String
	}
	return &l, nil
}

func (s *SQLiteStorage) ListServerBackupLogs(serverID string, limit int) ([]model.BackupJobLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, schedule_id, server_id, database_config_id, database_name, started_at, completed_at, status, file_size_bytes, file_path, duration_sec, error_message
	FROM backup_logs`
	var rows *sql.Rows
	var err error
	if serverID != "" {
		rows, err = s.db.Query(query+` WHERE server_id = ? ORDER BY id DESC LIMIT ?`, serverID, limit)
	} else {
		rows, err = s.db.Query(query+` ORDER BY id DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.BackupJobLog
	for rows.Next() {
		var l model.BackupJobLog
		var comp sql.NullTime
		var schedID, errMsg sql.NullString
		if err := rows.Scan(&l.ID, &schedID, &l.ServerID, &l.DatabaseConfigID, &l.DatabaseName, &l.StartedAt, &comp, &l.Status, &l.FileSizeBytes, &l.FilePath, &l.DurationSec, &errMsg); err == nil {
			if schedID.Valid {
				l.ScheduleID = schedID.String
			}
			if comp.Valid {
				l.CompletedAt = &comp.Time
			}
			if errMsg.Valid {
				l.ErrorMessage = errMsg.String
			}
			logs = append(logs, l)
		}
	}
	return logs, nil
}

func (s *SQLiteStorage) ListBackupLogs(dbConfigID string, limit int) ([]model.BackupJobLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, schedule_id, server_id, database_config_id, database_name, started_at, completed_at, status, file_size_bytes, file_path, duration_sec, error_message
	FROM backup_logs`
	var rows *sql.Rows
	var err error
	if dbConfigID != "" {
		rows, err = s.db.Query(query+` WHERE database_config_id = ? ORDER BY id DESC LIMIT ?`, dbConfigID, limit)
	} else {
		rows, err = s.db.Query(query+` ORDER BY id DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.BackupJobLog
	for rows.Next() {
		var l model.BackupJobLog
		var comp sql.NullTime
		var schedID, errMsg sql.NullString
		if err := rows.Scan(&l.ID, &schedID, &l.ServerID, &l.DatabaseConfigID, &l.DatabaseName, &l.StartedAt, &comp, &l.Status, &l.FileSizeBytes, &l.FilePath, &l.DurationSec, &errMsg); err == nil {
			if schedID.Valid {
				l.ScheduleID = schedID.String
			}
			if comp.Valid {
				l.CompletedAt = &comp.Time
			}
			if errMsg.Valid {
				l.ErrorMessage = errMsg.String
			}
			logs = append(logs, l)
		}
	}
	return logs, nil
}

// ListActiveBackupLogs returns all backup jobs currently in "running" state
func (s *SQLiteStorage) ListActiveBackupLogs() ([]model.BackupJobLog, error) {
	query := `SELECT id, schedule_id, server_id, database_config_id, database_name, started_at, completed_at, status, file_size_bytes, file_path, duration_sec, error_message
	FROM backup_logs WHERE status = 'running' ORDER BY started_at DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.BackupJobLog
	for rows.Next() {
		var l model.BackupJobLog
		var comp sql.NullTime
		var schedID, errMsg sql.NullString
		if err := rows.Scan(&l.ID, &schedID, &l.ServerID, &l.DatabaseConfigID, &l.DatabaseName, &l.StartedAt, &comp, &l.Status, &l.FileSizeBytes, &l.FilePath, &l.DurationSec, &errMsg); err == nil {
			if schedID.Valid {
				l.ScheduleID = schedID.String
			}
			if comp.Valid {
				l.CompletedAt = &comp.Time
			}
			if errMsg.Valid {
				l.ErrorMessage = errMsg.String
			}
			logs = append(logs, l)
		}
	}
	return logs, nil
}

// CleanupStaleRunningBackupJobs updates any stale "running" jobs to "failed" when application starts
func (s *SQLiteStorage) CleanupStaleRunningBackupJobs() error {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE backup_logs SET status = 'failed', completed_at = ?, error_message = 'Proses terhenti karena aplikasi dimatikan sebelum selesai' WHERE status = 'running'`, now)
	return err
}

// Uptime Monitors (FR-14 & FR-15)

func (s *SQLiteStorage) SaveUptimeMonitor(m *model.UptimeMonitor) error {
	_, err := s.db.Exec(`INSERT INTO uptime_monitors
	(id, name, type, target, server_id, interval_sec, enabled, status, status_code, latency_ms, ssl_expiry_days, ssl_issuer, ssl_valid_until, last_error, last_check, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name=excluded.name,
		type=excluded.type,
		target=excluded.target,
		server_id=excluded.server_id,
		interval_sec=excluded.interval_sec,
		enabled=excluded.enabled,
		status=excluded.status,
		status_code=excluded.status_code,
		latency_ms=excluded.latency_ms,
		ssl_expiry_days=excluded.ssl_expiry_days,
		ssl_issuer=excluded.ssl_issuer,
		ssl_valid_until=excluded.ssl_valid_until,
		last_error=excluded.last_error,
		last_check=excluded.last_check`,
		m.ID, m.Name, m.Type, m.Target, m.ServerID, m.IntervalSec, m.Enabled, m.Status,
		m.StatusCode, m.LatencyMs, m.SSLExpiryDays, m.SSLIssuer, m.SSLValidUntil, m.LastError, m.LastCheck, m.CreatedAt)
	return err
}

func (s *SQLiteStorage) ListUptimeMonitors() ([]model.UptimeMonitor, error) {
	rows, err := s.db.Query(`SELECT id, name, type, target, server_id, interval_sec, enabled, status, status_code, latency_ms, ssl_expiry_days, ssl_issuer, ssl_valid_until, last_error, last_check, created_at
	FROM uptime_monitors ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.UptimeMonitor
	for rows.Next() {
		var m model.UptimeMonitor
		var lastCheck sql.NullTime
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.Target, &m.ServerID, &m.IntervalSec, &m.Enabled, &m.Status, &m.StatusCode, &m.LatencyMs, &m.SSLExpiryDays, &m.SSLIssuer, &m.SSLValidUntil, &m.LastError, &lastCheck, &m.CreatedAt); err == nil {
			if lastCheck.Valid {
				m.LastCheck = &lastCheck.Time
			}
			list = append(list, m)
		}
	}
	return list, nil
}

func (s *SQLiteStorage) GetUptimeMonitor(id string) (*model.UptimeMonitor, error) {
	row := s.db.QueryRow(`SELECT id, name, type, target, server_id, interval_sec, enabled, status, status_code, latency_ms, ssl_expiry_days, ssl_issuer, ssl_valid_until, last_error, last_check, created_at
	FROM uptime_monitors WHERE id = ?`, id)
	var m model.UptimeMonitor
	var lastCheck sql.NullTime
	if err := row.Scan(&m.ID, &m.Name, &m.Type, &m.Target, &m.ServerID, &m.IntervalSec, &m.Enabled, &m.Status, &m.StatusCode, &m.LatencyMs, &m.SSLExpiryDays, &m.SSLIssuer, &m.SSLValidUntil, &m.LastError, &lastCheck, &m.CreatedAt); err != nil {
		return nil, err
	}
	if lastCheck.Valid {
		m.LastCheck = &lastCheck.Time
	}
	return &m, nil
}

func (s *SQLiteStorage) DeleteUptimeMonitor(id string) error {
	_, err := s.db.Exec(`DELETE FROM uptime_monitors WHERE id = ?`, id)
	return err
}

// App Settings (FR-25, FR-29, FR-32, FR-33)

func (s *SQLiteStorage) GetAppSettings() model.AppSettings {
	defaults := model.AppSettings{
		RetentionDays:      30,
		AutoStartOS:        false,
		RunInBackground:    true,
		ConfirmSSHCommands: true,
		Notifications: model.NotificationSettings{
			DesktopEnabled:   true,
			TelegramEnabled:  false,
			WebhookEnabled:   false,
			NotifyOnRecovery: true,
			NotifyOnBackup:   true,
		},
	}

	var raw string
	err := s.db.QueryRow(`SELECT value_json FROM app_settings WHERE key = 'global'`).Scan(&raw)
	if err != nil || raw == "" {
		return defaults
	}
	var cfg model.AppSettings
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return defaults
	}
	if !strings.Contains(raw, "confirmSshCommands") {
		cfg.ConfirmSSHCommands = true
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 30
	}
	// Never return secrets from SQLite (SEC-08)
	cfg.Notifications.TelegramBotToken = ""
	cfg.Notifications.WebhookURL = ""
	cfg.S3Backup.SecretAccessKey = ""
	if cfg.S3Backup.Region == "" {
		cfg.S3Backup.Region = "auto"
	}
	if cfg.S3Backup.PathPrefix == "" {
		cfg.S3Backup.PathPrefix = "serverwatch-backups"
	}
	return cfg
}

func (s *SQLiteStorage) SaveAppSettings(cfg model.AppSettings) error {
	// Strip secrets before persisting to SQLite (SEC-08)
	cfg.Notifications.TelegramBotToken = ""
	cfg.Notifications.WebhookURL = ""
	cfg.S3Backup.SecretAccessKey = ""
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 30
	}

	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO app_settings (key, value_json) VALUES ('global', ?)
	ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json`, string(b))
	return err
}



