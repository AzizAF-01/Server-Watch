# Graph Report - Monitoring-server  (2026-11-16)

## Corpus Check
- 63 files · ~108,314 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 983 nodes · 1619 edges · 92 communities (40 shown, 45 thin omitted)
- Extraction: 96% EXTRACTED · 4% INFERRED · 0% AMBIGUOUS · INFERRED: 60 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `22960d91`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- App.js
- types.go
- context.Context
- App
- SSHPool
- .convertValues
- ServerWatch Product Requirements Document v0.4
- devDependencies
- KeychainManager
- testing.T
- SQLiteStorage
- App.svelte
- runtime/package.json
- UptimeMonitor
- s3.go
- compilerOptions
- BackupScheduler
- wails.json
- BackupJobLog
- Server Detail View Mockup
- Add Server Modal Dialog
- Terminal and Log Viewer Mockup UI
- runtime.d.ts
- Dashboard Mockup UI
- Alert Management Mockup UI
- compilerOptions
- Wails Universal Logo Graphic
- .Close
- ServerStatus
- Exporter
- WCAG AA Contrast & Keyboard Focus Remediation
- chart.js
- layout.js
- ActiveSSHSession
- BackupVerifyResult
- BroadcastResult
- ConnectionTestResult
- CPUStats
- DatabaseActiveQuery
- DatabaseSchemaItem
- DatabaseSlowQuery
- Graph Report - Monitoring-server  (2026-11-15)
- DiscoveredDatabase
- DiskPartition
- FirewallStatus
- ListeningPort
- MemStats
- NetInterface
- loadDatabaseData
- ProcessItem
- models.ts
- ServerConfig
- ServiceItem
- ServerWatch: Design System & Visual Direction (`DESIGN.md`)
- loadServers
- EventsOnMultiple
- AlertRule
- Communities (81 total, 41 thin omitted)
- CI Test & Build Validation Workflow
- Frontend Vite + Svelte Architecture Notes
- Nunito Typography SIL Open Font License 1.1
- serverwatch
- connectPtyTerminal
- 🚨 Siklus Hidup Standar Pengerjaan Tugas (Issue ➔ Branch ➔ PR ➔ Merge ➔ Hapus Branch)
- DatabaseType
- loadAlerts
- loadQueryAnalysis
- AlertEvent
- AlertEvent
- BackupSchedule
- AppSettings
- DatabaseConfig
- Server
- pull_request_template.md
- DatabaseMetrics
- DevOpsSnapshot
- DatabaseTableItem
- executeBroadcast
- executeSSHCommand
- fetchLog
- handleDownloadBackup
- handleImportSSHConfig
- handleInspectTables
- handleManualBackup
- handleTestDBConnection

## God Nodes (most connected - your core abstractions)
1. `App` - 86 edges
2. `SQLiteStorage` - 63 edges
3. `Server` - 38 edges
4. `Communities (81 total, 41 thin omitted)` - 33 edges
5. `SSHPool` - 30 edges
6. `DatabaseConfig` - 25 edges
7. `KeychainManager` - 21 edges
8. `BackupEngine` - 17 edges
9. `NewApp()` - 16 edges
10. `BackupJobLog` - 16 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `NewApp()`  [INFERRED]
  main.go → app.go
- `App` --references--> `BackupEngine`  [EXTRACTED]
  app.go → internal/backup/engine.go
- `App` --references--> `BackupScheduler`  [EXTRACTED]
  app.go → internal/backup/scheduler.go
- `App` --references--> `Collector`  [EXTRACTED]
  app.go → internal/collector/collector.go
- `App` --references--> `DBCollector`  [EXTRACTED]
  app.go → internal/dbmonitor/collector.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **CI/CD Build, Provenance Attestation & Release Gate** — _github_workflows_release_release_workflow, _github_workflows_release_build_job, _github_workflows_release_publish_job, _github_workflows_release_artifact_attestation [EXTRACTED 1.00]
- **Anti-Slop Accessibility & Design System Standardization** — anti_slop_audit_001_2026_11_05_wcag_contrast_remediation, anti_slop_audit_002_2026_11_07_aria_semantics_remediation, changelog_security_hardening [INFERRED 0.85]
- **Agentless SSH Security Boundary & Trust Architecture** — docs_prd_threat_model_trust_boundary, docs_prd_wails_binding_security, docs_security_audit_report_vuln_01_command_dialog, docs_mockups_src_05_terminal_log_xss_protection_demo [INFERRED 0.95]

