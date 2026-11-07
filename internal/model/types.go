package model

import "time"

// SSHAuthType represents supported authentication mechanisms
type SSHAuthType string

const (
	AuthTypeKey      SSHAuthType = "key"
	AuthTypePassword SSHAuthType = "password"
	AuthTypeAgent    SSHAuthType = "agent"
)

// ServerStatus represents operational health status
type ServerStatus string

const (
	StatusNormal      ServerStatus = "normal"
	StatusWarning     ServerStatus = "warning"
	StatusCritical    ServerStatus = "critical"
	StatusUnreachable ServerStatus = "unreachable"
)

// Server represents a monitored target stored in SQLite
type Server struct {
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	Host               string       `json:"host"`
	Port               int          `json:"port"`
	User               string       `json:"user"`
	AuthType           SSHAuthType  `json:"authType"`
	KeyPath            string       `json:"keyPath"`
	JumpHostID         string       `json:"jumpHostId"`
	Tags               []string     `json:"tags"`
	Notes              string       `json:"notes"`
	Status             ServerStatus `json:"status"`
	StatusReason       string       `json:"statusReason"`
	LastCheck          *time.Time   `json:"lastCheck,omitempty"`
	HostKeyFingerprint string       `json:"hostKeyFingerprint"`
	TrustedHostKey     bool         `json:"trustedHostKey"`
	IntervalSec        int          `json:"intervalSec"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
}

// ServerConfig receives form input from UI (passwords are handled securely via keychain)
type ServerConfig struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Host        string      `json:"host"`
	Port        int         `json:"port"`
	User        string      `json:"user"`
	AuthType    SSHAuthType `json:"authType"`
	KeyPath     string      `json:"keyPath"`
	Passphrase  string      `json:"passphrase,omitempty"`
	Password    string      `json:"password,omitempty"`
	JumpHostID  string      `json:"jumpHostId"`
	Tags        []string    `json:"tags"`
	Notes       string      `json:"notes"`
	IntervalSec int         `json:"intervalSec"`
}

// ConnectionTestResult contains diagnostics for SSH connection testing
type ConnectionTestResult struct {
	Success            bool   `json:"success"`
	Error              string `json:"error,omitempty"`
	HostKeyType        string `json:"hostKeyType,omitempty"`
	HostKeyFingerprint string `json:"hostKeyFingerprint,omitempty"`
	OSInfo             string `json:"osInfo,omitempty"`
	KernelInfo         string `json:"kernelInfo,omitempty"`
	ShellPath          string `json:"shellPath,omitempty"`
	HasSystemd         bool   `json:"hasSystemd"`
	LatencyMs          int64  `json:"latencyMs"`
}

// CPUStats details processor load and utilization
type CPUStats struct {
	TotalPercent float64   `json:"totalPercent"`
	Cores        []float64 `json:"cores,omitempty"`
	Load1        float64   `json:"load1"`
	Load5        float64   `json:"load5"`
	Load15       float64   `json:"load15"`
}

// MemStats details RAM and Swap memory utilization
type MemStats struct {
	TotalBytes      uint64  `json:"totalBytes"`
	UsedBytes       uint64  `json:"usedBytes"`
	FreeBytes       uint64  `json:"freeBytes"`
	AvailableBytes  uint64  `json:"availableBytes"`
	CachedBytes     uint64  `json:"cachedBytes"`
	SwapTotalBytes  uint64  `json:"swapTotalBytes"`
	SwapUsedBytes   uint64  `json:"swapUsedBytes"`
	UsedPercent     float64 `json:"usedPercent"`
	SwapUsedPercent float64 `json:"swapUsedPercent"`
}

// DiskPartition details partition usage
type DiskPartition struct {
	MountPoint  string  `json:"mountPoint"`
	Device      string  `json:"device"`
	FSType      string  `json:"fsType"`
	TotalBytes  uint64  `json:"totalBytes"`
	UsedBytes   uint64  `json:"usedBytes"`
	FreeBytes   uint64  `json:"freeBytes"`
	UsedPercent float64 `json:"usedPercent"`
}

// NetInterface details network traffic
type NetInterface struct {
	Name         string  `json:"name"`
	RxBytesSec   float64 `json:"rxBytesSec"`
	TxBytesSec   float64 `json:"txBytesSec"`
	TotalRxBytes uint64  `json:"totalRxBytes"`
	TotalTxBytes uint64  `json:"totalTxBytes"`
}

// ProcessItem details high-resource processes
type ProcessItem struct {
	PID        int     `json:"pid"`
	User       string  `json:"user"`
	CPUPercent float64 `json:"cpuPercent"`
	MemPercent float64 `json:"memPercent"`
	Command    string  `json:"command"`
}

// ServiceItem details systemd service status
type ServiceItem struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ServerMetrics is a snapshot of all system performance metrics
type ServerMetrics struct {
	ServerID      string          `json:"serverId"`
	Timestamp     time.Time       `json:"timestamp"`
	CPU           CPUStats        `json:"cpu"`
	Memory        MemStats        `json:"memory"`
	Disks         []DiskPartition `json:"disks"`
	Networks      []NetInterface  `json:"networks"`
	Processes     []ProcessItem   `json:"processes"`
	Services      []ServiceItem   `json:"services"`
	UptimeSeconds uint64          `json:"uptimeSeconds"`
	OSInfo        string          `json:"osInfo"`
	KernelInfo    string          `json:"kernelInfo"`
}

// AlertRule defines metric threshold rules
type AlertRule struct {
	ID          string  `json:"id"`
	Metric      string  `json:"metric"` // "cpu", "disk", "memory", "unreachable"
	Operator    string  `json:"operator"` // ">", ">=", "<"
	Threshold   float64 `json:"threshold"`
	DurationSec int     `json:"durationSec"`
	Level       string  `json:"level"` // "warning", "critical"
	TargetTags  string  `json:"targetTags"` // "all" or specific tag
	Enabled     bool    `json:"enabled"`
	Channels    string  `json:"channels"` // "desktop,telegram"
}

// AlertEvent logs historical occurrences of alerts
type AlertEvent struct {
	ID         int        `json:"id"`
	ServerID   string     `json:"serverId"`
	ServerName string     `json:"serverName"`
	RuleID     string     `json:"ruleId"`
	Level      string     `json:"level"`
	Message    string     `json:"message"`
	StartedAt  time.Time  `json:"startedAt"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
	Status     string     `json:"status"` // "active", "resolved"
}

