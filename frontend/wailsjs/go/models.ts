export namespace model {
	
	export class ActiveSSHSession {
	    user: string;
	    terminal: string;
	    fromIp: string;
	    loginTime: string;
	
	    static createFrom(source: any = {}) {
	        return new ActiveSSHSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user = source["user"];
	        this.terminal = source["terminal"];
	        this.fromIp = source["fromIp"];
	        this.loginTime = source["loginTime"];
	    }
	}
	export class AlertEvent {
	    id: number;
	    serverId: string;
	    serverName: string;
	    ruleId: string;
	    level: string;
	    message: string;
	    // Go type: time
	    startedAt: any;
	    // Go type: time
	    resolvedAt?: any;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new AlertEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.serverId = source["serverId"];
	        this.serverName = source["serverName"];
	        this.ruleId = source["ruleId"];
	        this.level = source["level"];
	        this.message = source["message"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.resolvedAt = this.convertValues(source["resolvedAt"], null);
	        this.status = source["status"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AlertRule {
	    id: string;
	    metric: string;
	    operator: string;
	    threshold: number;
	    durationSec: number;
	    level: string;
	    targetTags: string;
	    enabled: boolean;
	    channels: string;
	
	    static createFrom(source: any = {}) {
	        return new AlertRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.metric = source["metric"];
	        this.operator = source["operator"];
	        this.threshold = source["threshold"];
	        this.durationSec = source["durationSec"];
	        this.level = source["level"];
	        this.targetTags = source["targetTags"];
	        this.enabled = source["enabled"];
	        this.channels = source["channels"];
	    }
	}
	export class S3BackupSettings {
	    enabled: boolean;
	    endpoint: string;
	    region: string;
	    bucket: string;
	    pathPrefix: string;
	    accessKeyId: string;
	    secretAccessKey?: string;
	    hasSecretKey: boolean;
	
	    static createFrom(source: any = {}) {
	        return new S3BackupSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.endpoint = source["endpoint"];
	        this.region = source["region"];
	        this.bucket = source["bucket"];
	        this.pathPrefix = source["pathPrefix"];
	        this.accessKeyId = source["accessKeyId"];
	        this.secretAccessKey = source["secretAccessKey"];
	        this.hasSecretKey = source["hasSecretKey"];
	    }
	}
	export class NotificationSettings {
	    desktopEnabled: boolean;
	    telegramEnabled: boolean;
	    telegramChatId: string;
	    telegramBotToken?: string;
	    hasTelegramToken: boolean;
	    webhookEnabled: boolean;
	    webhookUrl?: string;
	    hasWebhookUrl: boolean;
	    notifyOnRecovery: boolean;
	    notifyOnBackup: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NotificationSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.desktopEnabled = source["desktopEnabled"];
	        this.telegramEnabled = source["telegramEnabled"];
	        this.telegramChatId = source["telegramChatId"];
	        this.telegramBotToken = source["telegramBotToken"];
	        this.hasTelegramToken = source["hasTelegramToken"];
	        this.webhookEnabled = source["webhookEnabled"];
	        this.webhookUrl = source["webhookUrl"];
	        this.hasWebhookUrl = source["hasWebhookUrl"];
	        this.notifyOnRecovery = source["notifyOnRecovery"];
	        this.notifyOnBackup = source["notifyOnBackup"];
	    }
	}
	export class AppSettings {
	    retentionDays: number;
	    autoStartOs: boolean;
	    runInBackground: boolean;
	    confirmSshCommands: boolean;
	    notifications: NotificationSettings;
	    s3Backup: S3BackupSettings;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.retentionDays = source["retentionDays"];
	        this.autoStartOs = source["autoStartOs"];
	        this.runInBackground = source["runInBackground"];
	        this.confirmSshCommands = source["confirmSshCommands"];
	        this.notifications = this.convertValues(source["notifications"], NotificationSettings);
	        this.s3Backup = this.convertValues(source["s3Backup"], S3BackupSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupJobLog {
	    id: number;
	    scheduleId: string;
	    serverId: string;
	    databaseConfigId: string;
	    databaseName: string;
	    // Go type: time
	    startedAt: any;
	    // Go type: time
	    completedAt?: any;
	    status: string;
	    fileSizeBytes: number;
	    filePath: string;
	    durationSec: number;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupJobLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.scheduleId = source["scheduleId"];
	        this.serverId = source["serverId"];
	        this.databaseConfigId = source["databaseConfigId"];
	        this.databaseName = source["databaseName"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.status = source["status"];
	        this.fileSizeBytes = source["fileSizeBytes"];
	        this.filePath = source["filePath"];
	        this.durationSec = source["durationSec"];
	        this.errorMessage = source["errorMessage"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupSchedule {
	    id: string;
	    serverId: string;
	    databaseConfigId: string;
	    databaseName: string;
	    frequency: string;
	    timeOfDay: string;
	    dayOfWeek: number;
	    retentionDays: number;
	    storageTarget: string;
	    remotePath: string;
	    localPath: string;
	    compression: string;
	    enabled: boolean;
	    // Go type: time
	    lastRun?: any;
	    // Go type: time
	    nextRun?: any;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new BackupSchedule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.serverId = source["serverId"];
	        this.databaseConfigId = source["databaseConfigId"];
	        this.databaseName = source["databaseName"];
	        this.frequency = source["frequency"];
	        this.timeOfDay = source["timeOfDay"];
	        this.dayOfWeek = source["dayOfWeek"];
	        this.retentionDays = source["retentionDays"];
	        this.storageTarget = source["storageTarget"];
	        this.remotePath = source["remotePath"];
	        this.localPath = source["localPath"];
	        this.compression = source["compression"];
	        this.enabled = source["enabled"];
	        this.lastRun = this.convertValues(source["lastRun"], null);
	        this.nextRun = this.convertValues(source["nextRun"], null);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupVerifyResult {
	    backupLogId: number;
	    valid: boolean;
	    fileExists: boolean;
	    fileSizeBytes: number;
	    sha256: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupVerifyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backupLogId = source["backupLogId"];
	        this.valid = source["valid"];
	        this.fileExists = source["fileExists"];
	        this.fileSizeBytes = source["fileSizeBytes"];
	        this.sha256 = source["sha256"];
	        this.message = source["message"];
	    }
	}
	export class BroadcastResult {
	    serverId: string;
	    serverName: string;
	    host: string;
	    tags: string[];
	    output: string;
	    error?: string;
	    durationMs: number;
	    success: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BroadcastResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverId = source["serverId"];
	        this.serverName = source["serverName"];
	        this.host = source["host"];
	        this.tags = source["tags"];
	        this.output = source["output"];
	        this.error = source["error"];
	        this.durationMs = source["durationMs"];
	        this.success = source["success"];
	    }
	}
	export class CPUStats {
	    totalPercent: number;
	    cores?: number[];
	    load1: number;
	    load5: number;
	    load15: number;
	
	    static createFrom(source: any = {}) {
	        return new CPUStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalPercent = source["totalPercent"];
	        this.cores = source["cores"];
	        this.load1 = source["load1"];
	        this.load5 = source["load5"];
	        this.load15 = source["load15"];
	    }
	}
	export class ConnectionTestResult {
	    success: boolean;
	    error?: string;
	    hostKeyType?: string;
	    hostKeyFingerprint?: string;
	    osInfo?: string;
	    kernelInfo?: string;
	    shellPath?: string;
	    hasSystemd: boolean;
	    latencyMs: number;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.hostKeyType = source["hostKeyType"];
	        this.hostKeyFingerprint = source["hostKeyFingerprint"];
	        this.osInfo = source["osInfo"];
	        this.kernelInfo = source["kernelInfo"];
	        this.shellPath = source["shellPath"];
	        this.hasSystemd = source["hasSystemd"];
	        this.latencyMs = source["latencyMs"];
	    }
	}
	export class DatabaseActiveQuery {
	    pid: string;
	    user: string;
	    host: string;
	    database: string;
	    command: string;
	    timeSec: number;
	    state: string;
	    query: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseActiveQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.user = source["user"];
	        this.host = source["host"];
	        this.database = source["database"];
	        this.command = source["command"];
	        this.timeSec = source["timeSec"];
	        this.state = source["state"];
	        this.query = source["query"];
	    }
	}
	export class DatabaseConfig {
	    id: string;
	    serverId: string;
	    type: string;
	    name: string;
	    host: string;
	    port: number;
	    user: string;
	    password?: string;
	    databaseName: string;
	    enabled: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.serverId = source["serverId"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.databaseName = source["databaseName"];
	        this.enabled = source["enabled"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DatabaseMetrics {
	    id: number;
	    databaseConfigId: string;
	    // Go type: time
	    timestamp: any;
	    status: string;
	    uptimeSeconds: number;
	    activeConnections: number;
	    maxConnections: number;
	    queriesPerSec: number;
	    slowQueries: number;
	    cacheHitRate: number;
	    memoryUsedBytes: number;
	    databaseSizeBytes: number;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.databaseConfigId = source["databaseConfigId"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.status = source["status"];
	        this.uptimeSeconds = source["uptimeSeconds"];
	        this.activeConnections = source["activeConnections"];
	        this.maxConnections = source["maxConnections"];
	        this.queriesPerSec = source["queriesPerSec"];
	        this.slowQueries = source["slowQueries"];
	        this.cacheHitRate = source["cacheHitRate"];
	        this.memoryUsedBytes = source["memoryUsedBytes"];
	        this.databaseSizeBytes = source["databaseSizeBytes"];
	        this.version = source["version"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DatabaseSlowQuery {
	    schemaName: string;
	    execCount: number;
	    avgTimeMs: number;
	    maxTimeMs: number;
	    rowsExamined: number;
	    queryText: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseSlowQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaName = source["schemaName"];
	        this.execCount = source["execCount"];
	        this.avgTimeMs = source["avgTimeMs"];
	        this.maxTimeMs = source["maxTimeMs"];
	        this.rowsExamined = source["rowsExamined"];
	        this.queryText = source["queryText"];
	    }
	}
	export class DatabaseQueryAnalysis {
	    databaseConfigId: string;
	    activeQueries: DatabaseActiveQuery[];
	    slowQueries: DatabaseSlowQuery[];
	    sourceNote: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseQueryAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.databaseConfigId = source["databaseConfigId"];
	        this.activeQueries = this.convertValues(source["activeQueries"], DatabaseActiveQuery);
	        this.slowQueries = this.convertValues(source["slowQueries"], DatabaseSlowQuery);
	        this.sourceNote = source["sourceNote"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DatabaseSchemaItem {
	    name: string;
	    tableCount: number;
	    sizeBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseSchemaItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.tableCount = source["tableCount"];
	        this.sizeBytes = source["sizeBytes"];
	    }
	}
	
	export class DatabaseTableItem {
	    tableName: string;
	    engine: string;
	    rowCount: number;
	    dataBytes: number;
	    indexBytes: number;
	    totalBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseTableItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tableName = source["tableName"];
	        this.engine = source["engine"];
	        this.rowCount = source["rowCount"];
	        this.dataBytes = source["dataBytes"];
	        this.indexBytes = source["indexBytes"];
	        this.totalBytes = source["totalBytes"];
	    }
	}
	export class SSHAttackerIP {
	    ip: string;
	    attempts: number;
	
	    static createFrom(source: any = {}) {
	        return new SSHAttackerIP(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.attempts = source["attempts"];
	    }
	}
	export class SecurityAudit {
	    pendingUpdates: number;
	    securityUpdates: number;
	    packageManager: string;
	    failedSshCount: number;
	    topAttackers: SSHAttackerIP[];
	    activeSessions: ActiveSSHSession[];
	    permitRootLogin: string;
	    passwordAuth: string;
	    fail2banActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SecurityAudit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pendingUpdates = source["pendingUpdates"];
	        this.securityUpdates = source["securityUpdates"];
	        this.packageManager = source["packageManager"];
	        this.failedSshCount = source["failedSshCount"];
	        this.topAttackers = this.convertValues(source["topAttackers"], SSHAttackerIP);
	        this.activeSessions = this.convertValues(source["activeSessions"], ActiveSSHSession);
	        this.permitRootLogin = source["permitRootLogin"];
	        this.passwordAuth = source["passwordAuth"];
	        this.fail2banActive = source["fail2banActive"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FirewallStatus {
	    installed: boolean;
	    active: boolean;
	    toolName: string;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new FirewallStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.active = source["active"];
	        this.toolName = source["toolName"];
	        this.summary = source["summary"];
	    }
	}
	export class ListeningPort {
	    protocol: string;
	    localAddress: string;
	    port: number;
	    process: string;
	    pid: number;
	    exposure: string;
	
	    static createFrom(source: any = {}) {
	        return new ListeningPort(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.protocol = source["protocol"];
	        this.localAddress = source["localAddress"];
	        this.port = source["port"];
	        this.process = source["process"];
	        this.pid = source["pid"];
	        this.exposure = source["exposure"];
	    }
	}
	export class SystemdService {
	    name: string;
	    loadState: string;
	    activeState: string;
	    subState: string;
	    description: string;
	    isKeyService: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SystemdService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.loadState = source["loadState"];
	        this.activeState = source["activeState"];
	        this.subState = source["subState"];
	        this.description = source["description"];
	        this.isKeyService = source["isKeyService"];
	    }
	}
	export class DockerContainer {
	    id: string;
	    name: string;
	    image: string;
	    state: string;
	    status: string;
	    ports: string;
	    cpuPercent: number;
	    memUsage: string;
	    memPercent: number;
	    netIO: string;
	    blockIO: string;
	
	    static createFrom(source: any = {}) {
	        return new DockerContainer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.image = source["image"];
	        this.state = source["state"];
	        this.status = source["status"];
	        this.ports = source["ports"];
	        this.cpuPercent = source["cpuPercent"];
	        this.memUsage = source["memUsage"];
	        this.memPercent = source["memPercent"];
	        this.netIO = source["netIO"];
	        this.blockIO = source["blockIO"];
	    }
	}
	export class DevOpsSnapshot {
	    serverId: string;
	    // Go type: time
	    timestamp: any;
	    dockerAvailable: boolean;
	    dockerVersion: string;
	    containers: DockerContainer[];
	    services: SystemdService[];
	    listeningPorts: ListeningPort[];
	    firewall: FirewallStatus;
	    security: SecurityAudit;
	
	    static createFrom(source: any = {}) {
	        return new DevOpsSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverId = source["serverId"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.dockerAvailable = source["dockerAvailable"];
	        this.dockerVersion = source["dockerVersion"];
	        this.containers = this.convertValues(source["containers"], DockerContainer);
	        this.services = this.convertValues(source["services"], SystemdService);
	        this.listeningPorts = this.convertValues(source["listeningPorts"], ListeningPort);
	        this.firewall = this.convertValues(source["firewall"], FirewallStatus);
	        this.security = this.convertValues(source["security"], SecurityAudit);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DiscoveredDatabase {
	    type: string;
	    name: string;
	    host: string;
	    port: number;
	    socketPath?: string;
	    serviceState: string;
	    version?: string;
	    defaultUser: string;
	    databases?: string[];
	
	    static createFrom(source: any = {}) {
	        return new DiscoveredDatabase(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.socketPath = source["socketPath"];
	        this.serviceState = source["serviceState"];
	        this.version = source["version"];
	        this.defaultUser = source["defaultUser"];
	        this.databases = source["databases"];
	    }
	}
	export class DiskPartition {
	    mountPoint: string;
	    device: string;
	    fsType: string;
	    totalBytes: number;
	    usedBytes: number;
	    freeBytes: number;
	    usedPercent: number;
	
	    static createFrom(source: any = {}) {
	        return new DiskPartition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mountPoint = source["mountPoint"];
	        this.device = source["device"];
	        this.fsType = source["fsType"];
	        this.totalBytes = source["totalBytes"];
	        this.usedBytes = source["usedBytes"];
	        this.freeBytes = source["freeBytes"];
	        this.usedPercent = source["usedPercent"];
	    }
	}
	
	
	
	export class MemStats {
	    totalBytes: number;
	    usedBytes: number;
	    freeBytes: number;
	    availableBytes: number;
	    cachedBytes: number;
	    swapTotalBytes: number;
	    swapUsedBytes: number;
	    usedPercent: number;
	    swapUsedPercent: number;
	
	    static createFrom(source: any = {}) {
	        return new MemStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalBytes = source["totalBytes"];
	        this.usedBytes = source["usedBytes"];
	        this.freeBytes = source["freeBytes"];
	        this.availableBytes = source["availableBytes"];
	        this.cachedBytes = source["cachedBytes"];
	        this.swapTotalBytes = source["swapTotalBytes"];
	        this.swapUsedBytes = source["swapUsedBytes"];
	        this.usedPercent = source["usedPercent"];
	        this.swapUsedPercent = source["swapUsedPercent"];
	    }
	}
	export class NetInterface {
	    name: string;
	    rxBytesSec: number;
	    txBytesSec: number;
	    totalRxBytes: number;
	    totalTxBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new NetInterface(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.rxBytesSec = source["rxBytesSec"];
	        this.txBytesSec = source["txBytesSec"];
	        this.totalRxBytes = source["totalRxBytes"];
	        this.totalTxBytes = source["totalTxBytes"];
	    }
	}
	
	export class ProcessItem {
	    pid: number;
	    user: string;
	    cpuPercent: number;
	    memPercent: number;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.user = source["user"];
	        this.cpuPercent = source["cpuPercent"];
	        this.memPercent = source["memPercent"];
	        this.command = source["command"];
	    }
	}
	
	
	
	export class Server {
	    id: string;
	    name: string;
	    host: string;
	    port: number;
	    user: string;
	    authType: string;
	    keyPath: string;
	    jumpHostId: string;
	    tags: string[];
	    notes: string;
	    status: string;
	    statusReason: string;
	    // Go type: time
	    lastCheck?: any;
	    hostKeyFingerprint: string;
	    trustedHostKey: boolean;
	    intervalSec: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.authType = source["authType"];
	        this.keyPath = source["keyPath"];
	        this.jumpHostId = source["jumpHostId"];
	        this.tags = source["tags"];
	        this.notes = source["notes"];
	        this.status = source["status"];
	        this.statusReason = source["statusReason"];
	        this.lastCheck = this.convertValues(source["lastCheck"], null);
	        this.hostKeyFingerprint = source["hostKeyFingerprint"];
	        this.trustedHostKey = source["trustedHostKey"];
	        this.intervalSec = source["intervalSec"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ServerConfig {
	    id: string;
	    name: string;
	    host: string;
	    port: number;
	    user: string;
	    authType: string;
	    keyPath: string;
	    passphrase?: string;
	    password?: string;
	    jumpHostId: string;
	    tags: string[];
	    notes: string;
	    intervalSec: number;
	
	    static createFrom(source: any = {}) {
	        return new ServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.authType = source["authType"];
	        this.keyPath = source["keyPath"];
	        this.passphrase = source["passphrase"];
	        this.password = source["password"];
	        this.jumpHostId = source["jumpHostId"];
	        this.tags = source["tags"];
	        this.notes = source["notes"];
	        this.intervalSec = source["intervalSec"];
	    }
	}
	export class ServiceItem {
	    name: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	    }
	}
	export class ServerMetrics {
	    serverId: string;
	    // Go type: time
	    timestamp: any;
	    cpu: CPUStats;
	    memory: MemStats;
	    disks: DiskPartition[];
	    networks: NetInterface[];
	    processes: ProcessItem[];
	    services: ServiceItem[];
	    uptimeSeconds: number;
	    osInfo: string;
	    kernelInfo: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverId = source["serverId"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.cpu = this.convertValues(source["cpu"], CPUStats);
	        this.memory = this.convertValues(source["memory"], MemStats);
	        this.disks = this.convertValues(source["disks"], DiskPartition);
	        this.networks = this.convertValues(source["networks"], NetInterface);
	        this.processes = this.convertValues(source["processes"], ProcessItem);
	        this.services = this.convertValues(source["services"], ServiceItem);
	        this.uptimeSeconds = source["uptimeSeconds"];
	        this.osInfo = source["osInfo"];
	        this.kernelInfo = source["kernelInfo"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class UptimeMonitor {
	    id: string;
	    name: string;
	    type: string;
	    target: string;
	    serverId: string;
	    intervalSec: number;
	    enabled: boolean;
	    status: string;
	    statusCode: number;
	    latencyMs: number;
	    sslExpiryDays: number;
	    sslIssuer: string;
	    sslValidUntil: string;
	    lastError: string;
	    // Go type: time
	    lastCheck?: any;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new UptimeMonitor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.target = source["target"];
	        this.serverId = source["serverId"];
	        this.intervalSec = source["intervalSec"];
	        this.enabled = source["enabled"];
	        this.status = source["status"];
	        this.statusCode = source["statusCode"];
	        this.latencyMs = source["latencyMs"];
	        this.sslExpiryDays = source["sslExpiryDays"];
	        this.sslIssuer = source["sslIssuer"];
	        this.sslValidUntil = source["sslValidUntil"];
	        this.lastError = source["lastError"];
	        this.lastCheck = this.convertValues(source["lastCheck"], null);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

