package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kevinburke/ssh_config"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"serverwatch/internal/backup"
	"serverwatch/internal/collector"
	"serverwatch/internal/dbmonitor"
	"serverwatch/internal/devops"
	"serverwatch/internal/model"
	"serverwatch/internal/notifier"
	"serverwatch/internal/report"
	"serverwatch/internal/sshpool"
	"serverwatch/internal/storage"
	"serverwatch/internal/terminal"
	"serverwatch/internal/uptime"
)

// App struct
type App struct {
	ctx             context.Context
	storage         *storage.SQLiteStorage
	keychain        *storage.KeychainManager
	sshPool         *sshpool.SSHPool
	collector       *collector.Collector
	dbCollector     *dbmonitor.DBCollector
	dbDetector      *dbmonitor.DBDetector
	devopsInspector *devops.Inspector
	notifier        *notifier.Notifier
	uptimeChecker   *uptime.Checker
	reportExporter  *report.Exporter
	backupEngine     *backup.BackupEngine
	backupScheduler  *backup.BackupScheduler
	termManager     *terminal.Manager
	stopPoll         chan struct{}
	pollWg           sync.WaitGroup
	scanMu           sync.Mutex
	dbScanCancel     context.CancelFunc
	devopsScanCancel context.CancelFunc
}

// NewApp creates a new App application struct
func NewApp() *App {
	store, err := storage.NewSQLiteStorage("")
	if err != nil {
		fmt.Printf("Warning: failed to open SQLite storage: %v\n", err)
	}
	km := storage.NewKeychainManager()
	pool := sshpool.NewSSHPool(store, km)
	col := collector.NewCollector(pool, store)
	dbCol := dbmonitor.NewDBCollector(pool)
	dbDet := dbmonitor.NewDBDetector(pool)
	devIns := devops.NewInspector(pool)
	notif := notifier.NewNotifier(store, km)
	upCheck := uptime.NewChecker(pool, store)
	repExp := report.NewExporter(store)
	bEngine := backup.NewBackupEngine(pool, store, km)
	bSched := backup.NewBackupScheduler(bEngine, store)
	termMgr := terminal.NewManager(pool, store)

	app := &App{
		storage:         store,
		keychain:        km,
		sshPool:         pool,
		collector:       col,
		dbCollector:     dbCol,
		dbDetector:      dbDet,
		devopsInspector: devIns,
		notifier:        notif,
		uptimeChecker:   upCheck,
		reportExporter:  repExp,
		backupEngine:    bEngine,
		backupScheduler: bSched,
		termManager:     termMgr,
		stopPoll:        make(chan struct{}),
	}

	col.SetAlertCallback(func(serverName, level, message string, isRecovery bool) {
		if app.ctx != nil {
			notif.DispatchAlert(app.ctx, serverName, level, message, isRecovery)
		}
	})
	bEngine.SetProgressCallback(func(progress model.BackupProgressEvent) {
		if app.ctx != nil {
			runtime.EventsEmit(app.ctx, "backup:progress", progress)
		}
	})
	upCheck.SetAlertCallback(func(monitorName, level, message string, isRecovery bool) {
		if app.ctx != nil {
			notif.DispatchAlert(app.ctx, monitorName, level, message, isRecovery)
		}
	})
	notif.SetDesktopCallback(func(title, body, level string) {
		if app.ctx != nil {
			runtime.EventsEmit(app.ctx, "alert:notification", map[string]string{
				"title": title,
				"body":  body,
				"level": level,
				"time":  time.Now().Format("15:04:05"),
			})
		}
	})

	return app
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.storage != nil {
		_ = a.storage.CleanupStaleRunningBackupJobs()
	}
	a.startPoller()
	if a.backupScheduler != nil {
		a.backupScheduler.Start(ctx)
	}
}

// beforeClose handles background monitoring preference when user closes window (FR-32)
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.storage == nil {
		return false
	}
	cfg := a.storage.GetAppSettings()
	if !cfg.RunInBackground {
		return false
	}
	choice, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "ServerWatch Background Monitoring",
		Message:       "Pilih 'Yes' untuk meminimalkan jendela dan tetap memonitor server di latar belakang, atau 'No' untuk keluar sepenuhnya dari aplikasi.",
		Buttons:       []string{"Yes", "No"},
		DefaultButton: "Yes",
		CancelButton:  "No",
	})
	if err == nil && (choice == "Yes" || choice == "Ya" || choice == "OK") {
		runtime.WindowMinimise(ctx)
		return true
	}
	return false
}

// shutdown cleans up active connections
func (a *App) shutdown(ctx context.Context) {
	close(a.stopPoll)
	a.pollWg.Wait()
	if a.backupScheduler != nil {
		a.backupScheduler.Stop()
	}
	if a.termManager != nil {
		a.termManager.CloseAll()
	}
	if a.sshPool != nil {
		a.sshPool.Close()
	}
	if a.storage != nil {
		_ = a.storage.Close()
	}
}

// startPoller schedules periodic metrics collection
func (a *App) startPoller() {
	if a.storage == nil {
		return
	}

	a.pollWg.Add(1)
	go func() {
		defer a.pollWg.Done()
		a.pollActiveServers()
		a.pollActiveDatabases()
		a.pollActiveUptimeMonitors()

		settings := a.storage.GetAppSettings()
		_ = a.storage.CleanOldMetrics(settings.RetentionDays)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-a.stopPoll:
				return
			case <-ticker.C:
				a.pollActiveServers()
				a.pollActiveDatabases()
				a.pollActiveUptimeMonitors()
			}
		}
	}()
}

func (a *App) pollActiveUptimeMonitors() {
	if a.storage == nil || a.uptimeChecker == nil {
		return
	}
	monitors, err := a.storage.ListUptimeMonitors()
	if err != nil {
		return
	}
	now := time.Now()
	for _, mon := range monitors {
		if !mon.Enabled {
			continue
		}
		interval := mon.IntervalSec
		if interval < 10 {
			interval = 60
		}
		if mon.LastCheck != nil && now.Sub(*mon.LastCheck) < time.Duration(interval)*time.Second {
			continue
		}
		go func(m model.UptimeMonitor) {
			updated, err := a.uptimeChecker.CheckMonitor(a.ctx, &m)
			if err == nil && updated != nil && a.ctx != nil {
				runtime.EventsEmit(a.ctx, "uptime:update", updated)
			}
		}(mon)
	}
}