// DatabaseType represents supported DBMS
type DatabaseType string

const (
	DBTypeMySQL      DatabaseType = "mysql"
	DBTypePostgres   DatabaseType = "postgres"
	DBTypeRedis      DatabaseType = "redis"
)

// DatabaseSchemaItem details an individual schema/database inside an instance
type DatabaseSchemaItem struct {
	Name       string `json:"name"`
	TableCount int    `json:"tableCount"`
	SizeBytes  uint64 `json:"sizeBytes"`
}

// DiscoveredDatabase holds auto-detected database instances on a server
type DiscoveredDatabase struct {
	Type         DatabaseType `json:"type"`
	Name         string       `json:"name"`
	Host         string       `json:"host"`
	Port         int          `json:"port"`
	SocketPath   string       `json:"socketPath,omitempty"`
	ServiceState string       `json:"serviceState"` // "running", "active", "listening"
	Version      string       `json:"version,omitempty"`
	DefaultUser  string       `json:"defaultUser"`
	Databases    []string     `json:"databases,omitempty"`
}

// DatabaseConfig stores target database connection parameters
type DatabaseConfig struct {
	ID           string       `json:"id"`
	ServerID     string       `json:"serverId"`
	Type         DatabaseType `json:"type"`
	Name         string       `json:"name"`
	Host         string       `json:"host"`
	Port         int          `json:"port"`
	User         string       `json:"user"`
	Password     string       `json:"password,omitempty"`
	DatabaseName string       `json:"databaseName"`
	Enabled      bool         `json:"enabled"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

// DatabaseMetrics holds operational telemetry of a DBMS
type DatabaseMetrics struct {
	ID                int       `json:"id"`
	DatabaseConfigID  string    `json:"databaseConfigId"`
	Timestamp         time.Time `json:"timestamp"`
	Status            string    `json:"status"` // "up", "down", "warning"
	UptimeSeconds     uint64    `json:"uptimeSeconds"`
	ActiveConnections int       `json:"activeConnections"`
	MaxConnections    int       `json:"maxConnections"`
	QueriesPerSec     float64   `json:"queriesPerSec"`
	SlowQueries       int64     `json:"slowQueries"`
	CacheHitRate      float64   `json:"cacheHitRate"`
	MemoryUsedBytes   uint64    `json:"memoryUsedBytes"`
	DatabaseSizeBytes uint64    `json:"databaseSizeBytes"`
	Version           string    `json:"version"`
}

// BackupSchedule defines automated database dump rules
type BackupSchedule struct {
	ID               string     `json:"id"`
	ServerID         string     `json:"serverId"`
	DatabaseConfigID string     `json:"databaseConfigId"`
	DatabaseName     string     `json:"databaseName"`
	Frequency        string     `json:"frequency"` // "daily", "weekly", "hourly"
	TimeOfDay        string     `json:"timeOfDay"` // "02:00"
	DayOfWeek        int        `json:"dayOfWeek"` // 0-6 (Sunday-Saturday)
	RetentionDays    int        `json:"retentionDays"`
	StorageTarget    string     `json:"storageTarget"` // "remote", "local"
	RemotePath       string     `json:"remotePath"`
	LocalPath        string     `json:"localPath"`
	Compression      string     `json:"compression"` // "gzip"
	Enabled          bool       `json:"enabled"`
	LastRun          *time.Time `json:"lastRun,omitempty"`
	NextRun          *time.Time `json:"nextRun,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// BackupJobLog records historical dump executions and results
type BackupJobLog struct {
	ID               int        `json:"id"`
	ScheduleID       string     `json:"scheduleId"`
	ServerID         string     `json:"serverId"`
	DatabaseConfigID string     `json:"databaseConfigId"`
	DatabaseName     string     `json:"databaseName"`
	StartedAt        time.Time  `json:"startedAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
	Status           string     `json:"status"` // "running", "success", "failed"
	FileSizeBytes    int64      `json:"fileSizeBytes"`
	FilePath         string     `json:"filePath"`
	DurationSec      float64    `json:"durationSec"`
	ErrorMessage     string     `json:"errorMessage,omitempty"`
}

// BackupProgressEvent represents real-time progress for a background database backup execution
type BackupProgressEvent struct {
	JobID            int       `json:"jobId"`
	ServerID         string    `json:"serverId"`
	ServerName       string    `json:"serverName"`
	DatabaseConfigID string    `json:"databaseConfigId"`
	DatabaseName     string    `json:"databaseName"`
	StorageTarget    string    `json:"storageTarget"`
	Stage            string    `json:"stage"` // "started", "preflight", "dumping", "verifying", "transferring", "retention", "completed", "failed"
	Message          string    `json:"message"`
	ProgressPercent  int       `json:"progressPercent"` // 5 to 100
	StartedAt        time.Time `json:"startedAt"`
	ElapsedSec       float64   `json:"elapsedSec"`
	FileSizeBytes    int64     `json:"fileSizeBytes,omitempty"`
	FilePath         string    `json:"filePath,omitempty"`
	ErrorMessage     string    `json:"errorMessage,omitempty"`
}

// DockerContainer represents a container discovered on the remote host
type DockerContainer struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Image      string  `json:"image"`
	State      string  `json:"state"`  // "running", "exited", "restarting", "paused"
	Status     string  `json:"status"` // e.g. "Up 4 hours"
	Ports      string  `json:"ports"`
	CPUPercent float64 `json:"cpuPercent"`
	MemUsage   string  `json:"memUsage"`
	MemPercent float64 `json:"memPercent"`
	NetIO      string  `json:"netIO"`
	BlockIO    string  `json:"blockIO"`
}

// SystemdService represents a systemd service unit on the remote host
type SystemdService struct {
	Name         string `json:"name"`
	LoadState    string `json:"loadState"`
	ActiveState  string `json:"activeState"` // "active", "inactive", "failed"
	SubState     string `json:"subState"`    // "running", "exited", "dead"
	Description  string `json:"description"`
	IsKeyService bool   `json:"isKeyService"`
}

// ListeningPort represents a network socket listening on the server
type ListeningPort struct {
	Protocol     string `json:"protocol"`     // "tcp", "udp"
	LocalAddress string `json:"localAddress"` // "0.0.0.0", "127.0.0.1", "::"
	Port         int    `json:"port"`
	Process      string `json:"process"`
	PID          int    `json:"pid"`
	Exposure     string `json:"exposure"` // "public", "local"
}

// FirewallStatus summarizes host firewall state
type FirewallStatus struct {
	Installed bool   `json:"installed"`
	Active    bool   `json:"active"`
	ToolName  string `json:"toolName"` // "ufw", "firewalld", "iptables", "none"
	Summary   string `json:"summary"`
}

// SSHAttackerIP records a source IP with repeated failed SSH authentication attempts
type SSHAttackerIP struct {
	IP       string `json:"ip"`
	Attempts int    `json:"attempts"`
}

// ActiveSSHSession represents a currently logged-in user session
type ActiveSSHSession struct {
	User      string `json:"user"`
	Terminal  string `json:"terminal"`
	FromIP    string `json:"fromIp"`
	LoginTime string `json:"loginTime"`
}

// SecurityAudit holds OS update counts, SSH brute-force telemetry, and hardening checks
type SecurityAudit struct {
	PendingUpdates  int                `json:"pendingUpdates"`
	SecurityUpdates int                `json:"securityUpdates"`
	PackageManager  string             `json:"packageManager"`
	FailedSSHCount  int                `json:"failedSshCount"`
	TopAttackers    []SSHAttackerIP    `json:"topAttackers"`
	ActiveSessions  []ActiveSSHSession `json:"activeSessions"`
	PermitRootLogin string             `json:"permitRootLogin"`
	PasswordAuth    string             `json:"passwordAuth"`
	Fail2banActive  bool               `json:"fail2banActive"`
}

// DevOpsSnapshot aggregates container, service, port, firewall, and security states
type DevOpsSnapshot struct {
	ServerID        string            `json:"serverId"`
	Timestamp       time.Time         `json:"timestamp"`
	DockerAvailable bool              `json:"dockerAvailable"`
	DockerVersion   string            `json:"dockerVersion"`
	Containers      []DockerContainer `json:"containers"`
	Services        []SystemdService  `json:"services"`
	ListeningPorts  []ListeningPort   `json:"listeningPorts"`
	Firewall        FirewallStatus    `json:"firewall"`
	Security        SecurityAudit     `json:"security"`
}

// UptimeMonitor defines an HTTP(S), SSL, or TCP endpoint monitor (FR-14 & FR-15)
type UptimeMonitor struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`     // "http" or "tcp"
	Target          string     `json:"target"`   // URL ("https://example.com") or "host:port"
	ServerID        string     `json:"serverId"` // Empty = check from desktop app; non-empty = check via SSH from target server (FR-15)
	IntervalSec     int        `json:"intervalSec"`
	Enabled         bool       `json:"enabled"`
	Status          string     `json:"status"` // "up", "down", "warning"
	StatusCode      int        `json:"statusCode"`
	LatencyMs       int64      `json:"latencyMs"`
	SSLExpiryDays   int        `json:"sslExpiryDays"`
	SSLIssuer       string     `json:"sslIssuer"`
	SSLValidUntil   string     `json:"sslValidUntil"`
	LastError       string     `json:"lastError"`
	LastCheck       *time.Time `json:"lastCheck,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// NotificationSettings holds external alerting configuration (FR-25, SEC-08)
// Secrets (Bot Token & Webhook URL) are stored only in OS Keychain and never sent back in plaintext.
type NotificationSettings struct {
	DesktopEnabled     bool   `json:"desktopEnabled"`
	TelegramEnabled    bool   `json:"telegramEnabled"`
	TelegramChatID     string `json:"telegramChatId"`
	TelegramBotToken   string `json:"telegramBotToken,omitempty"` // Input-only when saving
	HasTelegramToken   bool   `json:"hasTelegramToken"`
	WebhookEnabled     bool   `json:"webhookEnabled"`
	WebhookURL         string `json:"webhookUrl,omitempty"` // Input-only when saving
	HasWebhookURL      bool   `json:"hasWebhookUrl"`
	NotifyOnRecovery   bool   `json:"notifyOnRecovery"`
	NotifyOnBackup     bool   `json:"notifyOnBackup"`
}

// AppSettings stores desktop application preferences (FR-29, FR-32, FR-33)
type AppSettings struct {
	RetentionDays      int                  `json:"retentionDays"`
	AutoStartOS        bool                 `json:"autoStartOs"`
	RunInBackground    bool                 `json:"runInBackground"`
	ConfirmSSHCommands bool                 `json:"confirmSshCommands"`
	Notifications      NotificationSettings `json:"notifications"`
	S3Backup           S3BackupSettings     `json:"s3Backup"`
}

// S3BackupSettings holds S3-compatible offsite backup storage configuration (AWS S3, Cloudflare R2, MinIO).
// SecretAccessKey is stored exclusively in OS Keychain (SEC-08 & SEC-15).
type S3BackupSettings struct {
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	PathPrefix      string `json:"pathPrefix"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
	HasSecretKey    bool   `json:"hasSecretKey"`
}

// DatabaseActiveQuery represents an in-flight SQL query or active client session
type DatabaseActiveQuery struct {
	PID      string  `json:"pid"`
	User     string  `json:"user"`
	Host     string  `json:"host"`
	Database string  `json:"database"`
	Command  string  `json:"command"`
	TimeSec  float64 `json:"timeSec"`
	State    string  `json:"state"`
	Query    string  `json:"query"`
}

// DatabaseSlowQuery represents an aggregated slow query digest or slowlog entry
type DatabaseSlowQuery struct {
	SchemaName   string  `json:"schemaName"`
	ExecCount    int64   `json:"execCount"`
	AvgTimeMs    float64 `json:"avgTimeMs"`
	MaxTimeMs    float64 `json:"maxTimeMs"`
	RowsExamined int64   `json:"rowsExamined"`
	QueryText    string  `json:"queryText"`
}

// DatabaseTableItem details a table inside a database schema
type DatabaseTableItem struct {
	TableName  string `json:"tableName"`
	Engine     string `json:"engine"`
	RowCount   int64  `json:"rowCount"`
	DataBytes  uint64 `json:"dataBytes"`
	IndexBytes uint64 `json:"indexBytes"`
	TotalBytes uint64 `json:"totalBytes"`
}

// DatabaseQueryAnalysis aggregates active queries and slow query telemetry for a DBMS
type DatabaseQueryAnalysis struct {
	DatabaseConfigID string                `json:"databaseConfigId"`
	ActiveQueries    []DatabaseActiveQuery `json:"activeQueries"`
	SlowQueries      []DatabaseSlowQuery   `json:"slowQueries"`
	SourceNote       string                `json:"sourceNote"`
}

// BackupVerifyResult reports archive integrity and checksum verification
type BackupVerifyResult struct {
	BackupLogID   int    `json:"backupLogId"`
	Valid         bool   `json:"valid"`
	FileExists    bool   `json:"fileExists"`
	FileSizeBytes int64  `json:"fileSizeBytes"`
	SHA256        string `json:"sha256"`
	Message       string `json:"message"`
}

// BroadcastResult details command execution output across a server in multi-server broadcast
type BroadcastResult struct {
	ServerID   string   `json:"serverId"`
	ServerName string   `json:"serverName"`
	Host       string   `json:"host"`
	Tags       []string `json:"tags"`
	Output     string   `json:"output"`
	Error      string   `json:"error,omitempty"`
	DurationMs int64    `json:"durationMs"`
	Success    bool     `json:"success"`
}






