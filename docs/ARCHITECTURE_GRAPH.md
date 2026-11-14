# Graph Report - Monitoring-server  (2026-11-15)

## Corpus Check
- 73 files · ~103,943 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 851 nodes · 1425 edges · 69 communities (28 shown, 33 thin omitted)
- Extraction: 96% EXTRACTED · 4% INFERRED · 0% AMBIGUOUS · INFERRED: 53 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Server Metrics Collector Engine
- Database Backup & S3 Engine
- App Core & Action Validation
- SSH Client & Database Discovery
- Alert & App Settings Models
- CI/CD Release & Attestation
- Frontend NPM Dependencies
- Notifier & Keychain Manager
- Engine & Collector Test Suite
- SQLite Storage Engine
- Svelte UI Views & Terminal
- Project Metadata & Manifest
- Uptime Monitor & TCP Checker
- S3 Client & Storage Protocol
- TypeScript Core Configuration
- Backup Task Scheduler
- Wails Project Configuration
- Backup Job Logging & History
- Server Detail UI Mockup
- Add Server Modal Mockup
- Terminal & Logs UI Mockup
- Wails Runtime Type Declarations
- Dashboard Inventory UI Mockup
- Alert Management UI Mockup
- Vite & Node TS Config
- Wails Brand Logo & Emblem
- Alert Events Data Model
- Server Status & Config Model
- Report Exporter Service
- Anti-Slop Accessibility Audits
- Chart Rendering Canvas Helper
- Mockup Shell Layout Helper
- SSH Session Data Model
- Backup Verification Result Model
- Broadcast Command Result Model
- Connection Test Result Model
- CPU Statistics Data Model
- Active Database Query Model
- Database Schema Item Model
- Database Slow Query Model
- Database Table Item Model
- Discovered Database Data Model
- Disk Partition Data Model
- Firewall Status Data Model
- Listening Port Data Model
- Memory Statistics Data Model
- Network Interface Data Model
- Notification Settings Data Model
- System Process Item Model
- S3 Backup Settings Model
- Server Configuration Data Model
- Service Item Data Model
- SSH Attacker IP Model
- Systemd Service Data Model
- Wails Event Dispatcher
- Alert Rules Data Model
- Database Configuration Data Model
- CI Test & Build Workflow
- Frontend Architecture Documentation
- Nunito Open Font License
- Go Module ServerWatch Root

## God Nodes (most connected - your core abstractions)
1. `App` - 86 edges
2. `SQLiteStorage` - 63 edges
3. `Server` - 38 edges
4. `SSHPool` - 30 edges
5. `DatabaseConfig` - 25 edges
6. `KeychainManager` - 21 edges
7. `BackupEngine` - 17 edges
8. `NewApp()` - 16 edges
9. `BackupJobLog` - 16 edges
10. `ServerMetrics` - 15 edges

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
- **Agentless SSH Security Boundary & Trust Architecture** — docs_prd_threat_model_trust_boundary, docs_prd_wails_binding_security, docs_security_audit_report_vuln_01_command_dialog, docs_mockups_src_05_terminal_log_xss_protection_demo [INFERRED 0.95]
- **CI/CD Build, Provenance Attestation & Release Gate** — _github_workflows_release_release_workflow, _github_workflows_release_build_job, _github_workflows_release_publish_job, _github_workflows_release_artifact_attestation [EXTRACTED 1.00]
- **Anti-Slop Accessibility & Design System Standardization** — anti_slop_audit_001_2026_11_05_wcag_contrast_remediation, anti_slop_audit_002_2026_11_07_aria_semantics_remediation, changelog_security_hardening [INFERRED 0.85]

## Communities (69 total, 33 thin omitted)

### Community 2 - "Server Metrics Collector Engine"
Cohesion: 0.07
Nodes (44): cpuTickSample, netTickSample, prevSample, containerStat, sync.Map, calcCPUUsage(), Collector, NewCollector() (+36 more)

### Community 3 - "Database Backup & S3 Engine"
Cohesion: 0.10
Nodes (25): BackupProgressCallback, context.Context, ExtractRemoteFilePath(), BackupEngine, NewBackupEngine(), TestValidateBackupInputs(), ValidateIdentifier(), ValidateRemotePath() (+17 more)

### Community 4 - "App Core & Action Validation"
Cohesion: 0.06
Nodes (8): setOSAutoStart(), context.CancelFunc, ValidateAction(), ValidateUnitName(), BackupVerifyResult, BroadcastResult, DatabaseTableItem, App