func (a *App) pollActiveServers() {
	servers, err := a.storage.ListServers()
	if err != nil {
		return
	}

	now := time.Now()
	for _, srv := range servers {
		interval := srv.IntervalSec
		if interval < 5 {
			interval = 10
		}
		if srv.LastCheck != nil && now.Sub(*srv.LastCheck) < time.Duration(interval)*time.Second {
			continue
		}

		go func(s model.Server) {
			metrics, err := a.collector.CollectServerMetrics(a.ctx, &s)
			if err != nil {
				runtime.EventsEmit(a.ctx, "server:error", s.ID, err.Error())
			} else if metrics != nil {
				runtime.EventsEmit(a.ctx, "metrics:update", s.ID, metrics)
			}
		}(srv)
	}
}

func (a *App) pollActiveDatabases() {
	if a.storage == nil || a.dbCollector == nil {
		return
	}

	servers, err := a.storage.ListServers()
	if err != nil {
		return
	}

	for _, srv := range servers {
		dbConfigs, err := a.storage.ListDatabaseConfigs(srv.ID)
		if err != nil {
			continue
		}
		for _, db := range dbConfigs {
			if !db.Enabled {
				continue
			}
			go func(dbID string) {
				_, _ = a.PollDatabaseMetrics(dbID)
			}(db.ID)
		}
	}
}

// GetServers returns all monitored servers
func (a *App) GetServers() ([]model.Server, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ListServers()
}

// GetServer returns a single server by ID
func (a *App) GetServer(id string) (*model.Server, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.GetServer(id)
}

// TestConnection tests SSH connectivity and returns server fingerprints & diagnostic info
func (a *App) TestConnection(cfg model.ServerConfig) (*model.ConnectionTestResult, error) {
	if a.sshPool == nil {
		return nil, fmt.Errorf("ssh pool not initialized")
	}
	return a.sshPool.TestConnection(&cfg)
}

// SaveServer adds or updates a server (storing secrets safely in OS Keychain)
func (a *App) SaveServer(cfg model.ServerConfig) (*model.Server, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	isNew := false
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
		isNew = true
	}

	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.IntervalSec <= 0 {
		cfg.IntervalSec = 10
	}

	// SEC-08: Save password or passphrase in OS Keychain
	if cfg.Password != "" {
		_ = a.keychain.SetPassword(cfg.ID, cfg.Password)
	}
	if cfg.Passphrase != "" {
		_ = a.keychain.SetPassphrase(cfg.ID, cfg.Passphrase)
	}

	now := time.Now()
	srv := &model.Server{
		ID:          cfg.ID,
		Name:        cfg.Name,
		Host:        cfg.Host,
		Port:        cfg.Port,
		User:        cfg.User,
		AuthType:    cfg.AuthType,
		KeyPath:     cfg.KeyPath,
		JumpHostID:  cfg.JumpHostID,
		Tags:        cfg.Tags,
		Notes:       cfg.Notes,
		Status:      model.StatusNormal,
		IntervalSec: cfg.IntervalSec,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if !isNew {
		if existing, err := a.storage.GetServer(cfg.ID); err == nil {
			srv.CreatedAt = existing.CreatedAt
			srv.HostKeyFingerprint = existing.HostKeyFingerprint
			srv.TrustedHostKey = existing.TrustedHostKey
			srv.LastCheck = existing.LastCheck
		}
		a.sshPool.RemoveClient(srv.ID)
	}

	if err := a.storage.SaveServer(srv); err != nil {
		return nil, err
	}

	// Trigger immediate poll in background
	go func() {
		m, err := a.collector.CollectServerMetrics(a.ctx, srv)
		if err != nil {
			runtime.EventsEmit(a.ctx, "server:error", srv.ID, err.Error())
		} else if m != nil {
			runtime.EventsEmit(a.ctx, "metrics:update", srv.ID, m)
		}
	}()

	return srv, nil
}

// TrustHostKey marks the server's fingerprint as verified and trusted
func (a *App) TrustHostKey(serverID string, fingerprint string) error {
	srv, err := a.storage.GetServer(serverID)
	if err != nil {
		return err
	}
	srv.HostKeyFingerprint = fingerprint
	srv.TrustedHostKey = true
	srv.UpdatedAt = time.Now()
	return a.storage.SaveServer(srv)
}

// DeleteServer removes server and evicts credentials from keychain
func (a *App) DeleteServer(id string) error {
	a.sshPool.RemoveClient(id)
	a.keychain.DeleteCredentials(id)
	return a.storage.DeleteServer(id)
}

// GetLatestMetrics fetches the most recent metric sample for a server
func (a *App) GetLatestMetrics(serverID string) (*model.ServerMetrics, error) {
	return a.storage.GetLatestMetrics(serverID)
}

// GetMetricsHistory retrieves time-series metrics over a given period (e.g. "1h", "24h", "7d", "30d")
func (a *App) GetMetricsHistory(serverID string, rangeStr string) ([]model.ServerMetrics, error) {
	duration := 1 * time.Hour
	switch rangeStr {
	case "15m":
		duration = 15 * time.Minute
	case "1h":
		duration = 1 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	}
	return a.storage.GetMetricsHistory(serverID, duration)
}

// ImportSSHConfig parses ~/.ssh/config and returns detected hosts (FR-04)
func (a *App) ImportSSHConfig() ([]model.ServerConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".ssh", "config")
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", configPath, err)
	}
	defer f.Close()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ssh config: %w", err)
	}

	var results []model.ServerConfig
	for _, host := range cfg.Hosts {
		for _, pattern := range host.Patterns {
			alias := pattern.String()
			if alias == "*" || strings.Contains(alias, "?") {
				continue
			}

			hostname, _ := cfg.Get(alias, "HostName")
			if hostname == "" {
				hostname = alias
			}
			user, _ := cfg.Get(alias, "User")
			portStr, _ := cfg.Get(alias, "Port")
			port := 22
			if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
				port = p
			}
			identityFile, _ := cfg.Get(alias, "IdentityFile")
			authType := model.AuthTypeKey
			if identityFile == "" {
				authType = model.AuthTypePassword
			} else {
				if strings.HasPrefix(identityFile, "~") {
					identityFile = filepath.Join(homeDir, strings.TrimPrefix(identityFile, "~"))
				}
			}

			results = append(results, model.ServerConfig{
				Name:        alias,
				Host:        hostname,
				Port:        port,
				User:        user,
				AuthType:    authType,
				KeyPath:     identityFile,
				IntervalSec: 10,
			})
		}
	}

	return results, nil
}

