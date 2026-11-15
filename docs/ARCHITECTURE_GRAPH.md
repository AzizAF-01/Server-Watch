# Graph Report - Monitoring-server  (2026-11-15)

## Corpus Check
- 62 files · ~106,972 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 912 nodes · 1483 edges · 83 communities (34 shown, 41 thin omitted)
- Extraction: 96% EXTRACTED · 4% INFERRED · 0% AMBIGUOUS · INFERRED: 53 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `a157221a`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
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
- models.ts
- runtime/package.json
- UptimeMonitor
- BackupEngine
- compilerOptions
- time.Time
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
- AlertEvent
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
- Communities (69 total, 33 thin omitted)
- DiscoveredDatabase
- DiskPartition
- FirewallStatus
- ListeningPort
- MemStats
- NetInterface
- NotificationSettings
- ProcessItem
- S3BackupSettings
- ServerConfig
- ServiceItem
- SSHAttackerIP
- SystemdService
- EventsOnMultiple
- AlertRule
- Manager
- CI Test & Build Validation Workflow
- Frontend Vite + Svelte Architecture Notes
- Nunito Typography SIL Open Font License 1.1
- serverwatch
- .Close
- 🚨 Siklus Hidup Standar Pengerjaan Tugas (Issue ➔ Branch ➔ PR ➔ Merge ➔ Hapus Branch)
- DBDetector
- NewApp
- boundedBuffer
- AlertEvent
- BackupJobLog
- BackupSchedule
- DatabaseQueryAnalysis
- SecurityAudit
- Server
- pull_request_template.md
- AlertRule
- DockerContainer

## God Nodes (most connected - your core abstractions)
1. `App` - 86 edges
2. `SQLiteStorage` - 63 edges
3. `Server` - 38 edges
4. `SSHPool` - 30 edges
5. `Communities (69 total, 33 thin omitted)` - 29 edges
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

## Communities (83 total, 41 thin omitted)

### Community 2 - "types.go"
Cohesion: 0.09
Nodes (30): cpuTickSample, netTickSample, prevSample, sync.Map, calcCPUUsage(), Collector, NewCollector(), parseCPULines() (+22 more)

### Community 3 - "context.Context"
Cohesion: 0.08
Nodes (22): context.Context, ExtractRemoteFilePath(), DBCollector, NewDBCollector(), parseMySQLStatus(), parseMySQLVar(), TestParseRedisQueryAnalysisAndTables(), ValidateDBConfig() (+14 more)

### Community 4 - "App"
Cohesion: 0.07
Nodes (5): setOSAutoStart(), context.CancelFunc, BackupVerifyResult, BroadcastResult, App

### Community 5 - "SSHPool"
Cohesion: 0.26
Nodes (8): golang.org/x/crypto/ssh.AuthMethod, golang.org/x/crypto/ssh.Client, golang.org/x/crypto/ssh.ClientConfig, io.Writer, ConnectionTestResult, ServerConfig, SSHPool, newBoundedBuffer()

### Community 6 - ".convertValues"
Cohesion: 0.11
Nodes (6): AppSettings, DatabaseConfig, DatabaseMetrics, DevOpsSnapshot, ServerMetrics, UptimeMonitor

### Community 7 - "ServerWatch Product Requirements Document v0.4"
Cohesion: 0.05
Nodes (45): Release Approval Protected Environment, Cryptographic Build Provenance & Attestation, Build & Verify (Windows AMD64) Job, Manual Approval & Publish Release Job, Release ServerWatch Workflow, ServerWatch Project Changelog, v1.0.0 Security Hardening & Remediation, ServerWatch v1.0.0 Release (+37 more)

### Community 8 - "devDependencies"
Cohesion: 0.06
Nodes (31): dependencies, @xterm/addon-fit, @xterm/xterm, devDependencies, svelte, svelte-check, svelte-preprocess, @sveltejs/vite-plugin-svelte (+23 more)

### Community 10 - "testing.T"
Cohesion: 0.08
Nodes (34): containerStat, testing.T, TestBackupSchedulerIsDue(), TestRestoreDatabaseValidation(), TestParseMySQLQueryAnalysis(), TestParseMySQLStatus_Mysqladmin(), TestParseMySQLStatus_TSV(), TestValidateDBConfig() (+26 more)

### Community 11 - "SQLiteStorage"
Cohesion: 0.13
Nodes (3): database/sql.DB, SQLiteStorage, NewSQLiteStorage()

### Community 12 - "models.ts"
Cohesion: 0.16
Nodes (3): target, DatabaseTableItem, model

### Community 13 - "runtime/package.json"
Cohesion: 0.11
Nodes (18): author, bugs, url, description, homepage, keywords, license, main (+10 more)

### Community 14 - "UptimeMonitor"
Cohesion: 0.16
Nodes (6): UptimeMonitor, Checker, NewChecker(), TestCheckLocalHTTPAndTCP(), TestValidateMonitorTarget(), ValidateMonitorTarget()