### Community 5 - "SSH Client & Database Discovery"
Cohesion: 0.07
Nodes (29): NewApp(), bytes.Buffer, golang.org/x/crypto/ssh.AuthMethod, golang.org/x/crypto/ssh.Client, golang.org/x/crypto/ssh.ClientConfig, golang.org/x/crypto/ssh.Session, io.WriteCloser, io.Writer (+21 more)

### Community 6 - "Alert & App Settings Models"
Cohesion: 0.05
Nodes (12): AlertEvent, AppSettings, BackupJobLog, BackupSchedule, DatabaseConfig, DatabaseMetrics, DatabaseQueryAnalysis, DevOpsSnapshot (+4 more)

### Community 7 - "CI/CD Release & Attestation"
Cohesion: 0.06
Nodes (38): Release Approval Protected Environment, Cryptographic Build Provenance & Attestation, Build & Verify (Windows AMD64) Job, Manual Approval & Publish Release Job, Release ServerWatch Workflow, ServerWatch Project Changelog, v1.0.0 Security Hardening & Remediation, ServerWatch v1.0.0 Release (+30 more)

### Community 8 - "Frontend NPM Dependencies"
Cohesion: 0.06
Nodes (31): dependencies, @xterm/addon-fit, @xterm/xterm, devDependencies, svelte, svelte-check, svelte-preprocess, @sveltejs/vite-plugin-svelte (+23 more)

### Community 9 - "Notifier & Keychain Manager"
Cohesion: 0.10
Nodes (6): net/http.Client, Notifier, NewNotifier(), TestSendTelegramAndWebhook(), KeychainManager, NewKeychainManager()

### Community 10 - "Engine & Collector Test Suite"
Cohesion: 0.12
Nodes (18): testing.T, TestBackupSchedulerIsDue(), TestRestoreDatabaseValidation(), TestParseMySQLQueryAnalysis(), TestParseMySQLStatus_Mysqladmin(), TestParseMySQLStatus_TSV(), TestValidateDBConfig(), TestParseProbeOutput_DebianNoDBMSInstalled() (+10 more)

### Community 11 - "SQLite Storage Engine"
Cohesion: 0.11
Nodes (3): database/sql.DB, SQLiteStorage, NewSQLiteStorage()

### Community 12 - "Svelte UI Views & Terminal"
Cohesion: 0.13
Nodes (4): target, AlertRule, DockerContainer, model

### Community 13 - "Project Metadata & Manifest"
Cohesion: 0.11
Nodes (18): author, bugs, url, description, homepage, keywords, license, main (+10 more)

### Community 14 - "Uptime Monitor & TCP Checker"
Cohesion: 0.16
Nodes (6): UptimeMonitor, Checker, NewChecker(), TestCheckLocalHTTPAndTCP(), TestValidateMonitorTarget(), ValidateMonitorTarget()

### Community 15 - "S3 Client & Storage Protocol"
Cohesion: 0.22
Nodes (12): S3Client, net/http.Request, net/url.Values, canonicalQueryString(), hmacSHA256(), NewS3Client(), sha256Hex(), signS3Request() (+4 more)

### Community 16 - "TypeScript Core Configuration"
Cohesion: 0.11
Nodes (17): compilerOptions, allowJs, baseUrl, checkJs, isolatedModules, module, resolveJsonModule, target (+9 more)

### Community 17 - "Backup Task Scheduler"
Cohesion: 0.18
Nodes (6): sync.WaitGroup, time.Time, BackupScheduler, NewBackupScheduler(), BackupSchedule, BackupProgressEvent

### Community 18 - "Wails Project Configuration"
Cohesion: 0.12
Nodes (16): author, email, name, frontend:build, frontend:dev:serverUrl, frontend:dev:watcher, frontend:install, info (+8 more)

### Community 20 - "Server Detail UI Mockup"
Cohesion: 0.27
Nodes (11): Server Detail View Mockup, Server Action Controls and Time Range Selector, Active Server Alerts and Notification Channels, CPU Utilization Line Chart, Disk Partition Utilization and Predictive Forecasting, Server Host Metadata and SSH Configuration, Memory and Swap Usage Line Chart, Network Traffic Throughput Monitor (+3 more)

### Community 21 - "Add Server Modal Mockup"
Cohesion: 0.33
Nodes (9): Add Server Modal Mockup UI, SSH Authentication Method Selector, Jump Host and Bastion Proxy Routing, Server Connection Configuration Fields, Connection Test and Save Actions, Credential Security and Storage Architecture, Host Key Verification and Server Fingerprint, Add Server Modal Dialog (+1 more)