// GetAlertRules returns all configurable alert rules
func (a *App) GetAlertRules() ([]model.AlertRule, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ListAlertRules()
}

// ToggleAlertRule enables or disables an alert rule
func (a *App) ToggleAlertRule(id string, enabled bool) error {
	if a.storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	return a.storage.ToggleAlertRule(id, enabled)
}

// GetAlertEvents returns historical alert occurrences
func (a *App) GetAlertEvents(limit int) ([]model.AlertEvent, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ListAlertEvents(limit)
}

// GetActiveAlerts returns currently active alerts
func (a *App) GetActiveAlerts() ([]model.AlertEvent, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.GetActiveAlerts()
}

// TailLog reads journalctl or system logs from a remote server (FR-21)
func (a *App) TailLog(serverID string, target string) (string, error) {
	if a.storage == nil || a.sshPool == nil {
		return "", fmt.Errorf("system not initialized")
	}
	srv, err := a.storage.GetServer(serverID)
	if err != nil {
		return "", fmt.Errorf("server not found: %w", err)
	}

	if target == "" {
		target = "syslog"
	}
	target = strings.TrimSpace(target)
	if err := devops.ValidateUnitName(target); err != nil {
		return "", err
	}

	cmd := fmt.Sprintf("journalctl -u '%s' -n 60 --no-pager 2>/dev/null || cat '/var/log/%s' 2>/dev/null | tail -n 60 || dmesg 2>/dev/null | tail -n 60", target, target)
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	out, err := a.sshPool.ExecuteCommand(ctx, srv, cmd)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == "" {
		out = fmt.Sprintf("Tidak ada baris log yang ditemukan untuk target: %s", target)
	}
	return out, nil
}

// RunSSHCommand executes a command in the server and returns stdout/stderr (FR-20, SEC-04, SEC-06)
func (a *App) RunSSHCommand(serverID string, cmd string) (string, error) {
	if a.storage == nil || a.sshPool == nil {
		return "", fmt.Errorf("system not initialized")
	}
	srv, err := a.storage.GetServer(serverID)
	if err != nil {
		return "", fmt.Errorf("server not found: %w", err)
	}

	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", fmt.Errorf("perintah tidak boleh kosong")
	}

	// SEC-06: Native OS dialog confirmation prevents automated execution by malicious frontend scripts (configurable via AppSettings)
	cfg := a.storage.GetAppSettings()
	if cfg.ConfirmSSHCommands && a.ctx != nil {
		choice, dlgErr := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Konfirmasi Eksekusi Perintah SSH",
			Message:       fmt.Sprintf("Jalankan perintah berikut pada server '%s' (%s)?\n\nPerintah:\n%s", srv.Name, srv.Host, cmd),
			Buttons:       []string{"Yes", "No"},
			DefaultButton: "No",
			CancelButton:  "No",
		})
		if dlgErr != nil || (choice != "Yes" && choice != "Ya" && choice != "OK") {
			return "", fmt.Errorf("eksekusi dibatalkan oleh pengguna (SEC-06)")
		}
	}

	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()

	return a.sshPool.ExecuteCommand(ctx, srv, cmd)
}

// StartSSHTerminal starts an interactive SSH shell with PTY support for xterm.js
func (a *App) StartSSHTerminal(serverID string, cols int, rows int) (string, error) {
	if a.storage == nil || a.termManager == nil {
		return "", fmt.Errorf("system not initialized")
	}
	srv, err := a.storage.GetServer(serverID)
	if err != nil {
		return "", fmt.Errorf("server not found: %w", err)
	}

	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	sessionID, err := a.termManager.StartSession(
		baseCtx,
		srv,
		cols,
		rows,
		func(sid, chunk string) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "terminal:data:"+sid, chunk)
			}
		},
		func(sid string) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "terminal:exit:"+sid, nil)
			}
		},
	)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

// SendTerminalInput forwards user keystrokes/data from xterm.js to the remote PTY stdin
func (a *App) SendTerminalInput(sessionID string, data string) error {
	if a.termManager == nil {
		return fmt.Errorf("terminal manager not initialized")
	}
	return a.termManager.Write(sessionID, data)
}

// ResizeSSHTerminal notifies remote PTY of terminal window dimension changes
func (a *App) ResizeSSHTerminal(sessionID string, cols int, rows int) error {
	if a.termManager == nil {
		return nil
	}
	return a.termManager.Resize(sessionID, cols, rows)
}

// CloseSSHTerminal closes an interactive SSH shell session
func (a *App) CloseSSHTerminal(sessionID string) error {
	if a.termManager == nil {
		return nil
	}
	return a.termManager.CloseSession(sessionID)
}

