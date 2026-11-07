<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    GetServers, GetServer, SaveServer, DeleteServer, TestConnection,
    TrustHostKey, ImportSSHConfig, GetLatestMetrics, GetMetricsHistory,
    GetAlertRules, ToggleAlertRule, GetAlertEvents, GetActiveAlerts,
    TailLog, RunSSHCommand,
    StartSSHTerminal, SendTerminalInput, ResizeSSHTerminal, CloseSSHTerminal, BroadcastSSHCommand,
    GetDatabaseConfigs, SaveDatabaseConfig, DeleteDatabaseConfig,
    GetDatabaseMetrics, PollDatabaseMetrics, AutoDetectDatabases, CancelDatabaseScan, TestDatabaseConnection, ListDatabases,
    AnalyzeDatabaseQueries, ListDatabaseTables, KillDatabaseQuery,
    GetBackupSchedules, SaveBackupSchedule, DeleteBackupSchedule,
    TriggerManualBackup, GetBackupLogs, GetServerBackupLogs, GetActiveBackupJobs,
    VerifyBackupArchive, DownloadBackupArchive, UploadBackupToS3, RestoreDatabaseFromBackup,
    SelectBackupFolder, ExportReport
  } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import type { model } from '../wailsjs/go/models';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import '@xterm/xterm/css/xterm.css';
  import DevOpsView from './DevOpsView.svelte';
  import UptimeView from './UptimeView.svelte';
  import SettingsView from './SettingsView.svelte';

  // Navigation
  let currentNav: 'server' | 'detail' | 'devops' | 'uptime' | 'alert' | 'term' | 'log' | 'database' | 'settings' = 'server';
  let selectedServerId: string | null = null;
  let selectedServer: model.Server | null = null;
  let devopsServerId: string = '';
  let serverMetricsMap: Record<string, model.ServerMetrics> = {};
  let liveToast: { title: string; body: string; level: string; time: string } | null = null;

  // Server List
  let servers: model.Server[] = [];
  let filterStatus: string = 'all';
  let searchQuery: string = '';
  let activeTag: string | null = null;

  // Alert State
  let alertRules: model.AlertRule[] = [];
  let alertEvents: model.AlertEvent[] = [];
  let activeAlerts: model.AlertEvent[] = [];
  let alertTab: 'events' | 'rules' = 'events';
  let alertFilter: 'all' | 'active' | 'resolved' = 'all';
  let isLoadingAlerts = false;

  // Terminal & Log State
  let termServerId: string = '';
  let termTab: 'pty' | 'broadcast' | 'terminal' | 'log' = 'pty';
  let termCommand: string = '';
  let termHistory: Array<{ cmd: string; output: string; time: string; error?: boolean }> = [];
  let isRunningCmd = false;
  let termContainer: HTMLElement | null = null;

  // Interactive PTY Terminal (xterm.js)
  let ptyContainer: HTMLElement | null = null;
  let ptyTerm: Terminal | null = null;
  let ptyFitAddon: FitAddon | null = null;
  let ptySessionId: string | null = null;
  let ptyStatus: 'disconnected' | 'connecting' | 'connected' = 'disconnected';
  let ptyError: string | null = null;
  let ptyUnsubData: (() => void) | null = null;
  let ptyUnsubExit: (() => void) | null = null;
  let ptyResizeObserver: ResizeObserver | null = null;

  // Multi-Server Broadcast SSH
  let broadcastSelectedServerIds: string[] = [];
  let broadcastFilterTag: string = 'all';
  let broadcastCmd: string = 'uptime';
  let isBroadcasting: boolean = false;
  let broadcastResults: model.BroadcastResult[] = [];
  let broadcastError: string | null = null;

  $: availableBroadcastTags = Array.from(new Set(servers.flatMap(s => s.tags || []))).filter(Boolean);

  let logTarget: string = 'syslog';
  let logOutput: string = '';
  let isLoadingLog = false;
  let logSearchQuery: string = '';

  // Database & Backup State
  let dbServerId: string = '';
  let dbActiveTab: 'overview' | 'queries' | 'schedules' | 'logs' = 'overview';
  let databaseConfigs: model.DatabaseConfig[] = [];
  let dbMetricsMap: Record<string, model.DatabaseMetrics> = {};
  let backupSchedules: model.BackupSchedule[] = [];
  let backupLogs: model.BackupJobLog[] = [];
  let isLoadingDB = false;
  let isBackingUp: Record<string, boolean> = {};

  // Background Database Backup Progress State
  interface BackupProgress {
    jobId: number;
    serverId: string;
    serverName: string;
    databaseConfigId: string;
    databaseName: string;
    storageTarget: string;
    stage: string;
    message: string;
    progressPercent: number;
    startedAt: string;
    elapsedSec: number;
    fileSizeBytes: number;
    filePath: string;
    errorMessage: string;
  }
  let activeBackupJobs: Record<number, BackupProgress> = {};
  let showActiveBackupsModal = false;
  $: activeBackupList = Object.values(activeBackupJobs);
  $: activeBackupCount = activeBackupList.length;

  // Slow Query & Active Process Explorer State
  let selectedQueryDbId: string = '';
  let queryAnalysis: model.DatabaseQueryAnalysis | null = null;
  let isLoadingQueries = false;
  let queryError = '';
  let killingQueryPid: Record<string, boolean> = {};

  // Backup Archive Verification, S3 Upload, Download & Restore Wizard State
  let verifyResultMap: Record<number, model.BackupVerifyResult> = {};
  let isVerifyingBackup: Record<number, boolean> = {};
  let isUploadingS3: Record<number, boolean> = {};
  let isDownloadingBackup: Record<number, boolean> = {};
  let showRestoreModal = false;
  let restoreLogItem: model.BackupJobLog | null = null;
  let restoreTargetMode: 'inplace' | 'custom' = 'inplace';
  let restoreCustomDBName = '';
  let isRestoringDB = false;

  // Add DB Modal State
  let showDBModal = false;
  let dbFormData: model.DatabaseConfig = {
    id: '',
    serverId: '',
    type: 'mysql',
    name: '',
    host: '127.0.0.1',
    port: 3306,
    user: 'root',
    databaseName: '',
    enabled: true,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString()
  } as any;
  let dbPasswordInput = '';
  let isEditingDB = false;
  let isTestingDB = false;
  let dbTestResult: { ok: boolean; msg: string } | null = null;

  // Database Schema Explorer State
  let showSchemasModal = false;
  let isLoadingSchemas = false;
  let currentSchemas: model.DatabaseSchemaItem[] = [];
  let currentDbForSchemas: model.DatabaseConfig | null = null;
  let selectedSchemaForTables: string = '';
  let schemaTables: model.DatabaseTableItem[] = [];
  let isLoadingTables = false;

  // Add Backup Schedule Modal State
  let showScheduleModal = false;
  let scheduleFormData: model.BackupSchedule = {
    id: '',
    serverId: '',
    databaseConfigId: '',
    databaseName: '',
    frequency: 'daily',
    timeOfDay: '02:00',
    dayOfWeek: 0,
    retentionDays: 7,
    storageTarget: 'remote',
    remotePath: '/var/backups/serverwatch',
    localPath: '',
    compression: 'gzip',
    enabled: true,
    createdAt: new Date().toISOString()
  } as any;

  // Auto-Detect Database State
  let isDetectingDB = false;
  let showDetectedModal = false;
  let detectedDatabases: model.DiscoveredDatabase[] = [];
  let detectedForServerId = '';
  let detectReqId = 0;
  let loadDbReqId = 0;
  let detectedDbCache: Record<string, model.DiscoveredDatabase[]> = {};

  // Add/Edit Server Modal
  let showModal = false;
  let isEditing = false;
  let isTesting = false;
  let isSavingServer = false;
  let testResult: model.ConnectionTestResult | null = null;
  let formData: model.ServerConfig = {
    id: '',
    name: '',
    host: '',
    port: 22,
    user: 'deploy',
    authType: 'key' as model.SSHAuthType,
    keyPath: '',
    passphrase: '',
    password: '',
    jumpHostId: '',
    tags: ['production'],
    notes: '',
    intervalSec: 10
  };
  let tagInput = 'production';

  // Import SSH Config Modal
  let showImportModal = false;
  let importedHosts: model.ServerConfig[] = [];
  let selectedImportIndices: Record<number, boolean> = {};

  onMount(async () => {
    await loadServers();
    await loadAlerts();

    if (servers.length > 0 && !termServerId) {
      termServerId = servers[0].id;
    }
    if (servers.length > 0 && !dbServerId) {
      dbServerId = servers[0].id;
    }

    // Listen to real-time events from Go backend
    try {
      EventsOn("metrics:update", (serverId: string, metrics: model.ServerMetrics) => {
        serverMetricsMap[serverId] = metrics;
        serverMetricsMap = { ...serverMetricsMap };

        let newStatus = 'normal';
        let newReason = '';
        const maxDisk = metrics.disks && metrics.disks.length > 0
          ? Math.max(...metrics.disks.map(d => d.usedPercent))
          : 0;
        if (maxDisk >= 90) {
          newStatus = 'critical';
          newReason = `Disk usage is critical (${maxDisk.toFixed(1)}%)`;
        } else if (metrics.cpu.totalPercent >= 90) {
          newStatus = 'warning';
          newReason = `CPU load is high (${metrics.cpu.totalPercent.toFixed(1)}%)`;
        } else if (metrics.memory.usedPercent >= 90) {
          newStatus = 'warning';
          newReason = `Memory usage is high (${metrics.memory.usedPercent.toFixed(1)}%)`;
        }

        // Update server item in list
        const idx = servers.findIndex(s => s.id === serverId);
        if (idx !== -1) {
          servers[idx].lastCheck = metrics.timestamp as any;
          servers[idx].status = newStatus as any;
          servers[idx].statusReason = newReason;
          servers = [...servers];
        }
        if (selectedServer && selectedServer.id === serverId) {
          selectedServer.lastCheck = metrics.timestamp as any;
          selectedServer.status = newStatus as any;
          selectedServer.statusReason = newReason;
          selectedServer = { ...selectedServer };
        }
      });

      EventsOn("server:error", (serverId: string, errMsg: string) => {
        const idx = servers.findIndex(s => s.id === serverId);
        if (idx !== -1) {
          servers[idx].status = 'unreachable' as any;
          servers[idx].statusReason = errMsg;
          servers = [...servers];
        }
        if (selectedServer && selectedServer.id === serverId) {
          selectedServer.status = 'unreachable' as any;
          selectedServer.statusReason = errMsg;
          selectedServer = { ...selectedServer };
        }
      });

      EventsOn("dbmetrics:update", (dbId: string, metrics: model.DatabaseMetrics) => {
        dbMetricsMap[dbId] = metrics;
        dbMetricsMap = { ...dbMetricsMap };
      });

      // Background Backup Events
      EventsOn("backup:progress", (evt: BackupProgress) => {
        if (!evt) return;
        if (evt.stage === 'completed' || evt.stage === 'failed') {
          delete activeBackupJobs[evt.jobId];
          activeBackupJobs = { ...activeBackupJobs };
          if (dbServerId) {
            loadDatabaseData();
          }
        } else {
          activeBackupJobs[evt.jobId] = evt;
          activeBackupJobs = { ...activeBackupJobs };
        }
      });

      EventsOn("backup:completed", (log: model.BackupJobLog) => {
        if (log && log.id) {
          delete activeBackupJobs[log.id];
          activeBackupJobs = { ...activeBackupJobs };
        }
        backupLogs = [log, ...backupLogs];
        liveToast = {
          title: "Backup Database Selesai",
          body: `Database ${log.databaseName || ''} berhasil dibackup (${(log.fileSizeBytes / (1024 * 1024)).toFixed(2)} MB)`,
          level: "info",
          time: new Date().toLocaleTimeString()
        };
        setTimeout(() => { liveToast = null; }, 6000);
      });

      EventsOn("backup:failed", (log: model.BackupJobLog) => {
        if (log && log.id) {
          delete activeBackupJobs[log.id];
          activeBackupJobs = { ...activeBackupJobs };
        }
        if (log) {
          backupLogs = [log, ...backupLogs];
        }
        liveToast = {
          title: "Backup Database Gagal",
          body: log?.errorMessage || "Terjadi kegagalan saat menjalankan backup di latar belakang.",
          level: "critical",
          time: new Date().toLocaleTimeString()
        };
        setTimeout(() => { liveToast = null; }, 8000);
      });

      EventsOn("alert:notification", (payload: { title: string; body: string; level: string; time: string }) => {
        if (payload) {
          liveToast = payload;
          loadAlerts();
          setTimeout(() => {
            if (liveToast === payload) {
              liveToast = null;
            }
          }, 7000);
        }
      });

      // Restore active background backup jobs if any
      try {
        const runningJobs = await GetActiveBackupJobs() || [];
        for (const rj of runningJobs) {
          if (rj.status === 'running') {
            activeBackupJobs[rj.id] = {
              jobId: rj.id,
              serverId: rj.serverId,
              serverName: (servers.find(s => s.id === rj.serverId)?.name) || 'Server',
              databaseConfigId: rj.databaseConfigId,
              databaseName: rj.databaseName,
              storageTarget: 'remote',
              stage: 'dumping',
              message: `Sedang memproses backup database '${rj.databaseName}' di latar belakang...`,
              progressPercent: 45,
              startedAt: String(rj.startedAt),
              elapsedSec: 0,
              fileSizeBytes: 0,
              filePath: '',
              errorMessage: ''
            };
          }
        }
        activeBackupJobs = { ...activeBackupJobs };
      } catch (errActive) {
        console.warn("GetActiveBackupJobs init:", errActive);
      }
    } catch (e) {
      console.warn("EventsOn init:", e);
    }
  });

  async function loadServers() {
    try {
      servers = await GetServers() || [];
      if (servers.length > 0 && !termServerId) {
        termServerId = servers[0].id;
      }
      if (servers.length > 0 && !dbServerId) {
        dbServerId = servers[0].id;
      }
      if (selectedServerId) {
        const found = servers.find(s => s.id === selectedServerId);
        if (found) {
          selectedServer = found;
        }
      }
      for (const srv of servers) {
        GetLatestMetrics(srv.id).then(m => {
          if (m) {
            serverMetricsMap[srv.id] = m;
            serverMetricsMap = { ...serverMetricsMap };
          }
        }).catch(() => {});
      }
    } catch (e) {
      console.error("Failed to load servers:", e);
    }
  }

  async function loadAlerts() {
    isLoadingAlerts = true;
    try {
      const [rules, events, active] = await Promise.all([
        GetAlertRules().catch(() => []),
        GetAlertEvents(50).catch(() => []),
        GetActiveAlerts().catch(() => [])
      ]);
      alertRules = rules || [];
      alertEvents = events || [];
      activeAlerts = active || [];
    } catch (e) {
      console.error("Failed to load alerts:", e);
    } finally {
      isLoadingAlerts = false;
    }
  }

  async function handleToggleRule(rule: model.AlertRule) {
    try {
      await ToggleAlertRule(rule.id, !rule.enabled);
      alertRules = await GetAlertRules() || [];
    } catch (err) {
      alert("Gagal mengubah status aturan alert: " + err);
    }
  }

  // Terminal Execution
  async function executeSSHCommand(cmdToRun?: string) {
    const cmd = (cmdToRun || termCommand).trim();
    if (!cmd) return;
    if (!termServerId) {
      alert("Silakan pilih server target terlebih dahulu.");
      return;
    }

    isRunningCmd = true;
    const timeStr = new Date().toLocaleTimeString();
    try {
      const output = await RunSSHCommand(termServerId, cmd);
      termHistory = [...termHistory, { cmd, output, time: timeStr, error: false }];
      termCommand = '';
    } catch (err: any) {
      termHistory = [...termHistory, { cmd, output: err?.message || String(err), time: timeStr, error: true }];
    } finally {
      isRunningCmd = false;
      setTimeout(() => {
        if (termContainer) {
          termContainer.scrollTop = termContainer.scrollHeight;
        }
      }, 50);
    }
  }

  function clearTerminal() {
    termHistory = [];
  }

  // Interactive PTY Terminal Methods (xterm.js + Go SSH PTY channel)
  async function connectPtyTerminal(targetServerId?: string) {
    const sId = targetServerId || termServerId;
    if (!sId) {
      alert("Silakan pilih target server terlebih dahulu.");
      return;
    }
    await disconnectPtyTerminal();

    ptyStatus = 'connecting';
    ptyError = null;

    // Small delay to ensure container element is mounted
    await new Promise(r => setTimeout(r, 60));

    if (!ptyContainer) {
      ptyStatus = 'disconnected';
      ptyError = 'Elemen viewport terminal belum siap.';
      return;
    }

    try {
      ptyTerm = new Terminal({
        cursorBlink: true,
        cursorStyle: 'block',
        fontFamily: 'Cascadia Mono, Consolas, "Courier New", monospace',
        fontSize: 13,
        lineHeight: 1.25,
        theme: {
          background: '#0d1117',
          foreground: '#e6edf3',
          cursor: '#58a6ff',
          cursorAccent: '#0d1117',
          selectionBackground: 'rgba(88, 166, 255, 0.35)',
          black: '#161b22',
          red: '#f85149',
          green: '#3fb950',
          yellow: '#d29922',
          blue: '#58a6ff',
          magenta: '#bc8cff',
          cyan: '#39c5cf',
          white: '#b1bac4',
          brightBlack: '#6e7681',
          brightRed: '#ff7b72',
          brightGreen: '#56d364',
          brightYellow: '#e3b341',
          brightBlue: '#79c0ff',
          brightMagenta: '#d2a8ff',
          brightCyan: '#56d4dd',
          brightWhite: '#ffffff'
        }
      });

      ptyFitAddon = new FitAddon();
      ptyTerm.loadAddon(ptyFitAddon);
      ptyTerm.open(ptyContainer);
      try {
        ptyFitAddon.fit();
      } catch (e) {}

      const cols = ptyTerm.cols || 100;
      const rows = ptyTerm.rows || 30;

      const sid = await StartSSHTerminal(sId, cols, rows);
      ptySessionId = sid;
      ptyStatus = 'connected';

      ptyUnsubData = EventsOn(`terminal:data:${sid}`, (chunk: string) => {
        ptyTerm?.write(chunk);
      });

      ptyUnsubExit = EventsOn(`terminal:exit:${sid}`, () => {
        ptyTerm?.write('\r\n\x1b[33m--- [Sesi SSH terputus / shell ditutup] ---\x1b[0m\r\n');
        disconnectPtyTerminal(false);
      });

      ptyTerm.onData((data: string) => {
        if (ptySessionId) {
          SendTerminalInput(ptySessionId, data).catch(() => {});
        }
      });

      ptyTerm.onResize(({ cols, rows }) => {
        if (ptySessionId) {
          ResizeSSHTerminal(ptySessionId, cols, rows).catch(() => {});
        }
      });

      if (window.ResizeObserver && ptyContainer) {
        ptyResizeObserver = new ResizeObserver(() => {
          try {
            if (ptyFitAddon && ptyTerm) {
              ptyFitAddon.fit();
            }
          } catch (e) {}
        });
        ptyResizeObserver.observe(ptyContainer);
      }

      ptyTerm.focus();
    } catch (err: any) {
      ptyStatus = 'disconnected';
      ptyError = err?.message || String(err);
      if (ptyTerm) {
        ptyTerm.dispose();
        ptyTerm = null;
      }
    }
  }

  async function disconnectPtyTerminal(callBackend = true) {
    if (ptyResizeObserver) {
      ptyResizeObserver.disconnect();
      ptyResizeObserver = null;
    }
    if (ptyUnsubData) {
      ptyUnsubData();
      ptyUnsubData = null;
    }
    if (ptyUnsubExit) {
      ptyUnsubExit();
      ptyUnsubExit = null;
    }
    if (ptySessionId && callBackend) {
      try {
        await CloseSSHTerminal(ptySessionId);
      } catch (e) {}
    }
    ptySessionId = null;
    ptyStatus = 'disconnected';
    if (ptyTerm) {
      ptyTerm.dispose();
      ptyTerm = null;
    }
    ptyFitAddon = null;
  }

  function sendPtyInput(input: string) {
    if (ptySessionId && ptyStatus === 'connected') {
      SendTerminalInput(ptySessionId, input).catch(() => {});
      ptyTerm?.focus();
    }
  }

  function clearPtyTerminal() {
    if (ptyTerm) {
      ptyTerm.clear();
      ptyTerm.focus();
    }
  }

  onDestroy(async () => {
    await disconnectPtyTerminal();
  });

  // Multi-Server Broadcast Methods
  function selectBroadcastTag(tag: string) {
    broadcastFilterTag = tag;
    if (tag === 'all') {
      broadcastSelectedServerIds = servers.map(s => s.id);
    } else {
      broadcastSelectedServerIds = servers.filter(s => (s.tags || []).includes(tag)).map(s => s.id);
    }
  }

  function toggleBroadcastServer(id: string) {
    if (broadcastSelectedServerIds.includes(id)) {
      broadcastSelectedServerIds = broadcastSelectedServerIds.filter(x => x !== id);
    } else {
      broadcastSelectedServerIds = [...broadcastSelectedServerIds, id];
    }
  }

  function toggleAllBroadcastServers() {
    if (broadcastSelectedServerIds.length === servers.length) {
      broadcastSelectedServerIds = [];
    } else {
      broadcastSelectedServerIds = servers.map(s => s.id);
    }
  }

  async function executeBroadcast(cmdToRun?: string) {
    const cmd = (cmdToRun || broadcastCmd).trim();
    if (cmdToRun) broadcastCmd = cmdToRun;
    if (!cmd) {
      alert("Perintah broadcast tidak boleh kosong.");
      return;
    }
    if (broadcastSelectedServerIds.length === 0) {
      alert("Pilih minimal satu server tujuan broadcast.");
      return;
    }

    isBroadcasting = true;
    broadcastError = null;
    broadcastResults = [];

    try {
      broadcastResults = await BroadcastSSHCommand(broadcastSelectedServerIds, cmd) || [];
    } catch (err: any) {
      broadcastError = err?.message || String(err);
    } finally {
      isBroadcasting = false;
    }
  }

  // Log Tail Fetch
  async function fetchLog(customTarget?: string) {
    if (customTarget) {
      logTarget = customTarget;
    }
    if (!termServerId) {
      alert("Silakan pilih server target terlebih dahulu.");
      return;
    }

    isLoadingLog = true;
    try {
      const out = await TailLog(termServerId, logTarget);
      logOutput = out;
    } catch (err: any) {
      logOutput = "Gagal mengambil log: " + (err?.message || String(err));
    } finally {
      isLoadingLog = false;
    }
  }

  function handleDBServerChange() {
    if (isDetectingDB) {
      detectReqId++;
      isDetectingDB = false;
      CancelDatabaseScan().catch(() => {});
    }
    showDetectedModal = false;
    loadDatabaseData();
  }

  async function loadDatabaseData() {
    if (!dbServerId && servers.length > 0) {
      dbServerId = servers[0].id;
    }
    const targetServerId = dbServerId;
    const reqId = ++loadDbReqId;
    isLoadingDB = true;
    try {
      const [configs, schedules, logs] = await Promise.all([
        GetDatabaseConfigs(targetServerId).catch(() => []),
        GetBackupSchedules(targetServerId).catch(() => []),
        GetServerBackupLogs(targetServerId, 50).catch(() => [])
      ]);
      if (reqId !== loadDbReqId || dbServerId !== targetServerId) {
        return;
      }
      databaseConfigs = configs || [];
      backupSchedules = schedules || [];
      backupLogs = logs || [];

      if (databaseConfigs.length > 0 && (!selectedQueryDbId || !databaseConfigs.some(d => d.id === selectedQueryDbId))) {
        selectedQueryDbId = databaseConfigs[0].id;
      } else if (databaseConfigs.length === 0) {
        selectedQueryDbId = '';
        queryAnalysis = null;
      }

      for (const db of databaseConfigs) {
        GetDatabaseMetrics(db.id).then(m => {
          if (m && dbServerId === targetServerId) {
            dbMetricsMap[db.id] = m;
            dbMetricsMap = { ...dbMetricsMap };
          }
        }).catch(() => {});
      }
    } catch (e) {
      console.error("Failed to load database data:", e);
    } finally {
      if (reqId === loadDbReqId) {
        isLoadingDB = false;
      }
    }
  }

  function navigateTo(nav: 'server' | 'detail' | 'devops' | 'uptime' | 'alert' | 'term' | 'log' | 'database' | 'settings') {
    currentNav = nav;
    if (nav === 'alert') {
      loadAlerts();
    }
    if (nav === 'devops' && servers.length > 0 && !devopsServerId) {
      devopsServerId = selectedServerId || servers[0].id;
    }
    if ((nav === 'term' || nav === 'log') && servers.length > 0 && !termServerId) {
      termServerId = servers[0].id;
    }
    if (nav === 'database') {
      if (servers.length > 0 && !dbServerId) {
        dbServerId = servers[0].id;
      }
      loadDatabaseData();
    }
  }

  function openAddDBModal() {
    isEditingDB = false;
    dbFormData = {
      id: '',
      serverId: dbServerId || (servers[0] ? servers[0].id : ''),
      type: 'mysql',
      name: '',
      host: '127.0.0.1',
      port: 3306,
      user: 'root',
      databaseName: '',
      enabled: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    } as any;
    dbPasswordInput = '';
    dbTestResult = null;
    showDBModal = true;
  }

  function openEditDBModal(db: model.DatabaseConfig) {
    isEditingDB = true;
    dbFormData = { ...db } as any;
    dbPasswordInput = '';
    dbTestResult = null;
    showDBModal = true;
  }

  async function handleTestDBConnection() {
    isTestingDB = true;
    dbTestResult = null;
    try {
      dbFormData.serverId = dbServerId || (servers[0] ? servers[0].id : '');
      const res = await TestDatabaseConnection(dbFormData, dbPasswordInput);
      if (res && res.status === 'up') {
        dbTestResult = { ok: true, msg: `Koneksi Berhasil! Uptime: ${res.uptimeSeconds}s | Koneksi Aktif: ${res.activeConnections}` };
      } else {
        dbTestResult = { ok: false, msg: `Status database: ${res ? res.status : 'down'}` };
      }
    } catch (err: any) {
      dbTestResult = { ok: false, msg: `Koneksi Gagal: ${err}` };
    } finally {
      isTestingDB = false;
    }
  }

  async function handleSaveDatabase() {
    if (!dbFormData.name.trim()) {
      alert("Nama konfigurasi database harus diisi.");
      return;
    }
    try {
      dbFormData.serverId = dbServerId || (servers[0] ? servers[0].id : '');
      await SaveDatabaseConfig(dbFormData, dbPasswordInput);
      showDBModal = false;
      await loadDatabaseData();
    } catch (err) {
      alert("Gagal menyimpan database: " + err);
    }
  }

  async function handleDeleteDatabase(id: string) {
    if (confirm("Apakah Anda yakin ingin menghapus database ini beserta jadwal backup-nya?")) {
      try {
        await DeleteDatabaseConfig(id);
        await loadDatabaseData();
      } catch (err) {
        alert("Gagal menghapus database: " + err);
      }
    }
  }

  async function handlePollDBMetrics(dbId: string) {
    try {
      const m = await PollDatabaseMetrics(dbId);
      if (m) {
        dbMetricsMap[dbId] = m;
        dbMetricsMap = { ...dbMetricsMap };
      }
    } catch (err) {
      alert("Gagal mengambil metrik database: " + err);
      await loadDatabaseData();
    }
  }

  async function handleManualBackup(dbId: string, target = 'remote', specificDB = '') {
    const key = specificDB ? `${dbId}_${specificDB}` : dbId;
    isBackingUp[key] = true;
    try {
      const res = await TriggerManualBackup(dbId, target, specificDB);
      showActiveBackupsModal = true;
      liveToast = {
        title: "Backup Berjalan di Latar Belakang",
        body: `Proses backup database '${specificDB || 'utama'}' telah dimulai. Anda dapat menutup modal dan memantau statusnya kapan saja.`,
        level: "info",
        time: new Date().toLocaleTimeString()
      };
      setTimeout(() => {
        if (liveToast && liveToast.title.includes("Backup Berjalan")) {
          liveToast = null;
        }
      }, 5000);
    } catch (err: any) {
      liveToast = {
        title: "Gagal Menjalankan Backup",
        body: String(err?.message || err),
        level: "critical",
        time: new Date().toLocaleTimeString()
      };
    } finally {
      setTimeout(() => {
        isBackingUp[key] = false;
      }, 800);
    }
  }

  function openAddScheduleModal(defaultDbId = '', defaultDbName = '') {
    const selectedDb = databaseConfigs.find(d => d.id === defaultDbId);
    scheduleFormData = {
      id: '',
      serverId: dbServerId || (servers[0] ? servers[0].id : ''),
      databaseConfigId: defaultDbId || (databaseConfigs[0] ? databaseConfigs[0].id : ''),
      databaseName: defaultDbName || (selectedDb ? selectedDb.databaseName : ''),
      frequency: 'daily',
      timeOfDay: '02:00',
      dayOfWeek: 0,
      retentionDays: 7,
      storageTarget: 'remote',
      remotePath: '/var/backups/serverwatch',
      localPath: '',
      compression: 'gzip',
      enabled: true,
      createdAt: new Date().toISOString()
    } as any;
    showScheduleModal = true;
  }

  let schemaSearchQuery = '';
  $: totalSchemasSize = (currentSchemas || []).reduce((acc, s) => acc + (s.sizeBytes || 0), 0);
  $: filteredSchemas = (currentSchemas || []).filter(s => 
    !schemaSearchQuery || s.name.toLowerCase().includes(schemaSearchQuery.toLowerCase())
  );

  function formatBytes(bytes: number): string {
    if (!bytes || bytes <= 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  function isSystemDb(name: string): boolean {
    const sys = ['information_schema', 'performance_schema', 'mysql', 'sys', 'template0', 'template1', 'postgres'];
    return sys.includes(name.toLowerCase());
  }

  function handleScheduleSchema(schemaName: string) {
    if (!currentDbForSchemas) return;
    showSchemasModal = false;
    openAddScheduleModal(currentDbForSchemas.id, schemaName);
  }

  function handleMonitorSchema(schemaName: string) {
    if (!currentDbForSchemas) return;
    isEditingDB = false;
    dbFormData = {
      id: '',
      serverId: currentDbForSchemas.serverId,
      type: currentDbForSchemas.type,
      name: `${currentDbForSchemas.name} (${schemaName})`,
      host: currentDbForSchemas.host,
      port: currentDbForSchemas.port,
      user: currentDbForSchemas.user,
      databaseName: schemaName,
      enabled: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    } as any;
    dbPasswordInput = '';
    showSchemasModal = false;
    showDBModal = true;
  }

  async function handleViewSchemas(db: model.DatabaseConfig) {
    currentDbForSchemas = db;
    selectedSchemaForTables = '';
    schemaTables = [];
    showSchemasModal = true;
    isLoadingSchemas = true;
    schemaSearchQuery = '';
    currentSchemas = [];
    try {
      currentSchemas = await ListDatabases(db.id) || [];
    } catch (err: any) {
      alert("Gagal memuat daftar database: " + err);
    } finally {
      isLoadingSchemas = false;
    }
  }

  async function handleInspectTables(schemaName: string) {
    if (!currentDbForSchemas) return;
    selectedSchemaForTables = schemaName;
    schemaTables = [];
    isLoadingTables = true;
    try {
      schemaTables = await ListDatabaseTables(currentDbForSchemas.id, schemaName) || [];
    } catch (err: any) {
      alert("Gagal memuat tabel untuk schema '" + schemaName + "': " + (err?.message || String(err)));
    } finally {
      isLoadingTables = false;
    }
  }

  async function openQueryExplorer(dbId?: string) {
    if (dbId) {
      selectedQueryDbId = dbId;
    } else if (!selectedQueryDbId && databaseConfigs.length > 0) {
      selectedQueryDbId = databaseConfigs[0].id;
    }
    dbActiveTab = 'queries';
    if (selectedQueryDbId) {
      await loadQueryAnalysis();
    }
  }

  async function loadQueryAnalysis() {
    if (!selectedQueryDbId) return;
    isLoadingQueries = true;
    queryError = '';
    try {
      queryAnalysis = await AnalyzeDatabaseQueries(selectedQueryDbId);
    } catch (err: any) {
      queryError = err?.message || String(err);
    } finally {
      isLoadingQueries = false;
    }
  }

  async function handleKillQuery(pid: string) {
    if (!selectedQueryDbId || !pid) return;
    killingQueryPid[pid] = true;
    try {
      await KillDatabaseQuery(selectedQueryDbId, pid);
      await loadQueryAnalysis();
    } catch (err: any) {
      const msg = err?.message || String(err);
      if (!msg.includes('dibatalkan')) {
        alert("Gagal menghentikan query: " + msg);
      }
    } finally {
      killingQueryPid[pid] = false;
    }
  }

  async function handleVerifyBackup(log: model.BackupJobLog) {
    isVerifyingBackup[log.id] = true;
    try {
      const res = await VerifyBackupArchive(log.id);
      if (res) {
        verifyResultMap[log.id] = res;
        verifyResultMap = { ...verifyResultMap };
      }
    } catch (err: any) {
      alert("Gagal memverifikasi arsip backup: " + (err?.message || String(err)));
    } finally {
      isVerifyingBackup[log.id] = false;
    }
  }

  async function handleDownloadBackup(log: model.BackupJobLog) {
    isDownloadingBackup[log.id] = true;
    try {
      const savedPath = await DownloadBackupArchive(log.id);
      if (savedPath) {
        alert(`Arsip backup berhasil diunduh ke:\n${savedPath}`);
      }
    } catch (err: any) {
      const msg = err?.message || String(err);
      if (!msg.includes('dibatalkan')) {
        alert("Gagal mengunduh file backup: " + msg);
      }
    } finally {
      isDownloadingBackup[log.id] = false;
    }
  }

  async function handleUploadBackupToS3(log: model.BackupJobLog) {
    isUploadingS3[log.id] = true;
    try {
      const s3Uri = await UploadBackupToS3(log.id);
      alert(`Arsip backup berhasil diunggah ke Cloud Object Storage:\n${s3Uri}`);
      await loadDatabaseData();
    } catch (err: any) {
      alert("Gagal mengunggah arsip ke S3/R2: " + (err?.message || String(err)) + "\n\nPastikan kredensial S3/R2 sudah dikonfigurasi di menu Pengaturan -> Offsite Backup Cloud.");
    } finally {
      isUploadingS3[log.id] = false;
    }
  }

  function openRestoreWizard(log: model.BackupJobLog) {
    restoreLogItem = log;
    restoreTargetMode = 'inplace';
    restoreCustomDBName = log.databaseName && log.databaseName !== 'all_databases'
      ? `${log.databaseName}_restored`
      : 'app_staging_restore';
    showRestoreModal = true;
    if (!verifyResultMap[log.id]) {
      handleVerifyBackup(log);
    }
  }

  async function handleExecuteRestore() {
    if (!restoreLogItem) return;
    const targetDB = restoreTargetMode === 'custom'
      ? restoreCustomDBName.trim()
      : restoreLogItem.databaseName;

    if (restoreTargetMode === 'custom' && !targetDB) {
      alert("Nama database tujuan restore tidak boleh kosong.");
      return;
    }

    isRestoringDB = true;
    try {
      await RestoreDatabaseFromBackup(restoreLogItem.id, targetDB);
      showRestoreModal = false;
      alert(`Restore / Rollback selesai!\nArsip '${restoreLogItem.databaseName}' berhasil dipulihkan ke database '${targetDB}'.`);
      await loadDatabaseData();
    } catch (err: any) {
      const msg = err?.message || String(err);
      if (!msg.includes('dibatalkan')) {
        alert("Proses Restore gagal: " + msg);
      }
    } finally {
      isRestoringDB = false;
    }
  }

  async function handleBrowseLocalFolder() {
    try {
      const selectedDir = await SelectBackupFolder();
      if (selectedDir) {
        scheduleFormData.localPath = selectedDir;
      }
    } catch (err) {
      console.error("Gagal memilih folder:", err);
    }
  }

  async function handleSaveSchedule() {
    if (!scheduleFormData.databaseConfigId) {
      alert("Pilih target database terlebih dahulu.");
      return;
    }
    const selectedDb = databaseConfigs.find(d => d.id === scheduleFormData.databaseConfigId);
    if (selectedDb) {
      scheduleFormData.databaseName = selectedDb.databaseName || selectedDb.name;
    }
    scheduleFormData.serverId = dbServerId;
    try {
      await SaveBackupSchedule(scheduleFormData);
      showScheduleModal = false;
      await loadDatabaseData();
    } catch (err) {
      alert("Gagal menyimpan jadwal backup: " + err);
    }
  }

  async function handleDeleteSchedule(id: string) {
    if (confirm("Hapus jadwal backup otomatis ini?")) {
      try {
        await DeleteBackupSchedule(id);
        await loadDatabaseData();
      } catch (err) {
        alert("Gagal menghapus jadwal: " + err);
      }
    }
  }

  function handleCancelAutoDetect() {
    detectReqId++;
    isDetectingDB = false;
    CancelDatabaseScan().catch(() => {});
  }

  async function handleAutoDetect() {
    if (!dbServerId) {
      alert("Silakan pilih server target terlebih dahulu.");
      return;
    }
    const targetServerId = dbServerId;
    const reqId = ++detectReqId;
    isDetectingDB = true;
    try {
      const res = await AutoDetectDatabases(targetServerId) || [];
      if (reqId !== detectReqId || dbServerId !== targetServerId) {
        return;
      }
      detectedDatabases = res;
      detectedForServerId = targetServerId;
      detectedDbCache[targetServerId] = res;
      showDetectedModal = true;
    } catch (err: any) {
      if (reqId !== detectReqId || dbServerId !== targetServerId) {
        return;
      }
      const msg = String(err?.message || err);
      if (!msg.toLowerCase().includes('canceled')) {
        alert("Gagal mendeteksi database: " + msg);
      }
    } finally {
      if (reqId === detectReqId) {
        isDetectingDB = false;
      }
    }
  }

  function handleAddDetectedDB(item: model.DiscoveredDatabase, specificDbName = '') {
    isEditingDB = false;
    const targetSrvId = detectedForServerId || dbServerId || (servers[0] ? servers[0].id : '');
    if (detectedForServerId && dbServerId !== detectedForServerId) {
      dbServerId = detectedForServerId;
      loadDatabaseData();
    }
    dbFormData = {
      id: '',
      serverId: targetSrvId,
      type: item.type,
      name: specificDbName ? `${item.name} (${specificDbName})` : item.name,
      host: item.host,
      port: item.port,
      user: item.defaultUser || 'root',
      databaseName: specificDbName,
      enabled: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    } as any;
    dbPasswordInput = '';
    dbTestResult = null;
    showDetectedModal = false;
    showDBModal = true;
  }

  function openAddModal() {
    isEditing = false;
    testResult = null;
    formData = {
      id: '',
      name: '',
      host: '',
      port: 22,
      user: 'deploy',
      authType: 'key',
      keyPath: '',
      passphrase: '',
      password: '',
      jumpHostId: '',
      tags: ['production'],
      notes: '',
      intervalSec: 10
    };
    tagInput = 'production';
    showModal = true;
  }

  function openEditServerModal(srv: model.Server) {
    isEditing = true;
    testResult = null;
    formData = {
      id: srv.id,
      name: srv.name,
      host: srv.host,
      port: srv.port,
      user: srv.user,
      authType: srv.authType,
      keyPath: srv.keyPath,
      passphrase: '',
      password: '',
      jumpHostId: srv.jumpHostId || '',
      tags: srv.tags || [],
      notes: srv.notes || '',
      intervalSec: srv.intervalSec || 10
    };
    tagInput = (srv.tags || []).join(', ');
    showModal = true;
  }

  async function handleTestConnection() {
    isTesting = true;
    testResult = null;
    formData.tags = tagInput.split(',').map(t => t.trim()).filter(Boolean);
    try {
      testResult = await TestConnection(formData);
    } catch (err: any) {
      testResult = {
        success: false,
        error: err?.message || String(err),
        hasSystemd: false,
        latencyMs: 0
      } as any;
    } finally {
      isTesting = false;
    }
  }

  async function handleSaveServer() {
    if (!formData.name || !formData.name.trim()) {
      alert("Nama server harus diisi.");
      return;
    }
    if (!formData.host || !formData.host.trim()) {
      alert("Host atau IP address server harus diisi.");
      return;
    }
    if (!formData.user || !formData.user.trim()) {
      alert("Username SSH harus diisi.");
      return;
    }

    isSavingServer = true;
    try {
      formData.port = Number(formData.port) || 22;
      formData.intervalSec = Number(formData.intervalSec) || 10;
      formData.tags = tagInput.split(',').map(t => t.trim()).filter(Boolean);
      const saved = await SaveServer(formData);
      if (testResult && testResult.success && testResult.hostKeyFingerprint) {
        await TrustHostKey(saved.id, testResult.hostKeyFingerprint);
      }
      showModal = false;
      await loadServers();
      if (saved && (selectedServerId === saved.id || (selectedServer && selectedServer.id === saved.id))) {
        selectedServer = saved;
      }
    } catch (err) {
      alert("Gagal menyimpan server: " + err);
    } finally {
      isSavingServer = false;
    }
  }

  async function handleDeleteServer(id: string) {
    if (confirm("Apakah Anda yakin ingin menghapus server ini dari monitoring?")) {
      try {
        await DeleteServer(id);
        if (selectedServerId === id) {
          selectedServerId = null;
          selectedServer = null;
          currentNav = 'server';
        }
        await loadServers();
      } catch (err) {
        alert("Gagal menghapus server: " + err);
      }
    }
  }

  async function handleImportSSHConfig() {
    try {
      importedHosts = await ImportSSHConfig() || [];
      selectedImportIndices = {};
      importedHosts.forEach((_, i) => selectedImportIndices[i] = true);
      showImportModal = true;
    } catch (err) {
      alert("Tidak dapat membaca ~/.ssh/config: " + err);
    }
  }

  async function executeImport() {
    for (let i = 0; i < importedHosts.length; i++) {
      if (selectedImportIndices[i]) {
        try {
          await SaveServer(importedHosts[i]);
        } catch (e) {
          console.error("Failed to import host:", importedHosts[i].name, e);
        }
      }
    }
    showImportModal = false;
    await loadServers();
  }

  async function viewServerDetail(srv: model.Server) {
    selectedServerId = srv.id;
    selectedServer = srv;
    currentNav = 'detail';
  }

  // Filtered servers
  $: filteredServers = servers.filter(s => {
    if (filterStatus === 'critical' && s.status !== 'critical') return false;
    if (filterStatus === 'warning' && s.status !== 'warning') return false;
    if (filterStatus === 'unreachable' && s.status !== 'unreachable') return false;
    if (filterStatus === 'normal' && s.status !== 'normal') return false;
    if (activeTag && (!s.tags || !s.tags.includes(activeTag))) return false;
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      const matchName = s.name.toLowerCase().includes(q);
      const matchHost = s.host.toLowerCase().includes(q);
      const matchTag = s.tags && s.tags.some(t => t.toLowerCase().includes(q));
      if (!matchName && !matchHost && !matchTag) return false;
    }
    return true;
  });

  // Attention servers
  $: attentionServers = servers.filter(s => s.status === 'critical' || s.status === 'warning' || s.status === 'unreachable');

  // Available tags
  $: allTags = Array.from(new Set(servers.flatMap(s => s.tags || [])));

  function handleGlobalKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (showModal) showModal = false;
      else if (showImportModal) showImportModal = false;
      else if (showDBModal) showDBModal = false;
      else if (showScheduleModal) showScheduleModal = false;
      else if (showDetectedModal) showDetectedModal = false;
      else if (showSchemasModal) showSchemasModal = false;
      else if (showRestoreModal) showRestoreModal = false;
      else if (showActiveBackupsModal) showActiveBackupsModal = false;
    }
  }