## Communities (92 total, 45 thin omitted)

### Community 1 - "App.js"
Cohesion: 0.07
Nodes (8): handleAutoDetect(), handleBrowseLocalFolder(), handleTestConnection(), handleViewSchemas(), AutoDetectDatabases(), ListDatabases(), SelectBackupFolder(), TestConnection()

### Community 2 - "types.go"
Cohesion: 0.08
Nodes (42): cpuTickSample, netTickSample, prevSample, containerStat, sync.Map, time.Time, calcCPUUsage(), Collector (+34 more)

### Community 3 - "context.Context"
Cohesion: 0.10
Nodes (26): BackupProgressCallback, context.Context, ExtractRemoteFilePath(), BackupEngine, NewBackupEngine(), TestValidateBackupInputs(), ValidateIdentifier(), ValidateRemotePath() (+18 more)

### Community 4 - "App"
Cohesion: 0.06
Nodes (9): setOSAutoStart(), context.CancelFunc, ValidateAction(), AppSettings, BackupVerifyResult, BroadcastResult, NotificationSettings, App (+1 more)

### Community 5 - "SSHPool"
Cohesion: 0.06
Nodes (30): NewApp(), bytes.Buffer, golang.org/x/crypto/ssh.AuthMethod, golang.org/x/crypto/ssh.Client, golang.org/x/crypto/ssh.ClientConfig, golang.org/x/crypto/ssh.Session, io.WriteCloser, io.Writer (+22 more)

### Community 6 - ".convertValues"
Cohesion: 0.13
Nodes (5): BackupJobLog, DatabaseQueryAnalysis, SecurityAudit, ServerMetrics, UptimeMonitor

### Community 7 - "ServerWatch Product Requirements Document v0.4"
Cohesion: 0.05
Nodes (45): Release Approval Protected Environment, Cryptographic Build Provenance & Attestation, Build & Verify (Windows AMD64) Job, Manual Approval & Publish Release Job, Release ServerWatch Workflow, ServerWatch Project Changelog, v1.0.0 Security Hardening & Remediation, ServerWatch v1.0.0 Release (+37 more)

### Community 8 - "devDependencies"
Cohesion: 0.06
Nodes (31): dependencies, @xterm/addon-fit, @xterm/xterm, devDependencies, svelte, svelte-check, svelte-preprocess, @sveltejs/vite-plugin-svelte (+23 more)

### Community 9 - "KeychainManager"
Cohesion: 0.10
Nodes (6): net/http.Client, Notifier, NewNotifier(), TestSendTelegramAndWebhook(), KeychainManager, NewKeychainManager()

### Community 10 - "testing.T"
Cohesion: 0.12
Nodes (19): testing.T, TestBackupSchedulerIsDue(), TestRestoreDatabaseValidation(), TestParseMySQLQueryAnalysis(), TestParseMySQLStatus_Mysqladmin(), TestParseMySQLStatus_TSV(), TestParseRedisQueryAnalysisAndTables(), TestValidateDBConfig() (+11 more)

### Community 11 - "SQLiteStorage"
Cohesion: 0.13
Nodes (3): database/sql.DB, SQLiteStorage, NewSQLiteStorage()

### Community 12 - "App.svelte"
Cohesion: 0.08
Nodes (11): applyTheme(), handleCancelAutoDetect(), handleDBServerChange(), handleScheduleSchema(), handleVerifyBackup(), openAddScheduleModal(), openRestoreWizard(), toggleTheme() (+3 more)

### Community 13 - "runtime/package.json"
Cohesion: 0.11
Nodes (18): author, bugs, url, description, homepage, keywords, license, main (+10 more)

### Community 14 - "UptimeMonitor"
Cohesion: 0.16
Nodes (6): UptimeMonitor, Checker, NewChecker(), TestCheckLocalHTTPAndTCP(), TestValidateMonitorTarget(), ValidateMonitorTarget()

### Community 15 - "s3.go"
Cohesion: 0.22
Nodes (12): S3Client, net/http.Request, net/url.Values, canonicalQueryString(), hmacSHA256(), NewS3Client(), sha256Hex(), signS3Request() (+4 more)

### Community 16 - "compilerOptions"
Cohesion: 0.11
Nodes (17): compilerOptions, allowJs, baseUrl, checkJs, isolatedModules, module, resolveJsonModule, target (+9 more)

### Community 17 - "BackupScheduler"
Cohesion: 0.36
Nodes (3): sync.WaitGroup, BackupScheduler, NewBackupScheduler()