// BroadcastSSHCommand executes a quick command concurrently across multiple servers (SEC-04, SEC-06)
func (a *App) BroadcastSSHCommand(serverIDs []string, cmd string) ([]model.BroadcastResult, error) {
	if a.storage == nil || a.sshPool == nil {
		return nil, fmt.Errorf("system not initialized")
	}
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil, fmt.Errorf("perintah broadcast tidak boleh kosong")
	}
	if len(serverIDs) == 0 {
		return nil, fmt.Errorf("pilih minimal satu server untuk broadcast")
	}

	// SEC-06: Native OS dialog confirmation prevents automated execution by malicious frontend scripts (configurable via AppSettings)
	cfg := a.storage.GetAppSettings()
	if cfg.ConfirmSSHCommands && a.ctx != nil {
		choice, dlgErr := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.WarningDialog,
			Title:         "Konfirmasi Broadcast Perintah SSH",
			Message:       fmt.Sprintf("PERINGATAN: Anda akan mengeksekusi perintah secara serentak ke %d server terdaftar!\n\nPerintah:\n%s\n\nLanjutkan broadcast?", len(serverIDs), cmd),
			Buttons:       []string{"Yes", "No"},
			DefaultButton: "No",
			CancelButton:  "No",
		})
		if dlgErr != nil || (choice != "Yes" && choice != "Ya" && choice != "OK") {
			return nil, fmt.Errorf("broadcast dibatalkan oleh pengguna (SEC-06)")
		}
	}

	results := make([]model.BroadcastResult, len(serverIDs))
	var wg sync.WaitGroup

	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	for i, sid := range serverIDs {
		wg.Add(1)
		go func(index int, sID string) {
			defer wg.Done()
			srv, err := a.storage.GetServer(sID)
			if err != nil {
				results[index] = model.BroadcastResult{
					ServerID: sID,
					Error:    fmt.Sprintf("server tidak ditemukan: %v", err),
				}
				return
			}

			start := time.Now()
			ctx, cancel := context.WithTimeout(baseCtx, 20*time.Second)
			defer cancel()

			out, execErr := a.sshPool.ExecuteCommand(ctx, srv, cmd)
			dur := time.Since(start).Milliseconds()

			res := model.BroadcastResult{
				ServerID:   srv.ID,
				ServerName: srv.Name,
				Host:       srv.Host,
				Tags:       srv.Tags,
				DurationMs: dur,
			}
			if execErr != nil {
				res.Error = execErr.Error()
				res.Output = out
				res.Success = false
			} else {
				res.Output = out
				res.Success = true
			}
			results[index] = res
		}(i, sid)
	}

	wg.Wait()
	return results, nil
}

// Database Monitoring Bindings (FR-35 to FR-37)

// AutoDetectDatabases automatically scans the remote server for running MySQL, PostgreSQL, or Redis instances
func (a *App) AutoDetectDatabases(serverID string) ([]model.DiscoveredDatabase, error) {
	if a.storage == nil || a.dbDetector == nil {
		return nil, fmt.Errorf("system not initialized")
	}

	srv, err := a.storage.GetServer(serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	a.scanMu.Lock()
	if a.dbScanCancel != nil {
		a.dbScanCancel()
	}
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(baseCtx)
	a.dbScanCancel = cancel
	a.scanMu.Unlock()
	defer cancel()

	return a.dbDetector.DetectDatabases(ctx, srv)
}

// CancelDatabaseScan cancels any in-flight database auto-detection scan.
func (a *App) CancelDatabaseScan() {
	a.scanMu.Lock()
	if a.dbScanCancel != nil {
		a.dbScanCancel()
		a.dbScanCancel = nil
	}
	a.scanMu.Unlock()
}

func (a *App) GetDatabaseConfigs(serverID string) ([]model.DatabaseConfig, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ListDatabaseConfigs(serverID)
}

func (a *App) SaveDatabaseConfig(cfg model.DatabaseConfig, password string) (*model.DatabaseConfig, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
		cfg.CreatedAt = time.Now()
	}
	cfg.UpdatedAt = time.Now()

	if cfg.Port <= 0 {
		switch cfg.Type {
		case model.DBTypeMySQL:
			cfg.Port = 3306
		case model.DBTypePostgres:
			cfg.Port = 5432
		case model.DBTypeRedis:
			cfg.Port = 6379
		}
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}

	// SEC-05 & SEC-16: Validate connection parameters strictly
	if err := dbmonitor.ValidateDBConfig(&cfg); err != nil {
		return nil, err
	}

	// SEC-15: Store DB password safely in OS Keychain
	if password != "" {
		_ = a.keychain.SetDBPassword(cfg.ID, password)
	}

	if err := a.storage.SaveDatabaseConfig(&cfg); err != nil {
		return nil, err
	}

	// Trigger initial metric collection
	go func() {
		_, _ = a.PollDatabaseMetrics(cfg.ID)
	}()

	return &cfg, nil
}

func (a *App) DeleteDatabaseConfig(id string) error {
	a.keychain.DeleteDBCredentials(id)
	return a.storage.DeleteDatabaseConfig(id)
}

func (a *App) GetDatabaseMetrics(dbConfigID string) (*model.DatabaseMetrics, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.GetLatestDatabaseMetrics(dbConfigID)
}

func (a *App) PollDatabaseMetrics(dbConfigID string) (*model.DatabaseMetrics, error) {
	if a.storage == nil || a.dbCollector == nil {
		return nil, fmt.Errorf("system not initialized")
	}

	cfg, err := a.storage.GetDatabaseConfig(dbConfigID)
	if err != nil {
		return nil, err
	}

	srv, err := a.storage.GetServer(cfg.ServerID)
	if err != nil {
		return nil, err
	}

	password, _ := a.keychain.GetDBPassword(dbConfigID)

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	metrics, err := a.dbCollector.CollectMetrics(ctx, srv, cfg, password)
	if metrics != nil {
		_ = a.storage.SaveDatabaseMetrics(metrics)
		runtime.EventsEmit(a.ctx, "dbmetrics:update", dbConfigID, metrics)
	}
	if err != nil {
		return metrics, err
	}

	return metrics, nil
}