### Community 22 - "Terminal & Logs UI Mockup"
Cohesion: 0.39
Nodes (9): Terminal and Log Viewer Mockup UI, Bastion Host SSH Tunneling Integration, Disk Space Troubleshooting Workflow, SSH Interactive Terminal Console, Unit and Keyword Log Filtering, Real-time Systemd Journal Log Viewer, Terminal Multi-Session Tabs, Terminal and Log Split Screen Layout (+1 more)

### Community 23 - "Wails Runtime Type Declarations"
Cohesion: 0.25
Nodes (7): EnvironmentInfo, NotificationAction, NotificationCategory, NotificationOptions, Position, Screen, Size

### Community 24 - "Dashboard Inventory UI Mockup"
Cohesion: 0.48
Nodes (7): Dashboard Mockup UI, Attention Cards Section, Header Search and Server Actions, Server Inventory and Metrics Table, Sidebar Navigation, SSH Connection and Bastion Routing, Status and Tag Filtering System

### Community 25 - "Alert Management UI Mockup"
Cohesion: 0.52
Nodes (7): Alert Management Mockup UI, Recent Incidents and Event Feed, Sidebar Navigation and Active Alert Status, Notification Channels Configuration, Alert Rules Configuration and Thresholds, Desktop Execution Constraint and Headless Monitoring, Alert Snooze Management

### Community 26 - "Vite & Node TS Config"
Cohesion: 0.29
Nodes (6): compilerOptions, composite, module, moduleResolution, include, vite.config.ts

### Community 27 - "Wails Brand Logo & Emblem"
Cohesion: 0.60
Nodes (5): Wails Universal Logo Graphic, Geometric Origami Eagle Emblem, Default Application Starter Asset, Wails Framework Branding, Wails Red Capitalized Typography Wordmark

### Community 31 - "Anti-Slop Accessibility Audits"
Cohesion: 0.50
Nodes (4): Anti-Slop UI & Code Audit 001 Report, WCAG AA Contrast & Keyboard Focus Remediation, ARIA Semantics & Design Token Standardization, Anti-Slop UI & Accessibility Audit 002 Report

### Community 33 - "Mockup Shell Layout Helper"
Cohesion: 0.67
Nodes (3): icon(), ICONS, renderShell()

### Community 56 - "Wails Event Dispatcher"
Cohesion: 0.67
Nodes (3): EventsOn(), EventsOnce(), EventsOnMultiple()

## Knowledge Gaps
- **100 isolated node(s):** `ICONS`, `name`, `private`, `version`, `type` (+95 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 355 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **33 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `App` connect `App Core & Action Validation` to `Server Metrics Collector Engine`, `Database Backup & S3 Engine`, `SSH Client & Database Discovery`, `Notifier & Keychain Manager`, `SQLite Storage Engine`, `Uptime Monitor & TCP Checker`, `S3 Client & Storage Protocol`, `Backup Task Scheduler`, `Backup Job Logging & History`, `Alert Rules Data Model`, `Alert Events Data Model`, `Report Exporter Service`?**
  _High betweenness centrality (0.072) - this node is a cross-community bridge._
- **Why does `SQLiteStorage` connect `SQLite Storage Engine` to `Server Metrics Collector Engine`, `Database Backup & S3 Engine`, `App Core & Action Validation`, `SSH Client & Database Discovery`, `Notifier & Keychain Manager`, `Uptime Monitor & TCP Checker`, `Backup Task Scheduler`, `Backup Job Logging & History`, `Alert Rules Data Model`, `Database Configuration Data Model`, `Alert Events Data Model`, `Server Status & Config Model`, `Report Exporter Service`?**
  _High betweenness centrality (0.044) - this node is a cross-community bridge._
- **Why does `SSHPool` connect `SSH Client & Database Discovery` to `Server Metrics Collector Engine`, `Database Backup & S3 Engine`, `App Core & Action Validation`, `Notifier & Keychain Manager`, `SQLite Storage Engine`, `Uptime Monitor & TCP Checker`?**
  _High betweenness centrality (0.021) - this node is a cross-community bridge._
- **What connects `ICONS`, `name`, `private` to the rest of the system?**
  _100 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Wails Desktop Runtime API` be split into smaller, more focused modules?**
  _Cohesion score 0.03076923076923077 - nodes in this community are weakly interconnected._
- **Should `App Backend Services Binding` be split into smaller, more focused modules?**
  _Cohesion score 0.03278688524590164 - nodes in this community are weakly interconnected._
- **Should `Server Metrics Collector Engine` be split into smaller, more focused modules?**
  _Cohesion score 0.06654567453115548 - nodes in this community are weakly interconnected._