</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<div class="titlebar">
  <b>ServerWatch</b> <span>v0.1.0</span> <span style="color:var(--dim)">• SSH Monitoring</span>
  <span class="spacer"></span>
</div>

<div class="app-layout">
  <!-- Sidebar -->
  <aside class="side">
    <div class="brand">
      <span class="brand-mark">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="4" width="18" height="7" rx="1.5"/><rect x="3" y="13" width="18" height="7" rx="1.5"/><circle cx="7" cy="7.5" r="1"/><circle cx="7" cy="16.5" r="1"/>
        </svg>
      </span>
      ServerWatch
    </div>

    <nav class="nav">
      <button type="button" class="nav-item {currentNav === 'server' ? 'active' : ''}" on:click={() => navigateTo('server')}>
        <svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="7" rx="1.5"/><rect x="3" y="13" width="18" height="7" rx="1.5"/><circle cx="7" cy="7.5" r="1"/><circle cx="7" cy="16.5" r="1"/></svg>
        Server
      </button>
      <button type="button" class="nav-item {currentNav === 'database' ? 'active' : ''}" on:click={() => navigateTo('database')}>
        <svg viewBox="0 0 24 24"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/></svg>
        Database & Backup
        {#if activeBackupCount > 0}
          <span class="count num" style="background:#0284c7; color:#fff" title="{activeBackupCount} backup sedang berjalan">⚡ {activeBackupCount}</span>
        {:else if databaseConfigs.length > 0}
          <span class="count num">{databaseConfigs.length}</span>
        {/if}
      </button>
      <button type="button" class="nav-item {currentNav === 'devops' ? 'active' : ''}" on:click={() => navigateTo('devops')}>
        <svg viewBox="0 0 24 24"><rect x="2" y="7" width="20" height="13" rx="2"/><path d="M6 7V4h4v3M10 7V4h4v3"/></svg>
        DevOps & Ekosistem
      </button>
      <button type="button" class="nav-item {currentNav === 'uptime' ? 'active' : ''}" on:click={() => navigateTo('uptime')}>
        <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M2.5 12h19M12 2.5a14.5 14.5 0 0 1 0 19M12 2.5a14.5 14.5 0 0 0 0 19"/></svg>
        Uptime & SSL
      </button>
      <button type="button" class="nav-item {currentNav === 'alert' ? 'active' : ''}" on:click={() => navigateTo('alert')}>
        <svg viewBox="0 0 24 24"><path d="M12 4 2.5 20h19L12 4z"/><path d="M12 10v4M12 17h.01"/></svg>
        Alert
        {#if activeAlerts.length > 0 || attentionServers.length > 0}
          <span class="count num">{activeAlerts.length || attentionServers.length}</span>
        {/if}
      </button>
      <button type="button" class="nav-item {currentNav === 'term' ? 'active' : ''}" on:click={() => navigateTo('term')}>
        <svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="16" rx="1.5"/><path d="m7 9 3 3-3 3M13 15h4"/></svg>
        Terminal & Log
      </button>
      <button type="button" class="nav-item {currentNav === 'settings' ? 'active' : ''}" on:click={() => navigateTo('settings')}>
        <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/></svg>
        Pengaturan & Laporan
      </button>
    </nav>

    {#if allTags.length > 0}
      <div class="side-sec">Tag</div>
      <nav class="nav tagnav">
        {#each allTags as tag}
          <button type="button" class="nav-item {activeTag === tag ? 'active' : ''}" on:click={() => activeTag = activeTag === tag ? null : tag}>
            <i></i>{tag}
          </button>
        {/each}
      </nav>
    {/if}

    {#if activeBackupCount > 0}
      <div style="margin: 10px 8px 6px 8px; padding: 8px 10px; background: rgba(56, 189, 248, 0.12); border: 1px solid rgba(56, 189, 248, 0.35); border-radius: 6px;">
        <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom: 4px;">
          <span style="font-size: 11px; font-weight: 600; color: #38bdf8; display:flex; align-items:center; gap:5px">
            <span style="display:inline-block; width:6px; height:6px; border-radius:50%; background:#38bdf8; box-shadow:0 0 5px #38bdf8;"></span>
            Backup ({activeBackupCount})
          </span>
          <button class="btn ghost" style="padding: 1px 6px; font-size: 10px; height: auto;" on:click={() => showActiveBackupsModal = true}>Buka</button>
        </div>
        {#each activeBackupList.slice(0, 2) as job}
          <div style="font-size: 10.5px; color: var(--dim); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-top: 2px;">
            {job.databaseName}: {job.progressPercent}%
          </div>
        {/each}
      </div>
    {/if}

    <div class="side-foot">
      <div class="row">
        <span>Server Aktif</span>
        <span class="num">{servers.filter(s => s.status !== 'unreachable').length} / {servers.length}</span>
      </div>
      <div class="row">
        <span>Interval Polling</span>
        <span class="num">10 dtk</span>
      </div>
    </div>
  </aside>

  <!-- Main Area -->
  <main class="main">
    {#if currentNav === 'server'}
      <!-- Dashboard Overview -->
      <div class="topbar">
        <h1>Server</h1>
        <span class="sub">{servers.length} server terdaftar</span>
        <span class="spacer"></span>
        <div class="search">
          <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="6.5"/><path d="m20 20-4.2-4.2"/></svg>
          <input type="text" placeholder="Cari nama, host, atau tag..." bind:value={searchQuery} />
        </div>
        <button class="btn" on:click={handleImportSSHConfig}>
          <svg viewBox="0 0 24 24"><path d="M12 4v11M7 10l5 5 5-5M5 20h14"/></svg>
          Import ~/.ssh/config
        </button>
        <button class="btn primary" on:click={openAddModal}>
          <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
          Tambah server
        </button>
      </div>

      <div class="content">
        <!-- Attention Box -->
        {#if attentionServers.length > 0}
          <div style="margin-bottom:10px; font-weight:600; font-size:13.5px">Perlu Perhatian ({attentionServers.length})</div>
          <div class="attn">
            {#each attentionServers as att}
              <div class="card {att.status === 'critical' ? 'crit' : att.status === 'warning' ? 'warn' : 'off'}">
                <div class="name">
                  <span class="st {att.status === 'critical' ? 'crit' : att.status === 'warning' ? 'warn' : 'off'}"></span>
                  {att.name}
                </div>
                <div class="host">{att.user}@{att.host}:{att.port}</div>
                <div class="why">
                  {#if att.status === 'unreachable'}
                    Koneksi SSH gagal: <b>{att.statusReason || 'Timeout'}</b>
                  {:else}
                    Status: <b>{att.statusReason || att.status}</b>
                  {/if}
                </div>
                <div class="acts">
                  <button class="btn" on:click={() => viewServerDetail(att)}>Buka detail</button>
                  <button class="btn danger" on:click={() => handleDeleteServer(att.id)}>Hapus</button>
                </div>
              </div>
            {/each}
          </div>
        {/if}

        <!-- Filter Chips -->
        <div style="display:flex; gap:8px; align-items:center; margin-bottom:14px">
          <button type="button" class="chip {filterStatus === 'all' ? 'on' : ''}" on:click={() => filterStatus = 'all'}>
            Semua <span class="n">{servers.length}</span>
          </button>
          <button type="button" class="chip {filterStatus === 'critical' ? 'on' : ''}" on:click={() => filterStatus = 'critical'}>
            <span class="st crit"></span> Critical <span class="n">{servers.filter(s => s.status === 'critical').length}</span>
          </button>
          <button type="button" class="chip {filterStatus === 'warning' ? 'on' : ''}" on:click={() => filterStatus = 'warning'}>
            <span class="st warn"></span> Warning <span class="n">{servers.filter(s => s.status === 'warning').length}</span>
          </button>
          <button type="button" class="chip {filterStatus === 'unreachable' ? 'on' : ''}" on:click={() => filterStatus = 'unreachable'}>
            <span class="st off"></span> Gagal SSH <span class="n">{servers.filter(s => s.status === 'unreachable').length}</span>
          </button>
          <button type="button" class="chip {filterStatus === 'normal' ? 'on' : ''}" on:click={() => filterStatus = 'normal'}>
            <span class="st ok"></span> Normal <span class="n">{servers.filter(s => s.status === 'normal').length}</span>
          </button>
        </div>

        <!-- Servers Table -->
        <div class="panel">
          {#if filteredServers.length === 0}
            <div style="padding: 30px; text-align: center; color: var(--muted)">
              Belum ada server yang terdaftar. Klik <b>Tambah server</b> atau <b>Import ~/.ssh/config</b> untuk memulai monitoring.
            </div>
          {:else}
            <table>
              <thead>
                <tr>
                  <th style="width:110px">Status</th>
                  <th>Server</th>
                  <th>Tag</th>
                  <th>CPU</th>
                  <th>RAM</th>
                  <th>Disk</th>
                  <th class="r">Load</th>
                  <th class="r">Uptime</th>
                  <th class="r" style="width:100px">Aksi</th>
                </tr>
              </thead>
              <tbody>
                {#each filteredServers as srv}
                  {@const metrics = serverMetricsMap[srv.id]}
                  <tr>
                    <td>
                      <span class="st {srv.status === 'critical' ? 'crit' : srv.status === 'warning' ? 'warn' : srv.status === 'unreachable' ? 'off' : 'ok'}">
                        {srv.status === 'critical' ? 'Critical' : srv.status === 'warning' ? 'Warning' : srv.status === 'unreachable' ? 'Gagal' : 'Normal'}
                      </span>
                    </td>
                    <td>
                      <button type="button" style="all:unset; cursor:pointer; font-weight:600; color:inherit; text-align:left; display:block" on:click={() => viewServerDetail(srv)}>{srv.name}</button>
                      <div style="color:var(--dim); font-family:var(--mono); font-size:11.5px">{srv.user}@{srv.host}</div>
                    </td>
                    <td>
                      {#if srv.tags && srv.tags.length > 0}
                        {#each srv.tags as t}
                          <span class="tag">{t}</span>
                        {/each}
                      {:else}
                        <span style="color:var(--dim)">-</span>
                      {/if}
                    </td>
                    <td>
                      {#if metrics}
                        <span class="metric">
                          <span class="bar {metrics.cpu.totalPercent > 90 ? 'crit' : metrics.cpu.totalPercent > 70 ? 'warn' : ''}">
                            <i style="width:{Math.min(100, metrics.cpu.totalPercent)}%"></i>
                          </span>
                          <span class="v {metrics.cpu.totalPercent > 90 ? 'crit' : metrics.cpu.totalPercent > 70 ? 'warn' : ''}">
                            {metrics.cpu.totalPercent}%
                          </span>
                        </span>
                      {:else}
                        <span style="color:var(--dim)">...</span>
                      {/if}
                    </td>
                    <td>
                      {#if metrics}
                        <span class="metric">
                          <span class="bar {metrics.memory.usedPercent > 90 ? 'crit' : metrics.memory.usedPercent > 75 ? 'warn' : ''}">
                            <i style="width:{Math.min(100, metrics.memory.usedPercent)}%"></i>
                          </span>
                          <span class="v {metrics.memory.usedPercent > 90 ? 'crit' : metrics.memory.usedPercent > 75 ? 'warn' : ''}">
                            {metrics.memory.usedPercent}%
                          </span>
                        </span>
                      {:else}
                        <span style="color:var(--dim)">...</span>
                      {/if}
                    </td>
                    <td>
                      {#if metrics && metrics.disks && metrics.disks.length > 0}
                        {@const maxD = Math.max(...metrics.disks.map(d => d.usedPercent))}
                        <span class="metric">
                          <span class="bar {maxD > 90 ? 'crit' : maxD > 80 ? 'warn' : ''}">
                            <i style="width:{Math.min(100, maxD)}%"></i>
                          </span>
                          <span class="v {maxD > 90 ? 'crit' : maxD > 80 ? 'warn' : ''}">
                            {maxD.toFixed(0)}%
                          </span>
                        </span>
                      {:else}
                        <span style="color:var(--dim)">-</span>
                      {/if}
                    </td>
                    <td class="r num">
                      {metrics ? metrics.cpu.load1.toFixed(2) : '-'}
                    </td>
                    <td class="r num">
                      {#if metrics && metrics.uptimeSeconds}
                        {Math.floor(metrics.uptimeSeconds / 86400)} hr
                      {:else}
                        -
                      {/if}
                    </td>
                    <td class="r">
                      <button class="btn ghost" on:click={() => openEditServerModal(srv)} title="Edit Konfigurasi Server {srv.name}">Edit</button>
                      <button class="btn ghost" on:click={() => viewServerDetail(srv)} title="Lihat Telemetri & Dasbor Detail Server {srv.name}">Detail</button>
                      <button class="btn ghost danger" on:click={() => handleDeleteServer(srv.id)} title="Hapus Server {srv.name}">✕</button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          {/if}
        </div>
      </div>

    {:else if currentNav === 'detail'}
      <!-- Detailed Server View -->
      {#if selectedServer}
        {@const metrics = serverMetricsMap[selectedServer.id]}
        <div class="topbar">
          <div>
            <button type="button" class="btn ghost" style="padding:0; height:auto; color:var(--dim); font-size:12px; border:none; background:none" on:click={() => currentNav = 'server'}>← Kembali ke Server</button>
            <h1 style="display:flex; align-items:center; gap:8px; margin-top:2px">
              <span class="st {selectedServer.status === 'critical' ? 'crit' : selectedServer.status === 'warning' ? 'warn' : selectedServer.status === 'unreachable' ? 'off' : 'ok'}"></span>
              {selectedServer.name}
              {#each (selectedServer.tags || []) as t}
                <span class="tag">{t}</span>
              {/each}
            </h1>
          </div>
          <span class="spacer"></span>
          <button class="btn primary" on:click={() => { devopsServerId = selectedServer ? selectedServer.id : ''; navigateTo('devops'); }}>
            DevOps & Service
          </button>
          <button class="btn" on:click={() => { if (selectedServer) { termServerId = selectedServer.id; termTab = 'pty'; navigateTo('term'); connectPtyTerminal(selectedServer.id); } }}>
            <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2" style="margin-right:4px; vertical-align:-2px"><rect x="3" y="4" width="18" height="16" rx="1.5"/><path d="m7 9 3 3-3 3M13 15h4"/></svg>
            Terminal PTY
          </button>
          <button class="btn" on:click={() => { if (selectedServer) openEditServerModal(selectedServer); }}>Edit Server</button>
          <button class="btn danger" on:click={() => { if (selectedServer) handleDeleteServer(selectedServer.id); }}>Hapus Server</button>
        </div>

        <div class="content">
          <!-- Server Diagnostic Specs Bar -->
          <div class="panel" style="padding: 12px 18px; margin-bottom: 16px; display:flex; gap:20px; font-size:12.5px; color:var(--muted)">
            <span>Host: <b class="num" style="color:var(--text)">{selectedServer.user}@{selectedServer.host}:{selectedServer.port}</b></span>
            <span>OS: <b style="color:var(--text)">{metrics?.osInfo || 'Linux'}</b></span>
            <span>Kernel: <b style="color:var(--text)">{metrics?.kernelInfo || '-'}</b></span>
            <span>Uptime: <b class="num" style="color:var(--text)">{metrics?.uptimeSeconds ? Math.floor(metrics.uptimeSeconds / 86400) + ' hari' : '-'}</b></span>
          </div>

          <!-- Metrics Grid -->
          <div style="display:grid; grid-template-columns: 1fr 1fr; gap:16px; margin-bottom:16px">
            <!-- CPU Card -->
            <div class="panel" style="padding:16px">
              <div class="panel-h" style="padding:0 0 10px 0">
                <h2>CPU Usage</h2>
                <span class="spacer"></span>
                <span class="num" style="font-size:18px; font-weight:600">{metrics?.cpu?.totalPercent || 0}%</span>
              </div>
              <div style="margin-top:12px">
                <div class="bar" style="width:100%; height:8px">
                  <i style="width:{metrics?.cpu?.totalPercent || 0}%; background:var(--accent)"></i>
                </div>
                <div style="display:flex; justify-content:space-between; margin-top:8px; font-size:12px; color:var(--dim)">
                  <span>Load 1m: <b class="num">{metrics?.cpu?.load1 || 0}</b></span>
                  <span>Load 5m: <b class="num">{metrics?.cpu?.load5 || 0}</b></span>
                  <span>Load 15m: <b class="num">{metrics?.cpu?.load15 || 0}</b></span>
                </div>
              </div>
            </div>

            <!-- Memory Card -->
            <div class="panel" style="padding:16px">
              <div class="panel-h" style="padding:0 0 10px 0">
                <h2>RAM Memory</h2>
                <span class="spacer"></span>
                <span class="num" style="font-size:18px; font-weight:600">{metrics?.memory?.usedPercent || 0}%</span>
              </div>
              <div style="margin-top:12px">
                <div class="bar" style="width:100%; height:8px">
                  <i style="width:{metrics?.memory?.usedPercent || 0}%; background:#8fb8c9"></i>
                </div>
                <div style="display:flex; justify-content:space-between; margin-top:8px; font-size:12px; color:var(--dim)">
                  <span>Terpakai: <b class="num">{metrics?.memory?.usedBytes ? (metrics.memory.usedBytes / 1073741824).toFixed(1) + ' GB' : '-'}</b></span>
                  <span>Total: <b class="num">{metrics?.memory?.totalBytes ? (metrics.memory.totalBytes / 1073741824).toFixed(1) + ' GB' : '-'}</b></span>
                </div>
              </div>
            </div>
          </div>

          <!-- Disks and Top Processes -->
          <div style="display:grid; grid-template-columns: 1fr 1fr; gap:16px">
            <!-- Disks -->
            <div class="panel">
              <div class="panel-h"><h2>Partisi Disk</h2></div>
              {#if metrics && metrics.disks && metrics.disks.length > 0}
                <table>
                  <thead><tr><th>Mount</th><th>Terpakai</th><th class="r">Total</th></tr></thead>
                  <tbody>
                    {#each metrics.disks as d}
                      <tr>
                        <td class="num">{d.mountPoint}</td>
                        <td>
                          <div class="bar {d.usedPercent > 90 ? 'crit' : ''}" style="width:100px">
                            <i style="width:{d.usedPercent}%"></i>
                          </div>
                          <span class="num">{d.usedPercent}%</span>
                        </td>
                        <td class="r num">{(d.totalBytes / 1073741824).toFixed(1)} GB</td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              {:else}
                <div style="padding:20px; color:var(--dim)">Belum ada data disk</div>
              {/if}
            </div>

            <!-- Top Processes -->
            <div class="panel">
              <div class="panel-h"><h2>Proses Teratas (CPU)</h2></div>
              {#if metrics && metrics.processes && metrics.processes.length > 0}
                <table>
                  <thead><tr><th>PID</th><th>Perintah</th><th>User</th><th class="r">CPU%</th></tr></thead>
                  <tbody>
                    {#each metrics.processes as p}
                      <tr>
                        <td class="num">{p.pid}</td>
                        <td class="num" style="max-width:140px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap">{p.command}</td>
                        <td style="color:var(--muted)">{p.user}</td>
                        <td class="r num">{p.cpuPercent}%</td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              {:else}
                <div style="padding:20px; color:var(--dim)">Belum ada data proses</div>
              {/if}
            </div>
          </div>
        </div>
      {/if}

    {:else if currentNav === 'devops'}
      <DevOpsView {servers} bind:selectedServerId={devopsServerId} />

    {:else if currentNav === 'uptime'}
      <UptimeView {servers} />

    {:else if currentNav === 'database'}
      <div class="topbar">
        <h1>Database & Backup Engine</h1>
        <span class="sub">{databaseConfigs.length} database terdaftar</span>
        <span class="spacer"></span>
        <div style="display:flex; align-items:center; gap:8px">
          <span style="font-size:12px; color:var(--muted)">Target Server:</span>
          <select style="height:32px; background:var(--bg); border:1px solid var(--line); color:var(--text); border-radius:5px; padding:0 10px; font-weight:500" bind:value={dbServerId} on:change={handleDBServerChange}>
            {#if servers.length === 0}
              <option value="">Belum ada server terdaftar</option>
            {/if}
            {#each servers as s}
              <option value={s.id}>{s.name} ({s.host})</option>
            {/each}
          </select>
        </div>
        <button
          class="btn"
          style={isDetectingDB ? 'border-color:var(--warn); color:var(--warn)' : ''}
          on:click={isDetectingDB ? handleCancelAutoDetect : handleAutoDetect}
          title={isDetectingDB ? 'Klik untuk membatalkan pemindaian database' : 'Pindai otomatis instans database di server target'}
        >
          {#if isDetectingDB}
            <i>●</i> Memindai... (Batalkan)
          {:else}
            <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
            Deteksi Otomatis
          {/if}
        </button>
        <button class="btn" on:click={() => openAddScheduleModal()}>
          <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          + Jadwal Backup
        </button>
        <button class="btn primary" on:click={openAddDBModal}>
          <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
          + Tambah Database
        </button>
      </div>

      <div class="content">
        <!-- Subnav Tabs -->
        <div class="subnav-tabs">
          <button class={dbActiveTab === 'overview' ? 'active' : ''} on:click={() => dbActiveTab = 'overview'}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/></svg>
            Kesehatan Database ({databaseConfigs.length})
          </button>
          <button class={dbActiveTab === 'queries' ? 'active' : ''} on:click={() => openQueryExplorer()}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
            Slow Query & Active Process
          </button>
          <button class={dbActiveTab === 'schedules' ? 'active' : ''} on:click={() => dbActiveTab = 'schedules'}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            Jadwal Backup Otomatis ({backupSchedules.length})
          </button>
          <button class={dbActiveTab === 'logs' ? 'active' : ''} on:click={() => dbActiveTab = 'logs'}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
            Riwayat Log & Restore ({backupLogs.length})
          </button>
          {#if activeBackupCount > 0}
            <button class="btn" style="margin-left:auto; background:rgba(56, 189, 248, 0.15); border:1px solid #38bdf8; color:#38bdf8; font-size:12px; display:inline-flex; align-items:center; gap:6px; padding:4px 10px; border-radius:6px; font-weight:600;" on:click={() => showActiveBackupsModal = true}>
              <span style="display:inline-block; width:7px; height:7px; border-radius:50%; background:#38bdf8; box-shadow:0 0 6px #38bdf8;"></span>
              <span>{activeBackupCount} Backup Berjalan</span>
            </button>
          {/if}
        </div>

        {#if dbActiveTab === 'overview'}
          {#if databaseConfigs.length === 0}
            <div class="panel" style="padding:48px 20px; text-align:center; color:var(--dim)">
              <svg viewBox="0 0 24 24" width="42" height="42" fill="none" stroke="currentColor" stroke-width="1.5" style="margin-bottom:12px; opacity:0.6"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/></svg>
              <div style="font-size:15px; color:var(--text); font-weight:600">Belum ada database yang dipantau</div>
              <div style="font-size:12.5px; margin-top:6px; max-width:440px; margin-left:auto; margin-right:auto">
                Tambahkan instans MySQL/MariaDB, PostgreSQL, atau Redis di server ini untuk memantau koneksi, throughput, dan melakukan backup otomatis aman tanpa downtime.
              </div>
              <div style="display:flex; justify-content:center; gap:10px; margin-top:16px">
                <button
                  class="btn"
                  style={isDetectingDB ? 'border-color:var(--warn); color:var(--warn)' : ''}
                  on:click={isDetectingDB ? handleCancelAutoDetect : handleAutoDetect}
                >
                  <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
                  {isDetectingDB ? 'Memindai... (Klik untuk Batal)' : '🔍 Pindai & Deteksi Otomatis'}
                </button>
                <button class="btn primary" on:click={openAddDBModal}>+ Tambah Database Manual</button>
              </div>
            </div>
          {:else}
            <div class="db-grid">
              {#each databaseConfigs as db}
                {@const m = dbMetricsMap[db.id]}
                <div class="db-card">
                  <div class="db-card-header">
                    <div style="display:flex; align-items:center; gap:8px">
                      <span class="st {m ? (m.status === 'up' ? 'ok' : 'crit') : 'off'}"></span>
                      <div>
                        <div style="font-weight:600; font-size:14px; color:var(--text)">{db.name}</div>
                        <div style="font-size:11px; color:var(--dim); font-family:var(--mono)">{db.host}:{db.port} {db.databaseName ? '• db: ' + db.databaseName : ''}</div>
                      </div>
                    </div>
                    <span class="db-badge {db.type}">{db.type}</span>
                  </div>

                  <!-- Live Stats Row -->
                  <div class="db-stats-row">
                    <div class="db-stat-item">
                      <span class="lbl">Koneksi Aktif</span>
                      <span class="val">{m && m.status === 'up' ? `${m.activeConnections} / ${m.maxConnections || '-'}` : '-'}</span>
                    </div>
                    <div class="db-stat-item">
                      <span class="lbl">Throughput</span>
                      <span class="val">{m && m.status === 'up' ? `${m.queriesPerSec.toFixed(1)} qps` : '-'}</span>
                    </div>
                    <div class="db-stat-item">
                      <span class="lbl">Cache Hit</span>
                      <span class="val">{m && m.status === 'up' && m.cacheHitRate ? `${m.cacheHitRate.toFixed(1)}%` : '-'}</span>
                    </div>
                    <div class="db-stat-item">
                      <span class="lbl">Slow Queries</span>
                      <span class="val">{m && m.status === 'up' ? m.slowQueries : '-'}</span>
                    </div>
                    <div class="db-stat-item">
                      <span class="lbl">Ukuran / RAM</span>
                      <span class="val">
                        {#if m && m.status === 'up' && m.databaseSizeBytes}
                          {(m.databaseSizeBytes / (1024 * 1024)).toFixed(1)} MB
                        {:else if m && m.status === 'up' && m.memoryUsedBytes}
                          {(m.memoryUsedBytes / (1024 * 1024)).toFixed(1)} MB
                        {:else}
                          -
                        {/if}
                      </span>
                    </div>
                    <div class="db-stat-item">
                      <span class="lbl">Status</span>
                      <span class="val" style="color: {m ? (m.status === 'up' ? 'var(--ok)' : 'var(--crit)') : 'var(--warn)'}">
                        {m ? m.status.toUpperCase() : 'MENGECEK...'}
                      </span>
                    </div>
                  </div>

                  <!-- Connection Usage Bar -->
                  {#if m && m.status === 'up' && m.maxConnections > 0}
                    {@const connPct = Math.min(100, (m.activeConnections / m.maxConnections) * 100)}
                    <div style="display:flex; flex-direction:column; gap:4px">
                      <div style="display:flex; justify-content:space-between; font-size:11px; color:var(--dim)">
                        <span>Kapasitas Koneksi</span>
                        <span class="num">{connPct.toFixed(1)}%</span>
                      </div>
                      <div class="bar {connPct > 85 ? 'crit' : connPct > 70 ? 'warn' : ''}" style="width:100%; height:6px">
                        <i style="width:{connPct}%"></i>
                      </div>
                    </div>
                  {/if}

                  <!-- Down Warning Banner if Down -->
                  {#if m && m.status === 'down'}
                    <div style="font-size:11.5px; color:var(--warn); background:rgba(224,164,58,0.1); border:1px solid rgba(224,164,58,0.25); border-radius:4px; padding:6px 10px; margin-top:8px; display:flex; align-items:center; justify-content:space-between; gap:8px">
                      <span>⚠️ Koneksi ditolak / belum terhubung. Periksa kredensial.</span>
                      <button class="btn" style="padding:4px 10px; font-size:11.5px; height:28px; white-space:nowrap" on:click={() => openEditDBModal(db)}>Edit Kredensial</button>
                    </div>
                  {/if}

                  <div class="db-card-footer">
                    <div class="db-actions-primary">
                      <button class="btn primary" style="font-size:12px; padding:4px 10px" on:click={() => handleManualBackup(db.id)} disabled={isBackingUp[db.id]}>
                        {#if isBackingUp[db.id]}
                          <i>●</i> Sedang Dump...
                        {:else}
                          <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
                          Backup Sekarang
                        {/if}
                      </button>
                      <button class="btn" style="font-size:12px; padding:4px 8px" on:click={() => handleViewSchemas(db)} title="Lihat semua database, ukuran & struktur tabel di server">
                        <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>
                        Daftar DB
                      </button>
                      <button class="btn" style="font-size:12px; padding:4px 8px" on:click={() => openQueryExplorer(db.id)} title="Analisis Slow Query & Active Processlist">
                        <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
                        Slow Query
                      </button>
                    </div>

                    <div class="db-actions-secondary">
                      <button class="btn ghost" style="font-size:12px; padding:4px 8px; white-space:nowrap" on:click={() => openAddScheduleModal(db.id)} title="Tambah Jadwal Backup Otomatis">
                        <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                        Jadwal Backup
                      </button>
                      <div class="db-card-tools">
                        <button class="btn ghost icon-btn" on:click={() => handlePollDBMetrics(db.id)} title="Perbarui Metrik">
                          <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2"><path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"/></svg>
                        </button>
                        <button class="btn ghost icon-btn" on:click={() => openEditDBModal(db)} title="Edit Kredensial Database">
                          <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
                        </button>
                        <button class="btn ghost danger icon-btn" on:click={() => handleDeleteDatabase(db.id)} title="Hapus Database">✕</button>
                      </div>
                    </div>
                  </div>
                </div>
              {/each}
            </div>
          {/if}

        {:else if dbActiveTab === 'queries'}
          {#if databaseConfigs.length === 0}
            <div class="panel" style="padding:36px 20px; text-align:center; color:var(--dim)">
              Tambahkan atau deteksi otomatis database terlebih dahulu untuk menganalisis Slow Query & Active Processlist.
            </div>
          {:else}
            <div style="display:flex; align-items:center; justify-content:space-between; gap:12px; margin-bottom:14px; flex-wrap:wrap">
              <div style="display:flex; align-items:center; gap:8px">
                <span style="font-size:12px; color:var(--muted)">Pilih Instans Database:</span>
                <select
                  style="height:32px; background:var(--panel); border:1px solid var(--line); color:var(--text); border-radius:5px; padding:0 10px; font-weight:500"
                  bind:value={selectedQueryDbId}
                  on:change={loadQueryAnalysis}
                >
                  {#each databaseConfigs as db}
                    <option value={db.id}>{db.name} ({db.type.toUpperCase()} - {db.host}:{db.port})</option>
                  {/each}
                </select>
                {#if queryAnalysis?.sourceNote}
                  <span class="tag" style="color:var(--dim)">{queryAnalysis.sourceNote}</span>
                {/if}
              </div>

              <button class="btn primary" on:click={loadQueryAnalysis} disabled={isLoadingQueries}>
                <svg viewBox="0 0 24 24"><path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"/></svg>
                {isLoadingQueries ? 'Menganalisis...' : 'Refresh Analisis Query'}
              </button>
            </div>

            {#if queryError}
              <div class="panel" style="padding:14px 16px; margin-bottom:14px; border-color:var(--crit); color:var(--crit)">
                {queryError}
              </div>
            {/if}

            <!-- Active Queries / Processlist -->
            <div class="panel" style="margin-bottom:16px">
              <div class="panel-h">
                <h2>Query & Sesi Aktif Saat Ini ({queryAnalysis ? queryAnalysis.activeQueries.length : 0})</h2>
                <span class="sub">Proses SQL yang sedang dieksekusi (non-sleep)</span>
              </div>
              {#if !queryAnalysis || queryAnalysis.activeQueries.length === 0}
                <div style="padding:26px 20px; text-align:center; color:var(--dim)">
                  Tidak ada query yang sedang menggantung atau berjalan lama saat ini.
                </div>
              {:else}
                <table>
                  <thead>
                    <tr>
                      <th>PID / ID</th>
                      <th>User & Host</th>
                      <th>Database</th>
                      <th>State / Command</th>
                      <th class="r">Durasi</th>
                      <th>SQL Query / Command</th>
                      <th class="r">Aksi</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each queryAnalysis.activeQueries as aq}
                      <tr>
                        <td class="num" style="font-weight:600">{aq.pid}</td>
                        <td class="num" style="font-size:12px">{aq.user}@{aq.host}</td>
                        <td><span class="tag">{aq.database || '-'}</span></td>
                        <td style="font-size:12px; color:var(--muted)">{aq.state || aq.command}</td>
                        <td class="r num" style="font-weight:600; color:{aq.timeSec > 5 ? 'var(--crit)' : aq.timeSec > 1 ? 'var(--warn)' : 'var(--ok)'}">
                          {aq.timeSec.toFixed(1)} dtk
                        </td>
                        <td class="num" style="font-size:11.5px; max-width:380px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap" title={aq.query}>
                          {aq.query}
                        </td>
                        <td class="r">
                          <button
                            class="btn ghost danger"
                            style="height:26px; padding:0 8px; font-size:11.5px"
                            on:click={() => handleKillQuery(aq.pid)}
                            disabled={killingQueryPid[aq.pid]}
                            title="Hentikan paksa query ini (KILL QUERY / pg_cancel_backend)"
                          >
                            {killingQueryPid[aq.pid] ? 'Menghentikan...' : 'Hentikan'}
                          </button>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              {/if}
            </div>

            <!-- Slow Query Digest Explorer -->
            <div class="panel">
              <div class="panel-h">
                <h2>Top Slow Queries & Digest ({queryAnalysis ? queryAnalysis.slowQueries.length : 0})</h2>
                <span class="sub">Query terberat berdasarkan rata-rata waktu eksekusi untuk optimasi indexing</span>
              </div>
              {#if !queryAnalysis || queryAnalysis.slowQueries.length === 0}
                <div style="padding:28px 20px; text-align:center; color:var(--dim)">
                  Belum ada catatan slow query digest yang terdeteksi pada instans ini.
                </div>
              {:else}
                <table>
                  <thead>
                    <tr>
                      <th>Schema</th>
                      <th class="r">Eksekusi</th>
                      <th class="r">Rata-rata (Avg)</th>
                      <th class="r">Maksimum (Max)</th>
                      <th class="r">Baris Diperiksa</th>
                      <th>Pola SQL Query (Digest)</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each queryAnalysis.slowQueries as sq}
                      <tr>
                        <td><span class="tag">{sq.schemaName || 'all'}</span></td>
                        <td class="r num">{sq.execCount}x</td>
                        <td class="r num" style="font-weight:600; color:{sq.avgTimeMs > 500 ? 'var(--crit)' : sq.avgTimeMs > 100 ? 'var(--warn)' : 'var(--text)'}">
                          {sq.avgTimeMs.toFixed(2)} ms
                        </td>
                        <td class="r num" style="color:var(--muted)">{sq.maxTimeMs.toFixed(2)} ms</td>
                        <td class="r num">{sq.rowsExamined}</td>
                        <td class="num" style="font-size:11.5px; color:var(--text); max-width:440px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap" title={sq.queryText}>
                          {sq.queryText}
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              {/if}
            </div>
          {/if}

        {:else if dbActiveTab === 'schedules'}
          <div class="panel">
            <div class="panel-h">
              <h2>Aturan Otomasi Backup Terjadwal</h2>
              <span class="spacer"></span>
              <button class="btn primary" on:click={() => openAddScheduleModal()}>+ Buat Jadwal Baru</button>
            </div>
            {#if backupSchedules.length === 0}
              <div style="padding:30px; text-align:center; color:var(--dim)">Belum ada jadwal backup otomatis untuk server ini.</div>
            {:else}
              <table>
                <thead>
                  <tr>
                    <th>Target DB</th>
                    <th>Frekuensi</th>
                    <th>Jam Eksekusi</th>
                    <th>Retensi</th>
                    <th>Tujuan Simpan</th>
                    <th>Terakhir Berjalan</th>
                    <th class="r">Aksi</th>
                  </tr>
                </thead>
                <tbody>
                  {#each backupSchedules as sc}
                    <tr>
                      <td style="font-weight:600; color:var(--text)">{sc.databaseName || 'all_databases'}</td>
                      <td><span class="tag">{sc.frequency.toUpperCase()}</span></td>
                      <td class="num">{sc.timeOfDay} WIB</td>
                      <td class="num">{sc.retentionDays} hari</td>
                      <td>
                        <span class="tag" style="color:var(--accent)">
                          {sc.storageTarget === 's3' ? '☁️ Cloud S3 / R2' : sc.storageTarget === 'local' ? 'Laptop Lokal' : 'Server Remote'}
                        </span>
                      </td>
                      <td class="num" style="color:var(--dim); font-size:12px">
                        {sc.lastRun ? new Date(sc.lastRun).toLocaleString() : 'Belum pernah'}
                      </td>
                      <td class="r">
                        <button class="btn ghost danger" on:click={() => handleDeleteSchedule(sc.id)}>Hapus</button>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {/if}
          </div>

        {:else if dbActiveTab === 'logs'}
          <div class="panel">
            <div class="panel-h">
              <h2>Riwayat Eksekusi Backup & Restore Database</h2>
              <span class="spacer"></span>
              <button class="btn" on:click={() => navigateTo('settings')}>Konfigurasi Cloud S3 / R2</button>
              <button class="btn" on:click={loadDatabaseData}>Refresh Log</button>
            </div>
            {#if backupLogs.length === 0}
              <div style="padding:30px; text-align:center; color:var(--dim)">Belum ada riwayat backup yang tercatat untuk server ini.</div>
            {:else}
              <table>
                <thead>
                  <tr>
                    <th>Status</th>
                    <th>Waktu Eksekusi</th>
                    <th>Database</th>
                    <th>Ukuran File</th>
                    <th>Durasi</th>
                    <th>Lokasi Arsip & Integritas</th>
                    <th class="r">Aksi Pemulihan & Offsite</th>
                  </tr>
                </thead>
                <tbody>
                  {#each backupLogs as log}
                    {@const vRes = verifyResultMap[log.id]}
                    <tr>
                      <td>
                        {#if log.status === 'success'}
                          <span class="tag" style="background:rgba(63,182,122,0.15); color:var(--ok); border:1px solid var(--ok)">SUKSES</span>
                        {:else if log.status === 'running'}
                          <span class="tag" style="background:rgba(224,164,58,0.15); color:var(--warn); border:1px solid var(--warn)">BERJALAN</span>
                        {:else}
                          <span class="tag" style="background:rgba(229,83,75,0.15); color:var(--crit); border:1px solid var(--crit)">GAGAL</span>
                        {/if}
                      </td>
                      <td class="num" style="font-size:12px">{new Date(log.startedAt).toLocaleString()}</td>
                      <td style="font-weight:600">{log.databaseName}</td>
                      <td class="num">{log.fileSizeBytes ? (log.fileSizeBytes / (1024 * 1024)).toFixed(2) + ' MB' : '-'}</td>
                      <td class="num">{log.durationSec.toFixed(1)} dtk</td>
                      <td style="max-width:250px">
                        <div class="num" style="font-size:11.5px; color:var(--dim); overflow:hidden; text-overflow:ellipsis; white-space:nowrap" title={log.filePath || log.errorMessage}>
                          {log.filePath || log.errorMessage || '-'}
                        </div>
                        {#if vRes}
                          <div style="font-size:11px; margin-top:3px; color:{vRes.valid ? 'var(--ok)' : 'var(--crit)'}" title={vRes.sha256 ? 'SHA-256: ' + vRes.sha256 : ''}>
                            {vRes.valid ? '✓ Arsip Utuh (gzip OK)' : '✕ ' + vRes.message}
                            {#if vRes.sha256}
                              <span class="num" style="color:var(--dim)"> • sha256:{vRes.sha256.slice(0, 10)}…</span>
                            {/if}
                          </div>
                        {/if}
                      </td>
                      <td class="r">
                        {#if log.status === 'success'}
                          <div style="display:flex; justify-content:flex-end; gap:5px; flex-wrap:wrap">
                            <button
                              class="btn ghost"
                              style="height:26px; padding:0 8px; font-size:11px"
                              on:click={() => handleVerifyBackup(log)}
                              disabled={isVerifyingBackup[log.id]}
                              title="Cek keutuhan gzip & checksum SHA-256 di server"
                            >
                              {isVerifyingBackup[log.id] ? 'Mengecek...' : '✓ Cek'}
                            </button>
                            <button
                              class="btn ghost"
                              style="height:26px; padding:0 8px; font-size:11px"
                              on:click={() => handleDownloadBackup(log)}
                              disabled={isDownloadingBackup[log.id]}
                              title="Unduh salinan arsip backup ke laptop Windows ini"
                            >
                              {isDownloadingBackup[log.id] ? 'Mengunduh...' : '⬇ Unduh'}
                            </button>
                            <button
                              class="btn ghost"
                              style="height:26px; padding:0 8px; font-size:11px; color:var(--accent)"
                              on:click={() => handleUploadBackupToS3(log)}
                              disabled={isUploadingS3[log.id]}
                              title="Unggah arsip backup ini ke bucket S3 / Cloudflare R2 / MinIO"
                            >
                              {isUploadingS3[log.id] ? 'Upload...' : '☁️ S3'}
                            </button>
                            <button
                              class="btn primary"
                              style="height:26px; padding:0 9px; font-size:11px"
                              on:click={() => openRestoreWizard(log)}
                              title="Buka Wizard Restore / Rollback Database dari arsip ini"
                            >
                              ⏪ Restore
                            </button>
                          </div>
                        {:else}
                          <span style="color:var(--dim); font-size:11.5px">-</span>
                        {/if}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {/if}
          </div>
        {/if}
      </div>

    {:else if currentNav === 'alert'}
      <div class="topbar">
        <h1>Alert & Notifikasi</h1>
        <span class="sub">{activeAlerts.length} insiden aktif</span>
        <span class="spacer"></span>
        <button class="btn" on:click={() => navigateTo('settings')}>
          Konfigurasi Telegram & Webhook
        </button>
        <button class="btn" on:click={() => ExportReport('csv').catch(() => {})}>
          Export CSV
        </button>
        <button class="btn" on:click={loadAlerts} disabled={isLoadingAlerts}>
          <svg viewBox="0 0 24 24"><path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"/></svg>
          {isLoadingAlerts ? 'Memuat...' : 'Refresh'}
        </button>
      </div>

      <div class="content">
        <!-- Subnav Tabs -->
        <div class="subnav-tabs">
          <button class={alertTab === 'events' ? 'active' : ''} on:click={() => alertTab = 'events'}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 4 2.5 20h19L12 4z"/><path d="M12 10v4M12 17h.01"/></svg>
            Riwayat Insiden Alert
            {#if activeAlerts.length > 0}
              <span class="tag" style="background:var(--crit); color:#fff; font-weight:600; padding:1px 6px; font-size:11px">{activeAlerts.length}</span>
            {/if}
          </button>
          <button class={alertTab === 'rules' ? 'active' : ''} on:click={() => alertTab = 'rules'}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
            Aturan Ambang Batas (Rules) ({alertRules.length})
          </button>
        </div>

        {#if alertTab === 'events'}
          <!-- Filter Buttons -->
          <div style="display:flex; gap:8px; margin-bottom:12px; align-items:center">
            <span style="font-size:12px; color:var(--dim)">Filter Status:</span>
            <div class="seg">
              <button class={alertFilter === 'all' ? 'on' : ''} on:click={() => alertFilter = 'all'}>Semua ({alertEvents.length})</button>
              <button class={alertFilter === 'active' ? 'on' : ''} on:click={() => alertFilter = 'active'}>Aktif ({activeAlerts.length})</button>
              <button class={alertFilter === 'resolved' ? 'on' : ''} on:click={() => alertFilter = 'resolved'}>Terselesaikan ({alertEvents.filter(e => e.status === 'resolved').length})</button>
            </div>
          </div>

          <div class="panel">
            {#if alertEvents.filter(ev => alertFilter === 'all' || ev.status === alertFilter).length === 0}
              <div style="padding:40px 20px; text-align:center; color:var(--dim)">
                <svg viewBox="0 0 24 24" width="36" height="36" fill="none" stroke="currentColor" stroke-width="1.5" style="margin-bottom:8px; opacity:0.6"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                <div style="font-size:14px; color:var(--text); font-weight:500">Tidak ada insiden alert</div>
                <div style="font-size:12px; margin-top:4px">Semua server yang dipantau berada dalam kondisi normal dan sehat.</div>
              </div>
            {:else}
              <table>
                <thead>
                  <tr>
                    <th>Status</th>
                    <th>Tingkat</th>
                    <th>Server</th>
                    <th>Deskripsi Insiden</th>
                    <th>Waktu Mulai</th>
                    <th>Waktu Pulih</th>
                  </tr>
                </thead>
                <tbody>
                  {#each alertEvents.filter(ev => alertFilter === 'all' || ev.status === alertFilter) as ev}
                    <tr>
                      <td>
                        {#if ev.status === 'active'}
                          <span class="tag" style="background:rgba(229,83,75,0.15); color:var(--crit); border:1px solid var(--crit); display:inline-flex; align-items:center; gap:5px">
                            <i style="width:6px; height:6px; border-radius:50%; background:var(--crit)"></i>
                            Aktif
                          </span>
                        {:else}
                          <span class="tag" style="background:rgba(63,182,122,0.15); color:var(--ok); border:1px solid var(--ok); display:inline-flex; align-items:center; gap:5px">
                            <i style="width:6px; height:6px; border-radius:50%; background:var(--ok)"></i>
                            Pulih
                          </span>
                        {/if}
                      </td>
                      <td>
                        {#if ev.level === 'critical'}
                          <span class="tag" style="color:var(--crit); font-weight:600">Kritis</span>
                        {:else if ev.level === 'warning'}
                          <span class="tag" style="color:var(--warn); font-weight:600">Peringatan</span>
                        {:else}
                          <span class="tag">{ev.level}</span>
                        {/if}
                      </td>
                      <td style="font-weight:600">{ev.serverName}</td>
                      <td>{ev.message}</td>
                      <td class="num" style="color:var(--dim); font-size:12px">{new Date(ev.startedAt).toLocaleString()}</td>
                      <td class="num" style="color:var(--dim); font-size:12px">
                        {ev.resolvedAt ? new Date(ev.resolvedAt).toLocaleString() : '-'}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {/if}
          </div>

        {:else if alertTab === 'rules'}
          <div class="panel">
            <table>
              <thead>
                <tr>
                  <th>Status</th>
                  <th>ID Rule</th>
                  <th>Metrik Dipantau</th>
                  <th>Kondisi Ambang Batas</th>
                  <th>Durasi Evaluasi</th>
                  <th>Tingkat</th>
                  <th class="r">Aksi</th>
                </tr>
              </thead>
              <tbody>
                {#each alertRules as rule}
                  <tr>
                    <td>
                      <label class="switch">
                        <input type="checkbox" checked={rule.enabled} on:change={() => handleToggleRule(rule)} />
                        <span class="slider"></span>
                      </label>
                    </td>
                    <td class="num" style="color:var(--dim)">{rule.id}</td>
                    <td style="font-weight:600; color:var(--text)">
                      {#if rule.metric === 'disk_used_pct'}
                        Partisi Disk (Semua mount)
                      {:else if rule.metric === 'cpu_total_pct'}
                        Beban Total CPU
                      {:else if rule.metric === 'mem_used_pct'}
                        Penggunaan Memori RAM
                      {:else if rule.metric === 'ssh_unreachable'}
                        Koneksi SSH Terputus
                      {:else}
                        {rule.metric}
                      {/if}
                    </td>
                    <td class="num" style="color:var(--accent)">
                      {rule.operator} {rule.threshold}{rule.metric.includes('pct') ? '%' : ''}
                    </td>
                    <td class="num">{rule.durationSec} detik</td>
                    <td>
                      {#if rule.level === 'critical'}
                        <span class="tag" style="color:var(--crit)">Kritis</span>
                      {:else}
                        <span class="tag" style="color:var(--warn)">Peringatan</span>
                      {/if}
                    </td>
                    <td class="r">
                      <button class="btn ghost" on:click={() => handleToggleRule(rule)}>
                        {rule.enabled ? 'Nonaktifkan' : 'Aktifkan'}
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>

    {:else if currentNav === 'term'}
      <div class="topbar">
        <h1>Terminal & Log Server</h1>
        <span class="sub">Akses baris perintah SSH dan pembaca log jarak jauh</span>
        <span class="spacer"></span>
        <div style="display:flex; align-items:center; gap:8px">
          <span style="font-size:12px; color:var(--muted)">Target Server:</span>
          <select style="height:32px; background:var(--bg); border:1px solid var(--line); color:var(--text); border-radius:5px; padding:0 10px; font-weight:500" bind:value={termServerId}>
            {#if servers.length === 0}
              <option value="">Belum ada server terdaftar</option>
            {/if}
            {#each servers as s}
              <option value={s.id}>{s.name} ({s.user}@{s.host}:{s.port})</option>
            {/each}
          </select>
        </div>
      </div>

      <div class="content">
        <!-- Subnav Tabs -->
        <div class="subnav-tabs">
          <button class={termTab === 'pty' ? 'active' : ''} on:click={() => termTab = 'pty'}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            Terminal Interaktif Penuh (PTY SSH)
          </button>
          <button class={termTab === 'broadcast' ? 'active' : ''} on:click={() => termTab = 'broadcast'}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>
            Multi-Server Broadcast SSH
          </button>
          <button class={termTab === 'terminal' ? 'active' : ''} on:click={() => termTab = 'terminal'}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 7 4 4 20 4 20 7"/><line x1="9" y1="20" x2="15" y2="20"/><line x1="12" y1="4" x2="12" y2="20"/></svg>
            Quick Command Console
          </button>
          <button class={termTab === 'log' ? 'active' : ''} on:click={() => { termTab = 'log'; if (!logOutput) fetchLog(); }}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
            Quick Log Viewer
          </button>
        </div>

        {#if termTab === 'pty'}
          <!-- Interactive PTY Terminal (xterm.js + Go SSH PTY) -->
          <div class="panel" style="padding:0; overflow:hidden; display:flex; flex-direction:column; height:calc(100vh - 200px); min-height:480px; border:1px solid var(--line); border-radius:6px">
            <div class="terminal-header" style="background:#14171b; padding:8px 14px; border-bottom:1px solid var(--line); display:flex; align-items:center; gap:10px">
              <span style="font-size:12px; font-weight:600; color:var(--text); display:flex; align-items:center; gap:6px">
                <span class="st {ptyStatus === 'connected' ? 'ok' : ptyStatus === 'connecting' ? 'warn' : 'off'}"></span>
                PTY SSH Session {ptyStatus === 'connected' ? '(Aktif)' : ptyStatus === 'connecting' ? '(Menghubungkan...)' : '(Terputus)'}
              </span>

              {#if ptyStatus === 'connected'}
                <span style="font-size:11px; color:var(--dim)">| Pintasan / Perintah:</span>
                <div class="quick-commands">
                  <button class="quick-btn" on:click={() => sendPtyInput('htop\r')}>htop</button>
                  <button class="quick-btn" on:click={() => sendPtyInput('top\r')}>top</button>
                  <button class="quick-btn" on:click={() => sendPtyInput('df -h\r')}>df -h</button>
                  <button class="quick-btn" on:click={() => sendPtyInput('free -h\r')}>free -h</button>
                  <button class="quick-btn" on:click={() => sendPtyInput('docker ps\r')}>docker ps</button>
                  <button class="quick-btn" on:click={() => sendPtyInput('ls -la\r')}>ls -la</button>
                  <button class="quick-btn" on:click={() => sendPtyInput('\x03')}>Ctrl+C</button>
                </div>
              {/if}

              <span class="spacer"></span>

              {#if ptyStatus === 'connected'}
                <button class="btn ghost" style="padding:4px 8px; font-size:11px" on:click={clearPtyTerminal}>Bersihkan Layar</button>
                <button class="btn danger" style="padding:4px 10px; font-size:11.5px" on:click={() => disconnectPtyTerminal()}>
                  Putuskan Sesi
                </button>
              {:else if ptyStatus === 'connecting'}
                <button class="btn" style="padding:4px 10px; font-size:11.5px" disabled>
                  Menghubungkan...
                </button>
              {:else}
                <button class="btn primary" style="padding:4px 12px; font-size:11.5px" on:click={() => connectPtyTerminal()} disabled={!termServerId}>
                  Mulai Terminal SSH
                </button>
              {/if}
            </div>

            <!-- Terminal Viewport -->
            <div style="flex:1; position:relative; background:#0d1117; overflow:hidden">
              {#if ptyStatus === 'disconnected'}
                <div style="height:100%; display:flex; flex-direction:column; align-items:center; justify-content:center; color:var(--dim); text-align:center; padding:24px">
                  <svg viewBox="0 0 24 24" width="44" height="44" fill="none" stroke="currentColor" stroke-width="1.5" style="opacity:0.5; margin-bottom:12px"><rect x="3" y="4" width="18" height="16" rx="2"/><polyline points="7 10 10 13 7 16"/><line x1="12" y1="16" x2="17" y2="16"/></svg>
                  <div style="font-weight:600; color:var(--text); font-size:14px; margin-bottom:4px">Terminal Interaktif PTY Belum Terhubung</div>
                  <div style="font-size:12px; max-width:460px; margin-bottom:16px; line-height:1.5">
                    Mendukung navigasi penuh bash/zsh, interactive CLI seperti htop, nano, vim, tmux, dan streaming dua arah real-time via RFC 4254 PTY channel.
                  </div>
                  {#if ptyError}
                    <div style="color:var(--crit); font-size:12px; background:rgba(239,68,68,0.1); border:1px solid rgba(239,68,68,0.25); padding:8px 14px; border-radius:5px; margin-bottom:14px">
                      Kesalahan: {ptyError}
                    </div>
                  {/if}
                  <button class="btn primary" on:click={() => connectPtyTerminal()} disabled={!termServerId}>
                    Buka Sesi Terminal Sekarang
                  </button>
                </div>
              {/if}

              <div
                bind:this={ptyContainer}
                class="pty-container-box"
                style="{ptyStatus === 'disconnected' ? 'display:none;' : ''}"
              ></div>
            </div>
          </div>

        {:else if termTab === 'broadcast'}
          <!-- Multi-Server Broadcast SSH -->
          <div style="display:flex; flex-direction:column; gap:16px">
            <!-- Filter & Server Selection Card -->
            <div class="panel" style="padding:16px">
              <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:12px; flex-wrap:wrap; gap:10px">
                <div>
                  <h2 style="font-size:14px; font-weight:600; margin-bottom:2px">Pilih Server Target Broadcast</h2>
                  <span style="font-size:11.5px; color:var(--dim)">Jalankan perintah serentak (concurrent) pada sekelompok server berdasarkan label/tag</span>
                </div>
                <div style="display:flex; align-items:center; gap:8px">
                  <button class="btn ghost" style="font-size:11.5px; padding:3px 8px" on:click={toggleAllBroadcastServers}>
                    {broadcastSelectedServerIds.length === servers.length ? 'Batalkan Semua' : 'Pilih Semua'}
                  </button>
                  <span class="num" style="font-size:12px; color:var(--text); background:var(--bg); padding:3px 8px; border-radius:4px; border:1px solid var(--line)">
                    {broadcastSelectedServerIds.length} / {servers.length} server dipilih
                  </span>
                </div>
              </div>

              <!-- Tag Quick Select Chips -->
              {#if availableBroadcastTags.length > 0}
                <div style="display:flex; align-items:center; gap:6px; margin-bottom:12px; flex-wrap:wrap">
                  <span style="font-size:11.5px; color:var(--dim)">Filter Tag:</span>
                  <button
                    class="tag {broadcastFilterTag === 'all' ? 'active-tag' : ''}"
                    style="cursor:pointer; font-size:11px; padding:2px 8px"
                    on:click={() => selectBroadcastTag('all')}
                  >
                    Semua ({servers.length})
                  </button>
                  {#each availableBroadcastTags as tag}
                    {@const count = servers.filter(s => (s.tags || []).includes(tag)).length}
                    <button
                      class="tag {broadcastFilterTag === tag ? 'active-tag' : ''}"
                      style="cursor:pointer; font-size:11px; padding:2px 8px"
                      on:click={() => selectBroadcastTag(tag)}
                    >
                      {tag} ({count})
                    </button>
                  {/each}
                </div>
              {/if}

              <!-- Server Checkbox Grid -->
              <div style="display:grid; grid-template-columns:repeat(auto-fill, minmax(260px, 1fr)); gap:8px; max-height:180px; overflow-y:auto; padding:4px 2px">
                {#each servers as s}
                  {@const isSelected = broadcastSelectedServerIds.includes(s.id)}
                  <label
                    style="display:flex; align-items:center; gap:8px; padding:8px 10px; background:var(--bg); border:1px solid {isSelected ? 'var(--accent)' : 'var(--line)'}; border-radius:6px; cursor:pointer; user-select:none; font-size:12px; transition:border-color 0.15s"
                  >
                    <input
                      type="checkbox"
                      checked={isSelected}
                      on:change={() => toggleBroadcastServer(s.id)}
                      style="cursor:pointer"
                    />
                    <span class="st {s.status === 'critical' ? 'crit' : s.status === 'warning' ? 'warn' : s.status === 'unreachable' ? 'off' : 'ok'}"></span>
                    <div style="flex:1; overflow:hidden">
                      <div style="font-weight:600; white-space:nowrap; overflow:hidden; text-overflow:ellipsis">{s.name}</div>
                      <div class="num" style="font-size:11px; color:var(--dim)">{s.user}@{s.host}:{s.port}</div>
                    </div>
                    {#if s.tags && s.tags.length > 0}
                      <span class="tag" style="font-size:10px; padding:1px 5px">{s.tags[0]}</span>
                    {/if}
                  </label>
                {/each}
              </div>
            </div>

            <!-- Command Input & Presets Card -->
            <div class="panel" style="padding:16px">
              <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:8px">
                <span style="font-size:12px; font-weight:600; color:var(--text)">Perintah Ringan Broadcast:</span>
                <div class="quick-commands">
                  <button class="quick-btn" on:click={() => { broadcastCmd = 'uptime'; executeBroadcast('uptime'); }}>uptime</button>
                  <button class="quick-btn" on:click={() => { broadcastCmd = 'df -h /'; executeBroadcast('df -h /'); }}>df -h /</button>
                  <button class="quick-btn" on:click={() => { broadcastCmd = 'free -h'; executeBroadcast('free -h'); }}>free -h</button>
                  <button class="quick-btn" on:click={() => { broadcastCmd = 'uname -r'; executeBroadcast('uname -r'); }}>kernel</button>
                  <button class="quick-btn" on:click={() => { broadcastCmd = 'docker ps --format "table {{.Names}}\t{{.Status}}"'; executeBroadcast('docker ps --format "table {{.Names}}\t{{.Status}}"'); }}>docker ps</button>
                  <button class="quick-btn" on:click={() => { broadcastCmd = 'systemctl is-active docker nginx ssh'; executeBroadcast('systemctl is-active docker nginx ssh'); }}>services</button>
                </div>
              </div>

              <div style="display:flex; gap:10px">
                <div class="terminal-input-wrap" style="flex:1; background:var(--bg); border:1px solid var(--line); border-radius:5px; padding:0 12px; display:flex; align-items:center; gap:8px">
                  <span style="color:var(--accent); font-family:var(--mono); font-weight:700">$</span>
                  <input
                    type="text"
                    style="flex:1; height:34px; background:transparent; border:none; color:var(--text); font-family:var(--mono); font-size:12.5px; outline:none"
                    placeholder="Ketik perintah shell broadcast (misal: uptime atau cat /etc/os-release)..."
                    bind:value={broadcastCmd}
                    on:keydown={(e) => e.key === 'Enter' && executeBroadcast()}
                    disabled={isBroadcasting}
                  />
                </div>
                <button
                  class="btn primary"
                  style="padding:0 16px; font-weight:600; font-size:12px"
                  on:click={() => executeBroadcast()}
                  disabled={isBroadcasting || broadcastSelectedServerIds.length === 0 || !broadcastCmd.trim()}
                >
                  {isBroadcasting ? 'Mengeksekusi...' : `Kirim ke ${broadcastSelectedServerIds.length} Server`}
                </button>
              </div>

              {#if broadcastError}
                <div style="color:var(--crit); font-size:12px; margin-top:8px">
                  Error: {broadcastError}
                </div>
              {/if}
            </div>

            <!-- Broadcast Results Display -->
            {#if isBroadcasting}
              <div class="panel" style="padding:24px; text-align:center; color:var(--accent)">
                <i>●</i> Menjalankan perintah secara paralel pada {broadcastSelectedServerIds.length} server remote...
              </div>
            {:else if broadcastResults.length > 0}
              {@const successCount = broadcastResults.filter(r => !r.error).length}
              {@const failCount = broadcastResults.filter(r => !!r.error).length}
              <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:-4px">
                <div style="font-size:12.5px; font-weight:600; color:var(--text)">
                  Hasil Eksekusi: <span style="color:var(--ok)">{successCount} Berhasil</span>, <span style="color:{failCount > 0 ? 'var(--crit)' : 'var(--dim)'}">{failCount} Gagal</span>
                </div>
                <span class="num" style="font-size:11.5px; color:var(--dim)">Perintah: <code>{broadcastCmd}</code></span>
              </div>

              <div style="display:grid; grid-template-columns:1fr; gap:12px">
                {#each broadcastResults as res}
                  <div class="panel" style="padding:12px 14px; background:#161b22; border:1px solid {res.error ? 'rgba(239,68,68,0.35)' : 'var(--line)'}">
                    <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:8px">
                      <div style="display:flex; align-items:center; gap:8px">
                        <span class="st {res.error ? 'crit' : 'ok'}"></span>
                        <span style="font-weight:600; font-size:13px; color:var(--text)">{res.serverName}</span>
                        <span class="num" style="font-size:11px; color:var(--dim)">({res.host})</span>
                        {#each (res.tags || []) as t}
                          <span class="tag" style="font-size:10px; padding:1px 5px">{t}</span>
                        {/each}
                      </div>
                      <div style="display:flex; align-items:center; gap:10px">
                        <span class="num" style="font-size:11px; color:var(--dim)">{res.durationMs} ms</span>
                        <button
                          class="btn ghost"
                          style="font-size:11px; padding:2px 6px"
                          on:click={() => navigator.clipboard.writeText(res.output || res.error || '')}
                        >
                          Salin
                        </button>
                      </div>
                    </div>

                    {#if res.error}
                      <div style="color:var(--crit); font-size:11.5px; margin-bottom:4px; font-family:var(--mono)">
                        Error: {res.error}
                      </div>
                    {/if}

                    <pre style="margin:0; background:#0d1117; color:#e6edf3; padding:10px 12px; border-radius:5px; font-family:var(--mono); font-size:12px; max-height:220px; overflow-y:auto; white-space:pre-wrap; border:1px solid rgba(255,255,255,0.06)">{res.output || '(Tidak ada output yang dihasilkan)'}</pre>
                  </div>
                {/each}
              </div>
            {/if}
          </div>

        {:else if termTab === 'terminal'}
          <!-- Terminal Area -->
          <div class="terminal-container">
            <div class="terminal-header">
              <span style="font-size:12px; font-weight:600; color:var(--text); display:flex; align-items:center; gap:6px">
                <span class="st ok"></span>
                Console Cepat Tunggal
              </span>
              <span style="font-size:11px; color:var(--dim)">| Perintah Cepat:</span>
              <div class="quick-commands">
                <button class="quick-btn" on:click={() => executeSSHCommand('uptime')}>uptime</button>
                <button class="quick-btn" on:click={() => executeSSHCommand('free -m')}>free -m</button>
                <button class="quick-btn" on:click={() => executeSSHCommand('df -h')}>df -h</button>
                <button class="quick-btn" on:click={() => executeSSHCommand('ps aux --sort=-%cpu | head -10')}>top cpu</button>
                <button class="quick-btn" on:click={() => executeSSHCommand('docker ps')}>docker ps</button>
                <button class="quick-btn" on:click={() => executeSSHCommand('ss -tuln')}>listening ports</button>
                <button class="quick-btn" on:click={() => executeSSHCommand('uname -a')}>uname -a</button>
              </div>
              <span class="spacer"></span>
              <button class="btn ghost" style="padding:4px 8px; font-size:11px" on:click={clearTerminal}>Bersihkan Layar</button>
            </div>

            <div class="terminal-body" bind:this={termContainer}>
              <div style="color:var(--dim); margin-bottom:12px; font-size:11.5px">
                # ServerWatch Agentless Remote Execution Console<br/>
                # Kredensial tersimpan aman di Windows Credential Manager. Buffer dibatasi 1MB.<br/>
                # Ketik perintah di bawah dan tekan Enter atau klik Kirim.
              </div>

              {#each termHistory as item}
                <div class="terminal-prompt-entry">
                  <div class="terminal-prompt-line">
                    <span>$ {item.cmd}</span>
                    <span class="terminal-prompt-time num">{item.time}</span>
                  </div>
                  <pre class="terminal-output {item.error ? 'err' : ''}">{item.output || '(Tidak ada output)'}</pre>
                </div>
              {/each}

              {#if isRunningCmd}
                <div style="color:var(--accent); display:flex; align-items:center; gap:8px">
                  <i>●</i> Menjalankan perintah di server remote...
                </div>
              {/if}
            </div>

            <div class="terminal-footer">
              <div class="terminal-input-wrap">
                <span>$</span>
                <input
                  type="text"
                  placeholder="Ketik perintah (contoh: systemctl status nginx)..."
                  bind:value={termCommand}
                  on:keydown={(e) => e.key === 'Enter' && executeSSHCommand()}
                  disabled={isRunningCmd}
                />
              </div>
              <button class="btn primary" on:click={() => executeSSHCommand()} disabled={isRunningCmd || !termCommand.trim()}>
                {isRunningCmd ? 'Menjalankan...' : 'Kirim'}
              </button>
            </div>
          </div>

        {:else if termTab === 'log'}
          <!-- Log Viewer Area -->
          <div class="log-viewer-container">
            <div class="terminal-header">
              <span style="font-size:12px; font-weight:600; color:var(--text)">Target Log:</span>
              <input
                type="text"
                style="height:28px; background:var(--bg); border:1px solid var(--line); color:var(--text); font-size:12px; border-radius:4px; padding:0 8px; width:130px; font-family:var(--mono)"
                bind:value={logTarget}
                placeholder="Target service / log"
              />
              <div class="quick-commands">
                <button class="quick-btn" on:click={() => fetchLog('syslog')}>syslog</button>
                <button class="quick-btn" on:click={() => fetchLog('ssh')}>ssh</button>
                <button class="quick-btn" on:click={() => fetchLog('docker')}>docker</button>
                <button class="quick-btn" on:click={() => fetchLog('nginx')}>nginx</button>
                <button class="quick-btn" on:click={() => fetchLog('dmesg')}>dmesg</button>
              </div>
              <button class="btn" style="height:28px; padding:0 10px; font-size:11.5px" on:click={() => fetchLog()} disabled={isLoadingLog}>
                {isLoadingLog ? 'Menarik...' : 'Tarik Log'}
              </button>
              <span class="spacer"></span>
              <div class="search" style="height:28px">
                <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="6.5"/><path d="m20 20-4.2-4.2"/></svg>
                <input type="text" placeholder="Filter teks log..." bind:value={logSearchQuery} style="font-size:12px" />
              </div>
            </div>

            <pre class="log-body">{logOutput ? (logSearchQuery ? logOutput.split('\n').filter(l => l.toLowerCase().includes(logSearchQuery.toLowerCase())).join('\n') : logOutput) : (isLoadingLog ? 'Sedang mengambil log dari server remote...' : 'Belum ada log yang diambil. Klik "Tarik Log" atau pilih preset di atas.')}</pre>
          </div>
        {/if}
      </div>

    {:else if currentNav === 'settings'}
      <SettingsView />
    {/if}
  </main>
</div>

{#if liveToast}
  <div
    style="position:fixed; bottom:20px; right:24px; z-index:2000; width:360px; background:var(--panel); border:1px solid {liveToast.level === 'critical' ? 'var(--crit)' : liveToast.level === 'warning' ? 'var(--warn)' : 'var(--ok)'}; border-radius:7px; padding:12px 16px; box-shadow:0 12px 32px rgba(0,0,0,0.55)"
  >
    <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:4px">
      <span style="font-weight:600; font-size:13px; color:{liveToast.level === 'critical' ? 'var(--crit)' : liveToast.level === 'warning' ? 'var(--warn)' : 'var(--ok)'}">
        {liveToast.title}
      </span>
      <button class="btn ghost" style="height:20px; padding:0 6px; font-size:11px" on:click={() => liveToast = null}>✕</button>
    </div>
    <div style="font-size:12.5px; color:var(--text)">{liveToast.body}</div>
    <div class="num" style="font-size:11px; color:var(--dim); margin-top:4px">{liveToast.time}</div>
  </div>
{/if}

<!-- Modal Tambah / Edit Server -->
{#if showModal}
  <div class="modal-overlay" role="presentation" on:click|self={() => showModal = false}>
    <div class="modal-box">
      <div class="m-h">
        <h2>{isEditing ? 'Edit Server' : 'Tambah Server Baru'}</h2>
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showModal = false}>✕</button>
      </div>

      <div class="m-b">
        <div class="row3">
          <div class="form-group">
            <label for="srv-name">Nama Server</label>
            <input id="srv-name" type="text" placeholder="Misal: web-prod-01" bind:value={formData.name} />
          </div>
          <div class="form-group">
            <label for="srv-tags">Tag (pisahkan koma)</label>
            <input id="srv-tags" type="text" placeholder="production, staging" bind:value={tagInput} />
          </div>
        </div>

        <div class="row2">
          <div class="form-group">
            <label for="srv-host">Host atau IP Address</label>
            <input id="srv-host" type="text" placeholder="10.0.1.20 atau domain.com" bind:value={formData.host} />
          </div>
          <div class="form-group">
            <label for="srv-port">Port SSH</label>
            <input id="srv-port" type="number" bind:value={formData.port} />
          </div>
        </div>

        <div class="row3">
          <div class="form-group">
            <label for="srv-user">Username SSH</label>
            <input id="srv-user" type="text" placeholder="deploy atau monitor" bind:value={formData.user} />
            <div class="note">Disarankan user read-only tanpa sudo</div>
          </div>
          <div class="form-group">
            <label for="srv-jumphost">Jump Host (Opsional)</label>
            <select id="srv-jumphost" bind:value={formData.jumpHostId}>
              <option value="">(Langsung / Tanpa Jump Host)</option>
              {#each servers as js}
                <option value={js.id}>{js.name} ({js.host})</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="form-group">
          <div class="label" style="font-size:12px; font-weight:500; margin-bottom:6px; color:var(--text)">Metode Autentikasi</div>
          <div class="seg">
            <button class={formData.authType === 'key' ? 'on' : ''} on:click={() => formData.authType = 'key'}>Private Key</button>
            <button class={formData.authType === 'password' ? 'on' : ''} on:click={() => formData.authType = 'password'}>Password</button>
            <button class={formData.authType === 'agent' ? 'on' : ''} on:click={() => formData.authType = 'agent'}>SSH Agent</button>
          </div>
        </div>

        {#if formData.authType === 'key'}
          <div class="row3">
            <div class="form-group">
              <label for="srv-keypath">Path Private Key</label>
              <input id="srv-keypath" type="text" placeholder="C:\Users\admin\.ssh\id_ed25519" bind:value={formData.keyPath} />
              <div class="note">File tidak disalin; hanya path yang disimpan</div>
            </div>
            <div class="form-group">
              <label for="srv-passphrase">Passphrase {isEditing ? '(Kosongkan jika tak diubah)' : '(jika ada)'}</label>
              <input id="srv-passphrase" type="password" placeholder={isEditing ? 'Biarkan kosong jika tidak diubah' : 'Passphrase key'} bind:value={formData.passphrase} />
              <div class="note">Disimpan aman di Windows Credential Manager</div>
            </div>
          </div>
        {:else if formData.authType === 'password'}
          <div class="form-group">
            <label for="srv-pwd">Password SSH {isEditing ? '(Kosongkan jika tak diubah)' : ''}</label>
            <input id="srv-pwd" type="password" placeholder={isEditing ? 'Biarkan kosong jika tidak diubah' : 'Password user'} bind:value={formData.password} />
            <div class="note">Disimpan aman di Windows Credential Manager</div>
          </div>
        {/if}

        <!-- Fingerprint & Diagnostic Preview -->
        {#if testResult}
          {#if testResult.success}
            <div class="hk-box">
              <h4>Konfirmasi Fingerprint Host Key</h4>
              <div style="font-size:12px; color:var(--muted)">
                Pastikan fingerprint berikut cocok dengan host key server Anda:
              </div>
              <div class="fp">{testResult.hostKeyType} • {testResult.hostKeyFingerprint}</div>
              <div style="display:flex; gap:16px; font-size:12px; color:var(--dim)">
                <span>OS: <b>{testResult.osInfo || 'Linux'}</b></span>
                <span>Latency: <b>{testResult.latencyMs} ms</b></span>
                <span>systemd: <b>{testResult.hasSystemd ? 'Ada' : 'Tidak'}</b></span>
              </div>
            </div>
          {:else}
            <div style="padding:10px 14px; background:rgba(229,83,75,0.1); border:1px solid var(--crit); border-radius:6px; color:#f08a84; font-size:12px">
              Gagal terhubung: {testResult.error}
            </div>
          {/if}
        {/if}
      </div>

      <div class="m-f">
        <button class="btn" on:click={handleTestConnection} disabled={isTesting}>
          {isTesting ? 'Sedang mengetes...' : 'Tes Koneksi'}
        </button>
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showModal = false} disabled={isSavingServer}>Batal</button>
        <button class="btn primary" on:click={handleSaveServer} disabled={isSavingServer}>
          {isSavingServer ? 'Menyimpan...' : 'Simpan Server'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal Import ~/.ssh/config -->
{#if showImportModal}
  <div class="modal-overlay" role="presentation" on:click|self={() => showImportModal = false}>
    <div class="modal-box" style="width:580px">
      <div class="m-h">
        <h2>Import Host dari ~/.ssh/config</h2>
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showImportModal = false}>✕</button>
      </div>
      <div class="m-b">
        <div style="font-size:12.5px; color:var(--muted)">
          Ditemukan <b>{importedHosts.length}</b> host di konfigurasi SSH lokal. Pilih host yang ingin ditambahkan ke ServerWatch:
        </div>
        <div style="max-height:280px; overflow-y:auto; border:1px solid var(--line); border-radius:6px">
          {#each importedHosts as h, idx}
            <div style="display:flex; align-items:center; gap:12px; padding:10px 14px; border-bottom:1px solid var(--line)">
              <input type="checkbox" bind:checked={selectedImportIndices[idx]} />
              <div>
                <div style="font-weight:600; font-size:13px">{h.name}</div>
                <div style="font-family:var(--mono); color:var(--dim); font-size:11.5px">{h.user}@{h.host}:{h.port}</div>
              </div>
            </div>
          {/each}
        </div>
      </div>
      <div class="m-f">
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showImportModal = false}>Batal</button>
        <button class="btn primary" on:click={executeImport}>Tambahkan ke Monitoring</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal Tambah/Edit Database -->
{#if showDBModal}
  <div class="modal-overlay" role="presentation" on:click|self={() => showDBModal = false}>
    <div class="modal-box">
      <div class="m-h">
        <h2>{isEditingDB ? 'Edit Konfigurasi Database' : 'Tambah Database untuk Monitoring'}</h2>
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showDBModal = false}>✕</button>
      </div>

      <div class="m-b">
        <div class="form-group">
          <div class="label" style="font-size:12px; font-weight:500; margin-bottom:6px; color:var(--text)">Tipe Database</div>
          <div class="seg">
            <button class={dbFormData.type === 'mysql' ? 'on' : ''} on:click={() => { dbFormData.type = 'mysql'; dbFormData.port = 3306; }}>MySQL / MariaDB</button>
            <button class={dbFormData.type === 'postgres' ? 'on' : ''} on:click={() => { dbFormData.type = 'postgres'; dbFormData.port = 5432; }}>PostgreSQL</button>
            <button class={dbFormData.type === 'redis' ? 'on' : ''} on:click={() => { dbFormData.type = 'redis'; dbFormData.port = 6379; }}>Redis</button>
          </div>
        </div>

        <div class="row3">
          <div class="form-group">
            <label for="db-label">Nama Display / Label</label>
            <input id="db-label" type="text" placeholder="Misal: db-app-prod" bind:value={dbFormData.name} />
          </div>
          <div class="form-group">
            <label for="db-name">Nama Database (Spesifik)</label>
            <input id="db-name" type="text" placeholder="Misal: ecommerce_db" bind:value={dbFormData.databaseName} />
            <div class="note">Kosongkan untuk all databases</div>
          </div>
        </div>

        <div class="row2">
          <div class="form-group">
            <label for="db-host">Host atau Socket</label>
            <input id="db-host" type="text" placeholder="127.0.0.1" bind:value={dbFormData.host} />
          </div>
          <div class="form-group">
            <label for="db-port">Port</label>
            <input id="db-port" type="number" bind:value={dbFormData.port} />
          </div>
        </div>

        {#if dbFormData.type !== 'redis'}
          <div class="row3">
            <div class="form-group">
              <label for="db-user">Username DB</label>
              <input id="db-user" type="text" placeholder="root atau admin" bind:value={dbFormData.user} />
            </div>
            <div class="form-group">
              <label for="db-pwd">Password DB {isEditingDB ? '(Kosongkan jika tak diubah)' : '(Opsional)'}</label>
              <input id="db-pwd" type="password" placeholder="Password database" bind:value={dbPasswordInput} />
              <div class="note">Tersimpan aman di Windows Credential Manager</div>
            </div>
          </div>
        {:else}
          <div class="form-group">
            <label for="db-redis-auth">AUTH Password Redis (Opsional)</label>
            <input id="db-redis-auth" type="password" placeholder="Requirepass jika diatur" bind:value={dbPasswordInput} />
            <div class="note">Tersimpan aman di Windows Credential Manager</div>
          </div>
        {/if}

        {#if dbTestResult}
          <div style="margin-top:12px; padding:8px 12px; border-radius:4px; font-size:12px; background:{dbTestResult.ok ? 'rgba(78,201,176,0.12)' : 'rgba(241,76,76,0.12)'}; color:{dbTestResult.ok ? 'var(--ok)' : 'var(--crit)'}; border:1px solid {dbTestResult.ok ? 'var(--ok)' : 'var(--crit)'}">
            {dbTestResult.msg}
          </div>
        {/if}
      </div>

      <div class="m-f">
        <button class="btn" on:click={handleTestDBConnection} disabled={isTestingDB}>
          {#if isTestingDB}
            <i>●</i> Mengecek...
          {:else}
            ⚡ Tes Koneksi
          {/if}
        </button>
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showDBModal = false}>Batal</button>
        <button class="btn primary" on:click={handleSaveDatabase}>Simpan Database</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal Buat Jadwal Backup Otomatis -->
{#if showScheduleModal}
  <div class="modal-overlay" role="presentation" on:click|self={() => showScheduleModal = false}>
    <div class="modal-box">
      <div class="m-h">
        <h2>Buat Jadwal Backup Otomatis</h2>
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showScheduleModal = false}>✕</button>
      </div>

      <div class="m-b">
        <div class="form-group">
          <label for="sched-db">Pilih Target Database</label>
          <select id="sched-db" bind:value={scheduleFormData.databaseConfigId}>
            {#each databaseConfigs as db}
              <option value={db.id}>{db.name} ({db.type} - {db.databaseName || 'all'})</option>
            {/each}
          </select>
        </div>

        <div class="row3">
          <div class="form-group">
            <label for="sched-freq">Frekuensi Eksekusi</label>
            <select id="sched-freq" bind:value={scheduleFormData.frequency}>
              <option value="daily">Harian (Daily)</option>
              <option value="weekly">Mingguan (Weekly)</option>
              <option value="hourly">Setiap Jam (Hourly)</option>
            </select>
          </div>
          <div class="form-group">
            <label for="sched-time">Jam Eksekusi (WIB)</label>
            <input id="sched-time" type="time" bind:value={scheduleFormData.timeOfDay} />
            <div class="note">Disarankan saat jam sepi (off-peak, misal 02:00)</div>
          </div>
        </div>

        <div class="row3">
          <div class="form-group">
            <label for="sched-retention">Retensi (Hari)</label>
            <input id="sched-retention" type="number" min="1" max="90" bind:value={scheduleFormData.retentionDays} />
            <div class="note">Otomatis hapus file > N hari agar disk tidak penuh</div>
          </div>
          <div class="form-group">
            <label for="sched-target">Tujuan Penyimpanan</label>
            <select id="sched-target" bind:value={scheduleFormData.storageTarget}>
              <option value="remote">Server Remote (/var/backups)</option>
              <option value="local">Laptop Lokal (Offsite stream)</option>
              <option value="s3">Cloud Object Storage (S3 / Cloudflare R2 / MinIO)</option>
            </select>
          </div>
        </div>

        {#if scheduleFormData.storageTarget === 'local'}
          <div class="form-group">
            <label for="sched-localpath">Folder Penyimpanan di Laptop (Lokal)</label>
            <div style="display:flex; gap:8px">
              <input id="sched-localpath" type="text" bind:value={scheduleFormData.localPath} placeholder="C:\Users\...\ServerWatch\Backups (Default)" />
              <button class="btn" type="button" on:click={handleBrowseLocalFolder}>
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                Browse...
              </button>
            </div>
            <div class="note">Folder tujuan di laptop Anda. Kosongkan untuk menggunakan folder bawaan (~/ServerWatch/Backups).</div>
          </div>
        {/if}

        <div class="form-group">
          <label for="sched-remotepath">Direktori Sementara di Server Remote</label>
          <input id="sched-remotepath" type="text" bind:value={scheduleFormData.remotePath} placeholder="/var/backups/serverwatch" />
          <div class="note">Lokasi server melakukan dump sementara sebelum di-stream. Folder akan dibuat otomatis jika belum ada.</div>
        </div>
      </div>

      <div class="m-f">
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showScheduleModal = false}>Batal</button>
        <button class="btn primary" on:click={handleSaveSchedule}>Aktifkan Jadwal Backup</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal Hasil Auto-Detect Database -->
{#if showDetectedModal}
  {@const scannedSrv = servers.find(s => s.id === (detectedForServerId || dbServerId))}
  <div class="modal-overlay" role="presentation" on:click|self={() => showDetectedModal = false}>
    <div class="modal-box" style="width: 580px">
      <div class="m-h">
        <h2>Hasil Pemindaian Database{scannedSrv ? ` : ${scannedSrv.name} (${scannedSrv.host})` : ''}</h2>
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showDetectedModal = false}>✕</button>
      </div>

      <div class="m-b">
        {#if detectedDatabases.length === 0}
          <div style="padding: 24px; text-align:center; color:var(--dim)">
            <svg viewBox="0 0 24 24" width="36" height="36" fill="none" stroke="currentColor" stroke-width="1.5" style="margin-bottom:8px; opacity:0.6"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
            <div style="font-size:14px; color:var(--text); font-weight:600">Tidak ada database yang terdeteksi aktif</div>
            <div style="font-size:12px; margin-top:4px">Tidak ditemukan service MySQL (3306), PostgreSQL (5432), atau Redis (6379) yang sedang mendengarkan port di server ini.</div>
          </div>
        {:else}
          <div style="font-size:12.5px; color:var(--muted); margin-bottom:10px">
            Ditemukan <b>{detectedDatabases.length}</b> instans database yang berjalan di server:
          </div>
          <div style="display:flex; flex-direction:column; gap:12px; max-height:360px; overflow-y:auto">
            {#each detectedDatabases as item}
              <div style="border:1px solid var(--line); border-radius:6px; padding:12px 14px; background:var(--bg)">
                <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:6px">
                  <div style="font-weight:600; font-size:13.5px; display:flex; align-items:center; gap:8px">
                    <span class="st {item.serviceState === 'running' ? 'ok' : 'warn'}"></span>
                    {item.name}
                  </div>
                  <div style="display:flex; gap:6px; align-items:center">
                    {#if item.serviceState === 'stopped'}
                      <span class="tag" style="background:rgba(224,164,58,0.15); color:var(--warn); border:1px solid var(--warn); font-size:10px">SERVICE MATI</span>
                    {:else}
                      <span class="tag" style="background:rgba(63,182,122,0.15); color:var(--ok); border:1px solid var(--ok); font-size:10px">AKTIF</span>
                    {/if}
                    <span class="db-badge {item.type}">{item.type}</span>
                  </div>
                </div>
                <div style="font-size:11.5px; color:var(--dim); font-family:var(--mono); margin-bottom:6px">
                  Host: {item.host}:{item.port} • User default: {item.defaultUser || 'none'} {item.version ? '• ' + item.version : ''}
                </div>
                {#if item.serviceState === 'stopped'}
                  <div style="font-size:11.5px; color:#ecc070; background:rgba(224,164,58,0.08); padding:6px 10px; border-radius:4px; margin-bottom:8px">
                    ⚠️ Paket terinstal di WSL, tetapi service belum dijalankan. Jalankan perintah di WSL: <b style="font-family:var(--mono)">sudo service {item.type} start</b>
                  </div>
                {/if}
                {#if item.databases && item.databases.length > 0}
                  <div style="font-size:11.5px; color:var(--muted); margin-bottom:6px">
                    Database lokal ditemukan: <b>{item.databases.join(', ')}</b>
                  </div>
                {/if}
                <div style="display:flex; gap:8px; margin-top:8px; flex-wrap:wrap">
                  <button class="btn primary" style="font-size:11.5px; padding:3px 10px" on:click={() => handleAddDetectedDB(item)}>
                    + Tambahkan ({item.type})
                  </button>
                  {#if item.databases && item.databases.length > 0}
                    {#each item.databases as dbName}
                      <button class="btn ghost" style="font-size:11px; padding:2px 8px" on:click={() => handleAddDetectedDB(item, dbName)}>
                        + Tambah '{dbName}'
                      </button>
                    {/each}
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <div class="m-f">
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showDetectedModal = false}>Tutup</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal Eksplorasi Schema / Daftar Database -->
{#if showSchemasModal}
  <div class="modal-overlay" role="presentation" on:click|self={() => showSchemasModal = false}>
    <div class="modal-box" style="width: 760px; max-width: 92vw">
      <div class="m-h">
        <div style="display:flex; align-items:center; gap:8px">
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>
          <h2 style="margin:0; font-size:15px">
            {#if selectedSchemaForTables}
              Struktur Tabel & Index: {selectedSchemaForTables} ({currentDbForSchemas ? currentDbForSchemas.name : ''})
            {:else}
              Daftar Database di {currentDbForSchemas ? currentDbForSchemas.name : 'Server'}
            {/if}
          </h2>
        </div>
        <span class="spacer"></span>
        {#if selectedSchemaForTables}
          <button class="btn" style="height:26px; font-size:11.5px; margin-right:6px" on:click={() => selectedSchemaForTables = ''}>
            ← Kembali ke Daftar Database
          </button>
        {/if}
        <button class="btn ghost" on:click={() => showSchemasModal = false}>✕</button>
      </div>

      <div class="m-b">
        {#if selectedSchemaForTables}
          {#if isLoadingTables}
            <div style="padding: 40px 20px; text-align: center; color: var(--muted)">
              <div style="font-size:13.5px; font-weight:500; color:var(--text)">Mengambil statistik ukuran tabel & index...</div>
            </div>
          {:else if schemaTables.length === 0}
            <div style="padding: 32px; text-align: center; color: var(--dim)">
              Tidak ada tabel yang ditemukan pada schema <b>{selectedSchemaForTables}</b>.
            </div>
          {:else}
            <div style="max-height: 360px; overflow-y: auto; border: 1px solid var(--line); border-radius: 6px">
              <table>
                <thead>
                  <tr>
                    <th>Nama Tabel</th>
                    <th>Engine</th>
                    <th class="r">Estimasi Baris</th>
                    <th class="r">Ukuran Data</th>
                    <th class="r">Ukuran Index</th>
                    <th class="r">Total Disk</th>
                  </tr>
                </thead>
                <tbody>
                  {#each schemaTables as tbl}
                    <tr>
                      <td class="num" style="font-weight:600">{tbl.tableName}</td>
                      <td><span class="tag">{tbl.engine}</span></td>
                      <td class="r num">{tbl.rowCount.toLocaleString()}</td>
                      <td class="r num">{formatBytes(tbl.dataBytes)}</td>
                      <td class="r num" style="color:var(--accent)">{formatBytes(tbl.indexBytes)}</td>
                      <td class="r num" style="font-weight:600">{formatBytes(tbl.totalBytes)}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        {:else if isLoadingSchemas}
          <div style="padding: 40px 20px; text-align: center; color: var(--muted)">
            <svg class="spin" viewBox="0 0 24 24" width="30" height="30" fill="none" stroke="currentColor" stroke-width="2" style="margin-bottom: 12px; opacity:0.8"><circle cx="12" cy="12" r="10" stroke-dasharray="32" stroke-linecap="round"/></svg>
            <div style="font-size:13.5px; font-weight:500; color:var(--text)">Menghubungkan ke SSH & membaca informasi skema...</div>
            <div style="font-size:11.5px; color:var(--dim); margin-top:4px">Mengambil statistik dari information_schema.tables</div>
          </div>
        {:else if currentSchemas.length === 0}
          <div style="padding: 32px; text-align: center; color: var(--dim)">
            <svg viewBox="0 0 24 24" width="36" height="36" fill="none" stroke="currentColor" stroke-width="1.5" style="margin-bottom:8px; opacity:0.6"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
            <div style="font-size: 14px; font-weight: 600; color: var(--text); margin-bottom: 4px">Tidak ada database yang ditemukan</div>
            <div style="font-size: 12px; max-width: 440px; margin: 0 auto">
              Pastikan service database berjalan dan akun user memiliki izin (hak akses <code>SHOW DATABASES</code> / <code>SELECT ON information_schema</code>).
            </div>
          </div>
        {:else}
          <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:12px; gap:12px; flex-wrap:wrap">
            <div style="font-size:12.5px; color:var(--muted)">
              Total: <b>{currentSchemas.length}</b> database • Total Penggunaan Disk: <b style="color:var(--text)">{formatBytes(totalSchemasSize)}</b>
            </div>
            <input 
              type="text" 
              placeholder="🔍 Cari database..." 
              bind:value={schemaSearchQuery} 
              style="width: 200px; padding: 5px 10px; font-size: 12px; border-radius: 4px; border: 1px solid var(--line); background: var(--bg); color: var(--text)"
            />
          </div>

          <div style="max-height: 360px; overflow-y: auto; border: 1px solid var(--line); border-radius: 6px">
            <table>
              <thead>
                <tr>
                  <th>Nama Database</th>
                  <th style="width: 110px; text-align: right">Jumlah Tabel</th>
                  <th style="width: 120px; text-align: right">Ukuran di Disk</th>
                  <th style="width: 240px; text-align: right">Aksi Cepat</th>
                </tr>
              </thead>
              <tbody>
                {#each filteredSchemas as schema}
                  <tr>
                    <td>
                      <div style="display:flex; align-items:center; gap:6px">
                        <span style="font-family:var(--mono); font-weight:600; font-size:12.5px">{schema.name}</span>
                        {#if isSystemDb(schema.name)}
                          <span class="tag" style="font-size:9.5px; padding:1px 5px; opacity:0.6">Sistem</span>
                        {/if}
                      </div>
                    </td>
                    <td style="text-align: right; font-family:var(--mono); font-size:12px; color:var(--muted)">
                      {schema.tableCount !== undefined ? schema.tableCount + ' tabel' : '-'}
                    </td>
                    <td style="text-align: right; font-family:var(--mono); font-size:12px; font-weight:500">
                      {formatBytes(schema.sizeBytes)}
                    </td>
                    <td style="text-align: right">
                      <div style="display:flex; justify-content:flex-end; gap:5px">
                        <button
                          class="btn ghost"
                          style="font-size:11px; padding:2px 8px"
                          on:click={() => handleInspectTables(schema.name)}
                          title="Inspeksi ukuran tabel & index di dalam database ini"
                        >
                          Tabel
                        </button>
                        <button 
                          class="btn primary" 
                          style="font-size:11px; padding:2px 8px" 
                          on:click={() => currentDbForSchemas && handleManualBackup(currentDbForSchemas.id, 'remote', schema.name)}
                          disabled={isBackingUp[`${currentDbForSchemas?.id}_${schema.name}`]}
                          title="Backup database ini sekarang ke server"
                        >
                          {#if isBackingUp[`${currentDbForSchemas?.id}_${schema.name}`]}
                            <i>●</i> Dump...
                          {:else}
                            ⚡ Backup
                          {/if}
                        </button>
                        <button 
                          class="btn ghost" 
                          style="font-size:11px; padding:2px 8px" 
                          on:click={() => handleScheduleSchema(schema.name)}
                          title="Jadwalkan backup otomatis untuk database ini"
                        >
                          + Jadwal
                        </button>
                        <button 
                          class="btn ghost" 
                          style="font-size:11px; padding:2px 8px" 
                          on:click={() => handleMonitorSchema(schema.name)}
                          title="Tambahkan database ini ke monitoring individual"
                        >
                          + Pantau
                        </button>
                      </div>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>

      <div class="m-f">
        {#if currentDbForSchemas && !selectedSchemaForTables}
          <button class="btn ghost" on:click={() => { if (currentDbForSchemas) handleViewSchemas(currentDbForSchemas); }} disabled={isLoadingSchemas} style="font-size:12px">
            🔄 Muat Ulang
          </button>
        {/if}
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showSchemasModal = false}>Tutup</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal Restore / Rollback Database Wizard -->
{#if showRestoreModal && restoreLogItem}
  {@const vRes = verifyResultMap[restoreLogItem.id]}
  <div class="modal-overlay" role="presentation" on:click|self={() => showRestoreModal = false}>
    <div class="modal-box" style="width: 580px">
      <div class="m-h">
        <h2>Wizard Restore / Rollback Database</h2>
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showRestoreModal = false} aria-label="Tutup wizard restore" title="Tutup">✕</button>
      </div>

      <div class="m-b">
        <div style="border:1px solid var(--line); border-radius:var(--radius-md); padding:12px 14px; background:var(--bg); margin-bottom:14px">
          <div style="font-size:12px; color:var(--dim); margin-bottom:4px">Arsip Backup Sumber:</div>
          <div style="font-weight:600; font-size:14px; color:var(--text)">
            Database: <span class="num" style="color:var(--accent)">{restoreLogItem.databaseName}</span>
            <span class="tag" style="margin-left:8px">{(restoreLogItem.fileSizeBytes / (1024 * 1024)).toFixed(2)} MB</span>
          </div>
          <div class="num" style="font-size:11.5px; color:var(--muted); margin-top:4px; word-break:break-all">
            Waktu Dump: {new Date(restoreLogItem.startedAt).toLocaleString()}<br/>
            Lokasi File: {restoreLogItem.filePath}
          </div>

          <div style="margin-top:10px; padding-top:8px; border-top:1px solid var(--line); display:flex; align-items:center; justify-content:space-between">
            {#if isVerifyingBackup[restoreLogItem.id]}
              <span style="font-size:12px; color:var(--warn)"><i>●</i> Memverifikasi keutuhan arsip (gzip -t & SHA-256)...</span>
            {:else if vRes}
              <span style="font-size:12px; font-weight:500; color:{vRes.valid ? 'var(--ok)' : 'var(--crit)'}">
                {vRes.valid ? '✓ Pre-flight Check: Arsip .sql.gz utuh & siap direstorasi' : '✕ ' + vRes.message}
              </span>
            {:else}
              <span style="font-size:12px; color:var(--dim)">Belum diverifikasi</span>
            {/if}
            <button class="btn ghost" style="height:28px; font-size:11.5px; padding:0 10px" on:click={() => { if (restoreLogItem) handleVerifyBackup(restoreLogItem); }} disabled={isVerifyingBackup[restoreLogItem.id]} title="Verifikasi ulang keutuhan arsip backup">
              Cek Ulang
            </button>
          </div>
        </div>

        <div class="form-group">
          <div class="label" style="font-size:12px; font-weight:500; margin-bottom:6px; color:var(--text)">Mode Pemulihan (Target Restore)</div>
          <div class="seg">
            <button class={restoreTargetMode === 'inplace' ? 'on' : ''} on:click={() => restoreTargetMode = 'inplace'}>
              Restore In-Place ({restoreLogItem.databaseName})
            </button>
            <button class={restoreTargetMode === 'custom' ? 'on' : ''} on:click={() => restoreTargetMode = 'custom'}>
              Restore ke Database / Staging Baru
            </button>
          </div>
        </div>

        {#if restoreTargetMode === 'custom'}
          <div class="form-group">
            <label for="restore-custom-db">Nama Database Tujuan Baru (SEC-16)</label>
            <input
              id="restore-custom-db"
              type="text"
              placeholder="Contoh: app_staging_restore"
              bind:value={restoreCustomDBName}
            />
            <div class="note">Database baru akan dibuat otomatis (CREATE DATABASE IF NOT EXISTS) tanpa menimpa database produksi utama.</div>
          </div>
        {:else}
          <div style="padding:10px 12px; border-radius:var(--radius-md); background:rgba(229,83,75,0.1); border:1px solid rgba(229,83,75,0.35); color:var(--crit); font-size:12px; margin-top:8px">
            <b>Perhatian:</b> Mode In-Place akan menimpa tabel di dalam database <b>{restoreLogItem.databaseName}</b> dengan isi dari arsip backup ini. Dialog konfirmasi sistem operasi (SEC-06) akan muncul sebelum eksekusi dimulai.
          </div>
        {/if}
      </div>

      <div class="m-f">
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showRestoreModal = false}>Batal</button>
        <button
          class="btn primary"
          on:click={handleExecuteRestore}
          disabled={isRestoringDB || (vRes && !vRes.valid)}
        >
          {isRestoringDB ? 'Sedang Merestorasi...' : '⏪ Jalankan Restore Database'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal Proses Backup Database Berjalan -->
{#if showActiveBackupsModal}
  <div class="modal-overlay" role="presentation" on:click|self={() => showActiveBackupsModal = false}>
    <div class="modal-box" style="width: 640px; max-width: 95vw;">
      <div class="m-h">
        <div style="display:flex; align-items:center; gap:8px;">
          <span style="font-size:18px;">⚡</span>
          <div>
            <h2 style="margin:0; font-size:16px;">Proses Backup Database Berjalan</h2>
            <div style="font-size:11.5px; color:var(--dim)">Daftar proses backup yang sedang berjalan di latar belakang</div>
          </div>
        </div>
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showActiveBackupsModal = false} aria-label="Tutup modal proses backup" title="Tutup">✕</button>
      </div>

      <div class="m-b" style="max-height: 480px; overflow-y: auto;">
        {#if activeBackupList.length === 0}
          <div style="text-align: center; padding: 40px 20px; color: var(--dim);">
            <svg viewBox="0 0 24 24" width="36" height="36" fill="none" stroke="currentColor" stroke-width="1.5" style="margin-bottom: 10px; opacity: 0.5;">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
              <polyline points="22 4 12 14.01 9 11.01"/>
            </svg>
            <div style="font-size: 14px; font-weight: 600; color: var(--text);">Tidak ada backup yang sedang berjalan</div>
            <div style="font-size: 12px; margin-top: 4px;">Semua antrean backup telah selesai atau belum ada pekerjaan manual/terjadwal yang aktif.</div>
          </div>
        {:else}
          <div style="display: flex; flex-direction: column; gap: 14px;">
            {#each activeBackupList as job (job.jobId)}
              <div style="border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 14px; background: var(--bg);">
                <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:8px;">
                  <div style="display:flex; align-items:center; gap:8px;">
                    <span style="display:inline-block; width:8px; height:8px; border-radius:50%; background:#38bdf8; box-shadow:0 0 6px #38bdf8;"></span>
                    <b style="font-size: 14px; color: var(--text)">{job.databaseName}</b>
                    <span style="font-size: 11px; padding: 1px 7px; border-radius: var(--radius-sm); background: rgba(56,189,248,0.15); color: #38bdf8;">{job.serverName || 'Server'}</span>
                  </div>
                  <div style="font-size: 12px; font-family: monospace; color: var(--accent); font-weight: 600;">
                    {job.progressPercent}%
                  </div>
                </div>

                <!-- Stepper Progress Indicators -->
                <div style="display: flex; gap: 4px; margin-bottom: 10px;">
                  <span style="flex:1; text-align:center; font-size:10px; padding:3px 0; border-radius:var(--radius-sm); {job.progressPercent >= 10 ? 'background:#0284c7; color:#fff' : 'background:var(--panel-2); color:var(--dim)'}">
                    1. Pre-flight
                  </span>
                  <span style="flex:1; text-align:center; font-size:10px; padding:3px 0; border-radius:var(--radius-sm); {job.progressPercent >= 35 ? 'background:#0284c7; color:#fff' : 'background:var(--panel-2); color:var(--dim)'}">
                    2. Dump DB
                  </span>
                  <span style="flex:1; text-align:center; font-size:10px; padding:3px 0; border-radius:var(--radius-sm); {job.progressPercent >= 70 ? 'background:#0284c7; color:#fff' : 'background:var(--panel-2); color:var(--dim)'}">
                    3. Verifikasi
                  </span>
                  <span style="flex:1; text-align:center; font-size:10px; padding:3px 0; border-radius:var(--radius-sm); {job.progressPercent >= 85 ? 'background:#0284c7; color:#fff' : 'background:var(--panel-2); color:var(--dim)'}">
                    4. Salin/Offsite
                  </span>
                  <span style="flex:1; text-align:center; font-size:10px; padding:3px 0; border-radius:var(--radius-sm); {job.progressPercent >= 100 ? 'background:#10b981; color:#fff' : 'background:var(--panel-2); color:var(--dim)'}">
                    5. Selesai
                  </span>
                </div>

                <!-- Progress Bar -->
                <div style="height: 6px; border-radius: var(--radius-sm); background: var(--line); overflow: hidden; margin-bottom: 10px;">
                  <div style="height: 100%; width: {job.progressPercent}%; background: #38bdf8; transition: width 0.3s ease;"></div>
                </div>

                <!-- Details & Current Message -->
                <div style="font-size: 12px; color: var(--text); background: var(--panel-2); padding: 8px 10px; border-radius: var(--radius-md); border: 1px solid var(--line); margin-bottom: 6px;">
                  {job.message || 'Memproses backup database...'}
                </div>

                <div style="display: flex; justify-content: space-between; font-size: 11px; color: var(--dim); margin-top: 4px;">
                  <span>Target: <b style="color:var(--text)">{job.storageTarget === 'local' ? 'Lokal (ServerWatch)' : job.storageTarget === 's3' ? 'AWS S3 Cloud' : 'Remote SSH'}</b></span>
                  <span>Waktu Berjalan: <b style="color:var(--text)">{Math.round(job.elapsedSec || 0)} detik</b></span>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <div class="m-f" style="display:flex; align-items:center; justify-content:space-between;">
        <span style="font-size: 11px; color: var(--dim)">
          ℹ️ Proses backup terus berjalan di latar belakang meskipun modal ini ditutup.
        </span>
        <button class="btn primary" on:click={() => showActiveBackupsModal = false}>Tutup</button>
      </div>
    </div>
  </div>
{/if}