// TestDatabaseConnection tests database connectivity and credentials without saving
func (a *App) TestDatabaseConnection(cfg model.DatabaseConfig, password string) (*model.DatabaseMetrics, error) {
	if a.storage == nil || a.dbCollector == nil {
		return nil, fmt.Errorf("system not initialized")
	}

	// SEC-05 & SEC-16: Validate connection parameters strictly before sending any command to remote server
	if err := dbmonitor.ValidateDBConfig(&cfg); err != nil {
		return nil, err
	}

	srv, err := a.storage.GetServer(cfg.ServerID)
	if err != nil {
		return nil, fmt.Errorf("server target tidak ditemukan: %w", err)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	return a.dbCollector.CollectMetrics(ctx, srv, &cfg, password)
}

// ListDatabases lists all schemas/databases inside a configured database instance
func (a *App) ListDatabases(dbConfigID string) ([]model.DatabaseSchemaItem, error) {
	if a.storage == nil || a.dbCollector == nil {
		return nil, fmt.Errorf("system not initialized")
	}

	cfg, err := a.storage.GetDatabaseConfig(dbConfigID)
	if err != nil {
		return nil, err
	}

	srv, err := a.storage.GetServer(cfg.ServerID)
	if err != nil {
		return nil, err
	}

	password, _ := a.keychain.GetDBPassword(dbConfigID)

	ctx, cancel := context.WithTimeout(a.ctx, 12*time.Second)
	defer cancel()

	return a.dbCollector.ListDatabases(ctx, srv, cfg, password)
}

// AnalyzeDatabaseQueries inspects active queries/sessions and slow query digests on the target database.
func (a *App) AnalyzeDatabaseQueries(dbConfigID string) (*model.DatabaseQueryAnalysis, error) {
	if a.storage == nil || a.dbCollector == nil {
		return nil, fmt.Errorf("system not initialized")
	}

	cfg, err := a.storage.GetDatabaseConfig(dbConfigID)
	if err != nil {
		return nil, err
	}

	srv, err := a.storage.GetServer(cfg.ServerID)
	if err != nil {
		return nil, err
	}

	password, _ := a.keychain.GetDBPassword(dbConfigID)

	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()

	return a.dbCollector.AnalyzeQueries(ctx, srv, cfg, password)
}

// ListDatabaseTables lists tables, row counts, data sizes, and index sizes inside a specific schema.
func (a *App) ListDatabaseTables(dbConfigID string, schemaName string) ([]model.DatabaseTableItem, error) {
	if a.storage == nil || a.dbCollector == nil {
		return nil, fmt.Errorf("system not initialized")
	}

	cfg, err := a.storage.GetDatabaseConfig(dbConfigID)
	if err != nil {
		return nil, err
	}

	srv, err := a.storage.GetServer(cfg.ServerID)
	if err != nil {
		return nil, err
	}

	password, _ := a.keychain.GetDBPassword(dbConfigID)

	ctx, cancel := context.WithTimeout(a.ctx, 12*time.Second)
	defer cancel()

	return a.dbCollector.ListTables(ctx, srv, cfg, password, schemaName)
}

// KillDatabaseQuery terminates a running database query after native OS confirmation (SEC-05 & SEC-06).
func (a *App) KillDatabaseQuery(dbConfigID string, pid string) error {
	if a.storage == nil || a.dbCollector == nil {
		return fmt.Errorf("system not initialized")
	}

	cfg, err := a.storage.GetDatabaseConfig(dbConfigID)
	if err != nil {
		return err
	}

	srv, err := a.storage.GetServer(cfg.ServerID)
	if err != nil {
		return err
	}

	if a.ctx != nil {
		choice, dlgErr := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Konfirmasi Hentikan Query Database",
			Message:       fmt.Sprintf("Apakah Anda yakin ingin menghentikan paksa (KILL) proses/query ID #%s pada database '%s' (%s)?", pid, cfg.Name, srv.Name),
			Buttons:       []string{"Yes", "No"},
			DefaultButton: "No",
			CancelButton:  "No",
		})
		if dlgErr != nil || (choice != "Yes" && choice != "Ya" && choice != "OK") {
			return fmt.Errorf("aksi dibatalkan oleh pengguna")
		}
	}

	password, _ := a.keychain.GetDBPassword(dbConfigID)

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	return a.dbCollector.KillQuery(ctx, srv, cfg, password, pid)
}

// Backup Engine Bindings (FR-38 to FR-44)

func (a *App) GetBackupSchedules(serverID string) ([]model.BackupSchedule, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ListBackupSchedules(serverID)
}

func (a *App) SaveBackupSchedule(sched model.BackupSchedule) (*model.BackupSchedule, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	if sched.ID == "" {
		sched.ID = uuid.New().String()
		sched.CreatedAt = time.Now()
	}
	if sched.RetentionDays <= 0 {
		sched.RetentionDays = 7
	}
	if sched.RemotePath == "" {
		sched.RemotePath = "/var/backups/serverwatch"
	}
	// SEC-05 & SEC-16: Validate backup path strictly to prevent command injection
	if err := backup.ValidateRemotePath(sched.RemotePath); err != nil {
		return nil, err
	}
	if sched.DatabaseName != "" && sched.DatabaseName != "all_databases" {
		if err := backup.ValidateIdentifier(sched.DatabaseName); err != nil {
			return nil, err
		}
	}
	if sched.StorageTarget == "" {
		sched.StorageTarget = "remote"
	}
	if sched.Compression == "" {
		sched.Compression = "gzip"
	}

	if err := a.storage.SaveBackupSchedule(&sched); err != nil {
		return nil, err
	}

	return &sched, nil
}

func (a *App) DeleteBackupSchedule(id string) error {
	if a.storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	return a.storage.DeleteBackupSchedule(id)
}