### Community 18 - "wails.json"
Cohesion: 0.12
Nodes (16): author, email, name, frontend:build, frontend:dev:serverUrl, frontend:dev:watcher, frontend:install, info (+8 more)

### Community 20 - "Server Detail View Mockup"
Cohesion: 0.27
Nodes (11): Server Detail View Mockup, Server Action Controls and Time Range Selector, Active Server Alerts and Notification Channels, CPU Utilization Line Chart, Disk Partition Utilization and Predictive Forecasting, Server Host Metadata and SSH Configuration, Memory and Swap Usage Line Chart, Network Traffic Throughput Monitor (+3 more)

### Community 21 - "Add Server Modal Dialog"
Cohesion: 0.33
Nodes (9): Add Server Modal Mockup UI, SSH Authentication Method Selector, Jump Host and Bastion Proxy Routing, Server Connection Configuration Fields, Connection Test and Save Actions, Credential Security and Storage Architecture, Host Key Verification and Server Fingerprint, Add Server Modal Dialog (+1 more)

### Community 22 - "Terminal and Log Viewer Mockup UI"
Cohesion: 0.39
Nodes (9): Terminal and Log Viewer Mockup UI, Bastion Host SSH Tunneling Integration, Disk Space Troubleshooting Workflow, SSH Interactive Terminal Console, Unit and Keyword Log Filtering, Real-time Systemd Journal Log Viewer, Terminal Multi-Session Tabs, Terminal and Log Split Screen Layout (+1 more)

### Community 23 - "runtime.d.ts"
Cohesion: 0.25
Nodes (7): EnvironmentInfo, NotificationAction, NotificationCategory, NotificationOptions, Position, Screen, Size

### Community 24 - "Dashboard Mockup UI"
Cohesion: 0.48
Nodes (7): Dashboard Mockup UI, Attention Cards Section, Header Search and Server Actions, Server Inventory and Metrics Table, Sidebar Navigation, SSH Connection and Bastion Routing, Status and Tag Filtering System

### Community 25 - "Alert Management Mockup UI"
Cohesion: 0.52
Nodes (7): Alert Management Mockup UI, Recent Incidents and Event Feed, Sidebar Navigation and Active Alert Status, Notification Channels Configuration, Alert Rules Configuration and Thresholds, Desktop Execution Constraint and Headless Monitoring, Alert Snooze Management

### Community 26 - "compilerOptions"
Cohesion: 0.29
Nodes (6): compilerOptions, composite, module, moduleResolution, include, vite.config.ts

### Community 27 - "Wails Universal Logo Graphic"
Cohesion: 0.60
Nodes (5): Wails Universal Logo Graphic, Geometric Origami Eagle Emblem, Default Application Starter Asset, Wails Framework Branding, Wails Red Capitalized Typography Wordmark

### Community 31 - "WCAG AA Contrast & Keyboard Focus Remediation"
Cohesion: 0.50
Nodes (4): Anti-Slop UI & Code Audit 001 Report, WCAG AA Contrast & Keyboard Focus Remediation, ARIA Semantics & Design Token Standardization, Anti-Slop UI & Accessibility Audit 002 Report

### Community 33 - "layout.js"
Cohesion: 0.67
Nodes (3): icon(), ICONS, renderShell()

### Community 42 - "Graph Report - Monitoring-server  (2026-11-15)"
Cohesion: 0.18
Nodes (11): Community Hubs (Navigation), Corpus Check, God Nodes (most connected - your core abstractions), Graph Freshness, Graph Report - Monitoring-server  (2026-11-15), Hyperedges (group relationships), Import Cycles, Knowledge Gaps (+3 more)

### Community 49 - "loadDatabaseData"
Cohesion: 0.10
Nodes (20): handleAddDetectedDB(), handleDeleteDatabase(), handleDeleteSchedule(), handleExecuteRestore(), handlePollDBMetrics(), handleSaveDatabase(), handleSaveSchedule(), handleUploadBackupToS3() (+12 more)

### Community 51 - "models.ts"
Cohesion: 0.10
Nodes (6): AlertRule, DockerContainer, model, S3BackupSettings, SSHAttackerIP, SystemdService

### Community 54 - "ServerWatch: Design System & Visual Direction (`DESIGN.md`)"
Cohesion: 0.22
Nodes (8): 1. Identity & Personality, 2. Liveliness Dials (`ENERGY 1 / RHYTHM 2 / MOTION 1`), 3. Dual Theme Palette (R-21, R-25, R-29, R-34), 4. Typography & Identity Motifs (R-06, R-20), 5. Written Purpose Log (R-31: One-Line Reason per Decision), ServerWatch: Design System & Visual Direction (`DESIGN.md`), Theme A: Anodized Dark (`data-theme="dark"`, Default), Theme B: Brushed Aluminum Light (`data-theme="light"`)