### Community 15 - "BackupEngine"
Cohesion: 0.12
Nodes (20): BackupProgressCallback, S3Client, net/http.Client, net/http.Request, net/url.Values, BackupEngine, NewBackupEngine(), TestValidateBackupInputs() (+12 more)

### Community 16 - "compilerOptions"
Cohesion: 0.11
Nodes (17): compilerOptions, allowJs, baseUrl, checkJs, isolatedModules, module, resolveJsonModule, target (+9 more)

### Community 17 - "time.Time"
Cohesion: 0.26
Nodes (5): sync.WaitGroup, time.Time, BackupScheduler, NewBackupScheduler(), BackupProgressEvent

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

### Community 42 - "Communities (69 total, 33 thin omitted)"
Cohesion: 0.05
Nodes (39): Communities (69 total, 33 thin omitted), Community 10 - "Engine & Collector Test Suite", Community 11 - "SQLite Storage Engine", Community 12 - "Svelte UI Views & Terminal", Community 13 - "Project Metadata & Manifest", Community 14 - "Uptime Monitor & TCP Checker", Community 15 - "S3 Client & Storage Protocol", Community 16 - "TypeScript Core Configuration" (+31 more)

### Community 56 - "EventsOnMultiple"
Cohesion: 0.67
Nodes (3): EventsOn(), EventsOnce(), EventsOnMultiple()

### Community 58 - "Manager"
Cohesion: 0.18
Nodes (8): golang.org/x/crypto/ssh.Session, io.WriteCloser, sync.Mutex, sync.RWMutex, Manager, NewManager(), Session, windowChangeMsg

### Community 70 - "🚨 Siklus Hidup Standar Pengerjaan Tugas (Issue ➔ Branch ➔ PR ➔ Merge ➔ Hapus Branch)"
Cohesion: 0.22
Nodes (8): Langkah 1: Wajib Buat Issue Terlebih Dahulu, Langkah 2: Buat Branch Baru untuk Pengerjaan, Langkah 3: Koding & Pengujian Mandiri, Langkah 4: Buka Pull Request (PR), Langkah 5: Merge & Wajib Hapus Branch, ServerWatch - AI Agent Guidelines & Project Rules, 🚨 Siklus Hidup Standar Pengerjaan Tugas (Issue ➔ Branch ➔ PR ➔ Merge ➔ Hapus Branch), Standar Kualitas & Keamanan Kode Proyek

### Community 71 - "DBDetector"
Cohesion: 0.36
Nodes (5): extractDBNames(), extractVersion(), DBDetector, NewDBDetector(), DiscoveredDatabase

### Community 72 - "NewApp"
Cohesion: 0.25
Nodes (5): NewApp(), NewNotifier(), TestSendTelegramAndWebhook(), NewSSHPool(), main()

### Community 73 - "boundedBuffer"
Cohesion: 0.29
Nodes (4): bytes.Buffer, net.Conn, connectSSHAgent(), boundedBuffer

### Community 80 - "pull_request_template.md"
Cohesion: 0.50
Nodes (3): Checklist Kontributor & Reviewer, Deskripsi Perubahan, 🔗 Issue Terkait (Wajib)

## Knowledge Gaps
- **151 isolated node(s):** `ICONS`, `name`, `private`, `version`, `type` (+146 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 408 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **41 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `App` connect `App` to `types.go`, `context.Context`, `SSHPool`, `.Close`, `DBDetector`, `NewApp`, `KeychainManager`, `testing.T`, `SQLiteStorage`, `UptimeMonitor`, `BackupEngine`, `time.Time`, `BackupJobLog`, `AlertRule`, `Manager`, `AlertEvent`, `Exporter`?**
  _High betweenness centrality (0.063) - this node is a cross-community bridge._
- **Why does `SQLiteStorage` connect `SQLiteStorage` to `types.go`, `context.Context`, `App`, `SSHPool`, `.Close`, `NewApp`, `UptimeMonitor`, `BackupEngine`, `time.Time`, `BackupJobLog`, `AlertRule`, `Manager`, `AlertEvent`, `ServerStatus`, `Exporter`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._
- **Why does `SSHPool` connect `SSHPool` to `types.go`, `context.Context`, `App`, `DBDetector`, `NewApp`, `boundedBuffer`, `testing.T`, `KeychainManager`, `SQLiteStorage`, `UptimeMonitor`, `BackupEngine`, `Manager`?**
  _High betweenness centrality (0.018) - this node is a cross-community bridge._
- **What connects `ICONS`, `name`, `private` to the rest of the system?**
  _151 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `runtime.js` be split into smaller, more focused modules?**
  _Cohesion score 0.03076923076923077 - nodes in this community are weakly interconnected._
- **Should `App.js` be split into smaller, more focused modules?**
  _Cohesion score 0.03278688524590164 - nodes in this community are weakly interconnected._
- **Should `types.go` be split into smaller, more focused modules?**
  _Cohesion score 0.08710801393728224 - nodes in this community are weakly interconnected._