func (a *App) TriggerManualBackup(dbConfigID string, storageTarget string, specificDB string) (*model.BackupJobLog, error) {
	if a.storage == nil || a.backupEngine == nil {
		return nil, fmt.Errorf("system not initialized")
	}

	dbCfg, err := a.storage.GetDatabaseConfig(dbConfigID)
	if err != nil {
		return nil, fmt.Errorf("database config not found: %w", err)
	}

	srv, err := a.storage.GetServer(dbCfg.ServerID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	targetDB := dbCfg.DatabaseName
	if specificDB != "" {
		targetDB = specificDB
	}

	sched := &model.BackupSchedule{
		DatabaseConfigID: dbConfigID,
		ServerID:         srv.ID,
		DatabaseName:     targetDB,
		StorageTarget:    storageTarget,
		RemotePath:       "/var/backups/serverwatch",
		RetentionDays:    7,
	}

	initialJob := &model.BackupJobLog{
		ServerID:         srv.ID,
		DatabaseConfigID: dbCfg.ID,
		DatabaseName:     targetDB,
		StartedAt:        time.Now(),
		Status:           "running",
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		finalLog, err := a.backupEngine.ExecuteBackup(bgCtx, srv, dbCfg, sched)
		if a.notifier != nil && finalLog != nil {
			a.notifier.DispatchBackupStatus(a.ctx, srv.Name, finalLog)
		}
		if a.ctx != nil {
			if err != nil {
				runtime.EventsEmit(a.ctx, "backup:failed", finalLog)
			} else {
				runtime.EventsEmit(a.ctx, "backup:completed", finalLog)
			}
		}
	}()

	return initialJob, nil
}

// GetActiveBackupJobs returns all currently running database backup jobs
func (a *App) GetActiveBackupJobs() ([]model.BackupJobLog, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ListActiveBackupLogs()
}

func (a *App) GetBackupLogs(dbConfigID string, limit int) ([]model.BackupJobLog, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ListBackupLogs(dbConfigID, limit)
}

func (a *App) GetServerBackupLogs(serverID string, limit int) ([]model.BackupJobLog, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ListServerBackupLogs(serverID, limit)
}

// VerifyBackupArchive checks archive existence, gzip integrity, and SHA-256 hash on the remote server.
func (a *App) VerifyBackupArchive(backupLogID int) (*model.BackupVerifyResult, error) {
	if a.storage == nil || a.backupEngine == nil {
		return nil, fmt.Errorf("system not initialized")
	}
	logItem, err := a.storage.GetBackupLog(backupLogID)
	if err != nil {
		return nil, fmt.Errorf("log backup tidak ditemukan: %w", err)
	}
	srv, err := a.storage.GetServer(logItem.ServerID)
	if err != nil {
		return nil, fmt.Errorf("server tidak ditemukan: %w", err)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	return a.backupEngine.VerifyBackupArchive(ctx, srv, logItem)
}

// DownloadBackupArchive downloads a remote backup archive to the admin's local machine via SaveFileDialog.
func (a *App) DownloadBackupArchive(backupLogID int) (string, error) {
	if a.storage == nil || a.backupEngine == nil || a.ctx == nil {
		return "", fmt.Errorf("system not initialized")
	}
	logItem, err := a.storage.GetBackupLog(backupLogID)
	if err != nil {
		return "", fmt.Errorf("log backup tidak ditemukan: %w", err)
	}
	srv, err := a.storage.GetServer(logItem.ServerID)
	if err != nil {
		return "", fmt.Errorf("server tidak ditemukan: %w", err)
	}

	remotePath, err := backup.ExtractRemoteFilePath(logItem.FilePath)
	if err != nil {
		return "", err
	}
	parts := strings.Split(remotePath, "/")
	defaultName := parts[len(parts)-1]

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Unduh Arsip Backup Database",
		DefaultFilename: defaultName,
	})
	if err != nil || savePath == "" {
		return "", fmt.Errorf("unduhan dibatalkan")
	}

	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()

	if err := a.backupEngine.DownloadRemoteFile(ctx, srv, remotePath, savePath); err != nil {
		return "", fmt.Errorf("gagal mengunduh arsip backup: %w", err)
	}
	return savePath, nil
}

// SelectBackupFolder opens a native OS folder picker dialog to select destination directory for local backups.
func (a *App) SelectBackupFolder() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("system context not initialized")
	}
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Pilih Folder Penyimpanan Backup di Laptop",
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// UploadBackupToS3 uploads an existing remote backup archive to the configured S3/Cloudflare R2/MinIO bucket.
func (a *App) UploadBackupToS3(backupLogID int) (string, error) {
	if a.storage == nil || a.backupEngine == nil {
		return "", fmt.Errorf("system not initialized")
	}
	logItem, err := a.storage.GetBackupLog(backupLogID)
	if err != nil {
		return "", fmt.Errorf("log backup tidak ditemukan: %w", err)
	}
	srv, err := a.storage.GetServer(logItem.ServerID)
	if err != nil {
		return "", fmt.Errorf("server tidak ditemukan: %w", err)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()

	return a.backupEngine.UploadBackupLogToS3(ctx, srv, logItem)
}

// RestoreDatabaseFromBackup restores a backup archive into the target database after native OS confirmation (SEC-06, SEC-15, SEC-16).
func (a *App) RestoreDatabaseFromBackup(backupLogID int, targetDBName string) error {
	if a.storage == nil || a.backupEngine == nil {
		return fmt.Errorf("system not initialized")
	}
	logItem, err := a.storage.GetBackupLog(backupLogID)
	if err != nil {
		return fmt.Errorf("log backup tidak ditemukan: %w", err)
	}
	dbCfg, err := a.storage.GetDatabaseConfig(logItem.DatabaseConfigID)
	if err != nil {
		return fmt.Errorf("konfigurasi database tidak ditemukan: %w", err)
	}
	srv, err := a.storage.GetServer(logItem.ServerID)
	if err != nil {
		return fmt.Errorf("server tidak ditemukan: %w", err)
	}

	destDB := strings.TrimSpace(targetDBName)
	if destDB == "" {
		destDB = logItem.DatabaseName
	}

	// SEC-06: Require native OS confirmation dialog before executing database restore/rollback.
	if a.ctx != nil {
		choice, dlgErr := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.WarningDialog,
			Title:         "Konfirmasi Restore / Rollback Database",
			Message:       fmt.Sprintf("PERINGATAN KRITIS:\nAnda akan merestorasi arsip backup '%s' ke dalam database '%s' pada server %s (%s).\n\nData tabel yang ada di '%s' akan ditimpa. Lanjutkan proses restore sekarang?", logItem.DatabaseName, destDB, srv.Name, srv.Host, destDB),
			Buttons:       []string{"Yes", "No"},
			DefaultButton: "No",
			CancelButton:  "No",
		})
		if dlgErr != nil || (choice != "Yes" && choice != "Ya" && choice != "OK") {
			return fmt.Errorf("proses restore dibatalkan oleh pengguna")
		}
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Minute)
	defer cancel()

	return a.backupEngine.RestoreDatabase(ctx, srv, dbCfg, logItem, destDB)
}