### Community 55 - "loadServers"
Cohesion: 0.25
Nodes (9): executeImport(), handleDeleteServer(), handleSaveServer(), loadServers(), DeleteServer(), GetLatestMetrics(), GetServers(), SaveServer() (+1 more)

### Community 56 - "EventsOnMultiple"
Cohesion: 0.67
Nodes (3): EventsOn(), EventsOnce(), EventsOnMultiple()

### Community 58 - "Communities (81 total, 41 thin omitted)"
Cohesion: 0.06
Nodes (33): Communities (81 total, 41 thin omitted), Community 10 - "testing.T", Community 11 - "SQLiteStorage", Community 12 - "models.ts", Community 13 - "runtime/package.json", Community 14 - "UptimeMonitor", Community 15 - "s3.go", Community 16 - "compilerOptions" (+25 more)

### Community 69 - "connectPtyTerminal"
Cohesion: 0.29
Nodes (7): connectPtyTerminal(), disconnectPtyTerminal(), sendPtyInput(), CloseSSHTerminal(), ResizeSSHTerminal(), SendTerminalInput(), StartSSHTerminal()

### Community 70 - "🚨 Siklus Hidup Standar Pengerjaan Tugas (Issue ➔ Branch ➔ PR ➔ Merge ➔ Hapus Branch)"
Cohesion: 0.17
Nodes (10): antislop, 🗺️ Aturan Pemeliharaan Knowledge Graph (Graphify), Langkah 1: Wajib Buat Issue Terlebih Dahulu, Langkah 2: Buat Branch Baru untuk Pengerjaan, Langkah 3: Koding & Pengujian Mandiri, Langkah 4: Buka Pull Request (PR), Langkah 5: Merge & Wajib Hapus Branch, ServerWatch - AI Agent Guidelines & Project Rules (+2 more)

### Community 72 - "loadAlerts"
Cohesion: 0.29
Nodes (7): handleToggleRule(), loadAlerts(), navigateTo(), GetActiveAlerts(), GetAlertEvents(), GetAlertRules(), ToggleAlertRule()

### Community 73 - "loadQueryAnalysis"
Cohesion: 0.40
Nodes (5): handleKillQuery(), loadQueryAnalysis(), openQueryExplorer(), AnalyzeDatabaseQueries(), KillDatabaseQuery()

### Community 80 - "pull_request_template.md"
Cohesion: 0.50
Nodes (3): Checklist Kontributor & Reviewer, Deskripsi Perubahan, 🔗 Issue Terkait (Wajib)

## Knowledge Gaps
- **164 isolated node(s):** `ICONS`, `name`, `private`, `version`, `type` (+159 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 393 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **45 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `App` connect `App` to `types.go`, `context.Context`, `SSHPool`, `KeychainManager`, `SQLiteStorage`, `AlertEvent`, `UptimeMonitor`, `s3.go`, `BackupScheduler`, `BackupJobLog`, `AlertRule`, `.Close`, `Exporter`?**
  _High betweenness centrality (0.054) - this node is a cross-community bridge._
- **Why does `SQLiteStorage` connect `SQLiteStorage` to `types.go`, `context.Context`, `App`, `SSHPool`, `DatabaseType`, `KeychainManager`, `AlertEvent`, `UptimeMonitor`, `BackupScheduler`, `BackupJobLog`, `AlertRule`, `.Close`, `ServerStatus`, `Exporter`?**
  _High betweenness centrality (0.033) - this node is a cross-community bridge._
- **Why does `SSHPool` connect `SSHPool` to `types.go`, `context.Context`, `App`, `KeychainManager`, `SQLiteStorage`, `UptimeMonitor`?**
  _High betweenness centrality (0.016) - this node is a cross-community bridge._
- **What connects `ICONS`, `name`, `private` to the rest of the system?**
  _164 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `runtime.js` be split into smaller, more focused modules?**
  _Cohesion score 0.03076923076923077 - nodes in this community are weakly interconnected._
- **Should `App.js` be split into smaller, more focused modules?**
  _Cohesion score 0.06896551724137931 - nodes in this community are weakly interconnected._
- **Should `types.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07547169811320754 - nodes in this community are weakly interconnected._