// TestS3BackupConnection tests connectivity and credentials for S3 / Cloudflare R2 / MinIO offsite storage.
func (a *App) TestS3BackupConnection(cfg model.S3BackupSettings) error {
	if a.backupEngine == nil {
		return fmt.Errorf("backup engine not initialized")
	}
	secret := strings.TrimSpace(cfg.SecretAccessKey)
	if secret == "" && a.keychain != nil {
		secret, _ = a.keychain.GetNotificationSecret("s3_secret_key")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()

	return a.backupEngine.TestS3Connection(ctx, cfg, secret)
}

// GetDevOpsSnapshot inspects containers, systemd services, open ports, firewall, and SSH security on a target server.
func (a *App) GetDevOpsSnapshot(serverID string) (*model.DevOpsSnapshot, error) {
	if a.storage == nil || a.devopsInspector == nil {
		return nil, fmt.Errorf("system not initialized")
	}
	srv, err := a.storage.GetServer(serverID)
	if err != nil {
		return nil, fmt.Errorf("server tidak ditemukan: %w", err)
	}

	a.scanMu.Lock()
	if a.devopsScanCancel != nil {
		a.devopsScanCancel()
	}
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(baseCtx)
	a.devopsScanCancel = cancel
	a.scanMu.Unlock()
	defer cancel()

	return a.devopsInspector.InspectServer(ctx, srv)
}

// CancelDevOpsScan cancels any in-flight DevOps ecosystem scan.
func (a *App) CancelDevOpsScan() {
	a.scanMu.Lock()
	if a.devopsScanCancel != nil {
		a.devopsScanCancel()
		a.devopsScanCancel = nil
	}
	a.scanMu.Unlock()
}

// ManageDockerContainer starts, stops, or restarts a container after native OS confirmation (SEC-05 & SEC-06).
func (a *App) ManageDockerContainer(serverID string, containerName string, action string) error {
	if a.storage == nil || a.devopsInspector == nil {
		return fmt.Errorf("system not initialized")
	}
	if err := devops.ValidateUnitName(containerName); err != nil {
		return err
	}
	if err := devops.ValidateAction(action); err != nil {
		return err
	}

	srv, err := a.storage.GetServer(serverID)
	if err != nil {
		return fmt.Errorf("server tidak ditemukan: %w", err)
	}

	// SEC-06: Require native OS confirmation before mutating remote container state.
	if a.ctx != nil {
		choice, dlgErr := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Konfirmasi Aksi Container",
			Message:       fmt.Sprintf("Jalankan aksi '%s' pada container '%s' di server %s (%s)?", strings.ToUpper(action), containerName, srv.Name, srv.Host),
			Buttons:       []string{"Yes", "No"},
			DefaultButton: "No",
			CancelButton:  "No",
		})
		if dlgErr != nil || (choice != "Yes" && choice != "Ya" && choice != "OK") {
			return fmt.Errorf("aksi dibatalkan oleh pengguna")
		}
	}

	return a.devopsInspector.ManageContainer(a.ctx, srv, containerName, action)
}

// GetDockerContainerLogs retrieves the last 80 log lines of a container (SEC-05).
func (a *App) GetDockerContainerLogs(serverID string, containerName string) (string, error) {
	if a.storage == nil || a.devopsInspector == nil {
		return "", fmt.Errorf("system not initialized")
	}
	srv, err := a.storage.GetServer(serverID)
	if err != nil {
		return "", fmt.Errorf("server tidak ditemukan: %w", err)
	}
	return a.devopsInspector.GetContainerLogs(a.ctx, srv, containerName, 80)
}

// ManageSystemService starts, stops, or restarts a systemd service after native OS confirmation (FR-22, SEC-05 & SEC-06).
func (a *App) ManageSystemService(serverID string, serviceName string, action string) error {
	if a.storage == nil || a.devopsInspector == nil {
		return fmt.Errorf("system not initialized")
	}
	if err := devops.ValidateUnitName(serviceName); err != nil {
		return err
	}
	if err := devops.ValidateAction(action); err != nil {
		return err
	}

	srv, err := a.storage.GetServer(serverID)
	if err != nil {
		return fmt.Errorf("server tidak ditemukan: %w", err)
	}

	// SEC-06: Native OS dialog confirmation so frontend script cannot self-approve service restarts.
	if a.ctx != nil {
		choice, dlgErr := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Konfirmasi Manajemen Service",
			Message:       fmt.Sprintf("Jalankan 'systemctl %s %s' di server %s (%s)?", action, serviceName, srv.Name, srv.Host),
			Buttons:       []string{"Yes", "No"},
			DefaultButton: "No",
			CancelButton:  "No",
		})
		if dlgErr != nil || (choice != "Yes" && choice != "Ya" && choice != "OK") {
			return fmt.Errorf("aksi dibatalkan oleh pengguna")
		}
	}

	return a.devopsInspector.ManageService(a.ctx, srv, serviceName, action)
}

// Uptime & SSL Certificate Monitor Bindings (FR-14 & FR-15)

func (a *App) GetUptimeMonitors() ([]model.UptimeMonitor, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ListUptimeMonitors()
}

func (a *App) SaveUptimeMonitor(mon model.UptimeMonitor) (*model.UptimeMonitor, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	mon.Name = strings.TrimSpace(mon.Name)
	mon.Target = strings.TrimSpace(mon.Target)
	if mon.Name == "" {
		return nil, fmt.Errorf("nama endpoint monitor wajib diisi")
	}
	if mon.Type == "" {
		mon.Type = "http"
	}
	if err := uptime.ValidateMonitorTarget(mon.Type, mon.Target); err != nil {
		return nil, err
	}
	if mon.IntervalSec < 10 {
		mon.IntervalSec = 60
	}
	if mon.ID == "" {
		mon.ID = uuid.New().String()
		mon.CreatedAt = time.Now()
		mon.Status = "up"
	}

	if err := a.storage.SaveUptimeMonitor(&mon); err != nil {
		return nil, err
	}

	go func(m model.UptimeMonitor) {
		updated, err := a.uptimeChecker.CheckMonitor(a.ctx, &m)
		if err == nil && updated != nil && a.ctx != nil {
			runtime.EventsEmit(a.ctx, "uptime:update", updated)
		}
	}(mon)

	return &mon, nil
}

func (a *App) DeleteUptimeMonitor(id string) error {
	if a.storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	return a.storage.DeleteUptimeMonitor(id)
}

func (a *App) CheckUptimeMonitorNow(id string) (*model.UptimeMonitor, error) {
	if a.storage == nil || a.uptimeChecker == nil {
		return nil, fmt.Errorf("system not initialized")
	}
	mon, err := a.storage.GetUptimeMonitor(id)
	if err != nil {
		return nil, fmt.Errorf("monitor tidak ditemukan: %w", err)
	}
	updated, err := a.uptimeChecker.CheckMonitor(a.ctx, mon)
	if err != nil {
		return nil, err
	}
	if a.ctx != nil && updated != nil {
		runtime.EventsEmit(a.ctx, "uptime:update", updated)
	}
	return updated, nil
}

// OpenExternalURL opens an HTTP/HTTPS URL safely in the default OS browser (SEC-12).
func (a *App) OpenExternalURL(targetURL string) error {
	targetURL = strings.TrimSpace(targetURL)
	if !strings.HasPrefix(targetURL, "https://") && !strings.HasPrefix(targetURL, "http://") {
		return fmt.Errorf("hanya URL http:// atau https:// yang diizinkan (SEC-12)")
	}
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, targetURL)
	}
	return nil
}

// App Settings, Notifications, and Report Export Bindings (FR-25, FR-29, FR-31, FR-32, FR-33)

func (a *App) GetAppSettings() (model.AppSettings, error) {
	if a.storage == nil {
		return model.AppSettings{}, fmt.Errorf("storage not initialized")
	}
	cfg := a.storage.GetAppSettings()
	if a.keychain != nil {
		if tok, err := a.keychain.GetNotificationSecret("telegram_token"); err == nil && tok != "" {
			cfg.Notifications.HasTelegramToken = true
		}
		if wh, err := a.keychain.GetNotificationSecret("webhook_url"); err == nil && wh != "" {
			cfg.Notifications.HasWebhookURL = true
		}
		if s3sec, err := a.keychain.GetNotificationSecret("s3_secret_key"); err == nil && s3sec != "" {
			cfg.S3Backup.HasSecretKey = true
		}
	}
	return cfg, nil
}

func (a *App) SaveAppSettings(cfg model.AppSettings) error {
	if a.storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// SEC-08: Store notification tokens & S3 secret keys in OS Keychain, never in SQLite
	if a.keychain != nil {
		if strings.TrimSpace(cfg.Notifications.TelegramBotToken) != "" {
			_ = a.keychain.SetNotificationSecret("telegram_token", strings.TrimSpace(cfg.Notifications.TelegramBotToken))
			cfg.Notifications.HasTelegramToken = true
		}
		if strings.TrimSpace(cfg.Notifications.WebhookURL) != "" {
			_ = a.keychain.SetNotificationSecret("webhook_url", strings.TrimSpace(cfg.Notifications.WebhookURL))
			cfg.Notifications.HasWebhookURL = true
		}
		if strings.TrimSpace(cfg.S3Backup.SecretAccessKey) != "" {
			_ = a.keychain.SetNotificationSecret("s3_secret_key", strings.TrimSpace(cfg.S3Backup.SecretAccessKey))
			cfg.S3Backup.HasSecretKey = true
		}
	}

	// FR-33: Configure Windows Auto-Start registry entry
	_ = setOSAutoStart(cfg.AutoStartOS)

	return a.storage.SaveAppSettings(cfg)
}

func (a *App) TestNotificationChannel(channel string, cfg model.NotificationSettings) error {
	if a.notifier == nil {
		return fmt.Errorf("notifier not initialized")
	}
	testMsg := fmt.Sprintf("[TES NOTIFIKASI SERVERWATCH]\nChannel '%s' berhasil terhubung pada %s.", strings.ToUpper(channel), time.Now().Format("2006-01-02 15:04:05"))

	switch channel {
	case "telegram":
		tok := strings.TrimSpace(cfg.TelegramBotToken)
		if tok == "" && a.keychain != nil {
			tok, _ = a.keychain.GetNotificationSecret("telegram_token")
		}
		return a.notifier.SendTelegram(a.ctx, tok, cfg.TelegramChatID, testMsg)
	case "webhook":
		urlStr := strings.TrimSpace(cfg.WebhookURL)
		if urlStr == "" && a.keychain != nil {
			urlStr, _ = a.keychain.GetNotificationSecret("webhook_url")
		}
		return a.notifier.SendWebhook(a.ctx, urlStr, testMsg)
	case "desktop":
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "alert:notification", map[string]string{
				"title": "Tes Notifikasi Desktop ServerWatch",
				"body":  "Notifikasi real-time berhasil ditampilkan.",
				"level": "ok",
				"time":  time.Now().Format("15:04:05"),
			})
		}
		return nil
	default:
		return fmt.Errorf("channel '%s' tidak dikenal", channel)
	}
}

// ExportReport exports server metrics, uptime, and alert logs to CSV or printable HTML/PDF report (FR-31).
func (a *App) ExportReport(format string) (string, error) {
	if a.reportExporter == nil || a.ctx == nil {
		return "", fmt.Errorf("exporter not initialized")
	}

	format = strings.ToLower(strings.TrimSpace(format))
	var defaultFilename string
	var filters []runtime.FileFilter
	var data []byte
	var err error

	dateTag := time.Now().Format("20060102_1504")
	if format == "csv" {
		defaultFilename = fmt.Sprintf("ServerWatch_Report_%s.csv", dateTag)
		filters = []runtime.FileFilter{{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"}}
		data, err = a.reportExporter.GenerateCSV()
	} else {
		defaultFilename = fmt.Sprintf("ServerWatch_Executive_Report_%s.html", dateTag)
		filters = []runtime.FileFilter{{DisplayName: "HTML / Printable PDF Report (*.html)", Pattern: "*.html"}}
		data, err = a.reportExporter.GenerateHTML()
	}
	if err != nil {
		return "", err
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Simpan Laporan ServerWatch",
		DefaultFilename: defaultFilename,
		Filters:         filters,
	})
	if err != nil || savePath == "" {
		return "", fmt.Errorf("ekspor dibatalkan")
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("gagal menulis file laporan: %w", err)
	}

	return savePath, nil
}

// CleanOldMetricsNow manually triggers historical data cleanup according to configured retention days (FR-29).
func (a *App) CleanOldMetricsNow() error {
	if a.storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	cfg := a.storage.GetAppSettings()
	return a.storage.CleanOldMetrics(cfg.RetentionDays)
}

func setOSAutoStart(enabled bool) error {
	if goruntime.GOOS != "windows" {
		return nil
	}
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	regKey := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	if enabled {
		cmd := exec.Command("reg", "add", regKey, "/v", "ServerWatch", "/t", "REG_SZ", "/d", fmt.Sprintf(`"%s"`, exePath), "/f")
		return cmd.Run()
	}
	cmd := exec.Command("reg", "delete", regKey, "/v", "ServerWatch", "/f")
	_ = cmd.Run()
	return nil
}




