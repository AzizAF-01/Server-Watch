<script lang="ts">
  import { onMount } from 'svelte';
  import {
    GetAppSettings,
    SaveAppSettings,
    TestNotificationChannel,
    TestS3BackupConnection,
    ExportReport,
    CleanOldMetricsNow
  } from '../wailsjs/go/main/App';
  import type { model } from '../wailsjs/go/models';

  let activeTab: 'notifications' | 's3' | 'reports' | 'desktop' = 'notifications';
  let isLoading = false;
  let isSaving = false;
  let isTestingS3 = false;
  let testingChannel: Record<string, boolean> = {};
  let isExporting: Record<string, boolean> = {};
  let isCleaning = false;
  let feedbackBanner: { type: 'ok' | 'err'; text: string } | null = null;

  let settings: model.AppSettings = {
    retentionDays: 30,
    autoStartOs: false,
    runInBackground: true,
    confirmSshCommands: true,
    notifications: {
      desktopEnabled: true,
      telegramEnabled: false,
      telegramChatId: '',
      telegramBotToken: '',
      hasTelegramToken: false,
      webhookEnabled: false,
      webhookUrl: '',
      hasWebhookUrl: false,
      notifyOnRecovery: true,
      notifyOnBackup: true
    },
    s3Backup: {
      enabled: false,
      endpoint: '',
      region: 'auto',
      bucket: '',
      pathPrefix: 'serverwatch-backups',
      accessKeyId: '',
      secretAccessKey: '',
      hasSecretKey: false
    }
  } as any;

  onMount(async () => {
    await loadSettings();
  });

  export async function loadSettings() {
    isLoading = true;
    try {
      const cfg = await GetAppSettings();
      if (cfg) {
        settings = {
          retentionDays: cfg.retentionDays || 30,
          autoStartOs: Boolean(cfg.autoStartOs),
          runInBackground: cfg.runInBackground !== false,
          confirmSshCommands: cfg.confirmSshCommands !== false,
          notifications: {
            desktopEnabled: cfg.notifications?.desktopEnabled !== false,
            telegramEnabled: Boolean(cfg.notifications?.telegramEnabled),
            telegramChatId: cfg.notifications?.telegramChatId || '',
            telegramBotToken: '',
            hasTelegramToken: Boolean(cfg.notifications?.hasTelegramToken),
            webhookEnabled: Boolean(cfg.notifications?.webhookEnabled),
            webhookUrl: '',
            hasWebhookUrl: Boolean(cfg.notifications?.hasWebhookUrl),
            notifyOnRecovery: cfg.notifications?.notifyOnRecovery !== false,
            notifyOnBackup: cfg.notifications?.notifyOnBackup !== false
          },
          s3Backup: {
            enabled: Boolean(cfg.s3Backup?.enabled),
            endpoint: cfg.s3Backup?.endpoint || '',
            region: cfg.s3Backup?.region || 'auto',
            bucket: cfg.s3Backup?.bucket || '',
            pathPrefix: cfg.s3Backup?.pathPrefix || 'serverwatch-backups',
            accessKeyId: cfg.s3Backup?.accessKeyId || '',
            secretAccessKey: '',
            hasSecretKey: Boolean(cfg.s3Backup?.hasSecretKey)
          }
        } as any;
      }
    } catch (err: any) {
      showFeedback('err', 'Gagal memuat pengaturan: ' + (err?.message || String(err)));
    } finally {
      isLoading = false;
    }
  }

  function showFeedback(type: 'ok' | 'err', text: string) {
    feedbackBanner = { type, text };
    setTimeout(() => {
      if (feedbackBanner && feedbackBanner.text === text) {
        feedbackBanner = null;
      }
    }, 5500);
  }

  async function handleSaveSettings() {
    isSaving = true;
    try {
      settings.retentionDays = Number(settings.retentionDays) || 30;
      await SaveAppSettings(settings);
      showFeedback('ok', 'Pengaturan berhasil disimpan. Token rahasia tersimpan aman di OS Keychain (SEC-08).');
      await loadSettings();
    } catch (err: any) {
      showFeedback('err', 'Gagal menyimpan pengaturan: ' + (err?.message || String(err)));
    } finally {
      isSaving = false;
    }
  }

  async function handleTestChannel(channel: 'desktop' | 'telegram' | 'webhook') {
    testingChannel[channel] = true;
    try {
      await TestNotificationChannel(channel, settings.notifications);
      showFeedback('ok', `Pesan uji coba ke channel '${channel.toUpperCase()}' berhasil dikirim.`);
    } catch (err: any) {
      showFeedback('err', `Tes channel ${channel.toUpperCase()} gagal: ` + (err?.message || String(err)));
    } finally {
      testingChannel[channel] = false;
    }
  }

  async function handleTestS3() {
    isTestingS3 = true;
    try {
      await TestS3BackupConnection(settings.s3Backup);
      showFeedback('ok', `Koneksi ke bucket '${settings.s3Backup.bucket}' berhasil diverifikasi dengan AWS Signature V4.`);
    } catch (err: any) {
      showFeedback('err', 'Tes koneksi S3/R2 gagal: ' + (err?.message || String(err)));
    } finally {
      isTestingS3 = false;
    }
  }

  async function handleExportReport(format: 'csv' | 'html') {
    isExporting[format] = true;
    try {
      const savedPath = await ExportReport(format);
      if (savedPath) {
        showFeedback('ok', `Laporan berhasil diekspor ke: ${savedPath}`);
      }
    } catch (err: any) {
      const msg = err?.message || String(err);
      if (!msg.includes('dibatalkan')) {
        showFeedback('err', 'Gagal mengekspor laporan: ' + msg);
      }
    } finally {
      isExporting[format] = false;
    }
  }

  async function handleCleanOldMetrics() {
    isCleaning = true;
    try {
      await CleanOldMetricsNow();
      showFeedback('ok', `Data metrik yang lebih tua dari ${settings.retentionDays} hari telah dibersihkan dari SQLite.`);
    } catch (err: any) {
      showFeedback('err', 'Gagal membersihkan metrik lama: ' + (err?.message || String(err)));
    } finally {
      isCleaning = false;
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && feedbackBanner) {
      feedbackBanner = null;
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="topbar">
  <h1>Pengaturan, Notifikasi & Laporan</h1>
  <span class="sub">Integrasi Telegram/Webhook, Export Laporan, dan Preferensi Desktop</span>
  <span class="spacer"></span>
  <button class="btn primary" on:click={handleSaveSettings} disabled={isSaving || isLoading}>
    {isSaving ? 'Menyimpan...' : 'Simpan Pengaturan'}
  </button>
</div>

<div class="content">
  {#if feedbackBanner}
    <div
      class="panel"
      style="padding:10px 14px; margin-bottom:14px; display:flex; align-items:center; justify-content:space-between; border-color:{feedbackBanner.type === 'ok' ? 'var(--ok)' : 'var(--crit)'}; color:{feedbackBanner.type === 'ok' ? 'var(--ok)' : 'var(--crit)'}"
    >
      <span>{feedbackBanner.text}</span>
      <button class="btn ghost" style="height:28px; padding:0 12px; font-size:12px" on:click={() => feedbackBanner = null} aria-label="Tutup notifikasi">Tutup</button>
    </div>
  {/if}

  <div class="subnav-tabs" role="tablist" aria-label="Kategori Pengaturan">
    <button
      role="tab"
      id="tab-notifications"
      aria-controls="panel-notifications"
      aria-selected={activeTab === 'notifications'}
      class={activeTab === 'notifications' ? 'active' : ''}
      on:click={() => activeTab = 'notifications'}
    >
      <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/>
      </svg>
      Channel Notifikasi Eksternal (FR-25)
    </button>

    <button
      role="tab"
      id="tab-s3"
      aria-controls="panel-s3"
      aria-selected={activeTab === 's3'}
      class={activeTab === 's3' ? 'active' : ''}
      on:click={() => activeTab = 's3'}
    >
      <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"/>
      </svg>
      Offsite Backup Cloud (S3 / R2 / MinIO)
    </button>

    <button
      role="tab"
      id="tab-reports"
      aria-controls="panel-reports"
      aria-selected={activeTab === 'reports'}
      class={activeTab === 'reports' ? 'active' : ''}
      on:click={() => activeTab = 'reports'}
    >
      <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      </svg>
      Export Laporan & Retensi Data (FR-29 & FR-31)
    </button>

    <button
      role="tab"
      id="tab-desktop"
      aria-controls="panel-desktop"
      aria-selected={activeTab === 'desktop'}
      class={activeTab === 'desktop' ? 'active' : ''}
      on:click={() => activeTab = 'desktop'}
    >
      <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/>
      </svg>
      Aplikasi Desktop & Background (FR-32 & FR-33)
    </button>
  </div>

  {#if activeTab === 'notifications'}
    <div id="panel-notifications" role="tabpanel" aria-labelledby="tab-notifications">
      <div style="display:grid; grid-template-columns: 1fr 1fr; gap:16px; margin-bottom:16px">
        <!-- Telegram Bot Card -->
        <div class="panel" style="padding:18px">
          <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:14px">
            <div>
              <div style="font-weight:600; font-size:14.5px; color:var(--text)">Telegram Bot Alerting</div>
              <div style="font-size:12px; color:var(--dim)">Kirim peringatan insiden & backup langsung ke chat/grup Telegram</div>
            </div>
            <label class="switch">
              <input type="checkbox" aria-label="Aktifkan Telegram Bot Alerting" bind:checked={settings.notifications.telegramEnabled} />
              <span class="slider"></span>
            </label>
          </div>

          <div class="form-group" style="margin-bottom:12px">
            <label for="tg-token">
              Telegram Bot Token (SEC-08 Keychain)
              {#if settings.notifications.hasTelegramToken}
                <span class="tag" style="margin-left:6px; color:var(--ok)">Tersimpan di Keychain</span>
              {/if}
            </label>
            <input
              id="tg-token"
              type="password"
              placeholder={settings.notifications.hasTelegramToken ? '•••••••••••••••• (Kosongkan jika tidak diubah)' : 'Contoh: 123456789:ABCdefGHIjklMNOpqrSTUvwxYZ'}
              bind:value={settings.notifications.telegramBotToken}
            />
            <div class="note">Token disimpan di Windows Credential Manager dan tidak pernah diekspos ke file database.</div>
          </div>

          <div class="form-group" style="margin-bottom:14px">
            <label for="tg-chat">Telegram Chat ID / Group ID</label>
            <input
              id="tg-chat"
              type="text"
              placeholder="Contoh: -1001234567890 atau 987654321"
              bind:value={settings.notifications.telegramChatId}
            />
          </div>

          <div style="display:flex; justify-content:flex-end">
            <button
              class="btn"
              on:click={() => handleTestChannel('telegram')}
              disabled={testingChannel['telegram']}
            >
              {testingChannel['telegram'] ? 'Mengirim...' : 'Kirim Pesan Tes Telegram'}
            </button>
          </div>
        </div>

        <!-- Discord / Slack Webhook Card -->
        <div class="panel" style="padding:18px">
          <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:14px">
            <div>
              <div style="font-weight:600; font-size:14.5px; color:var(--text)">Webhook (Discord / Slack / Custom)</div>
              <div style="font-size:12px; color:var(--dim)">Kirim payload JSON otomatis ke channel tim DevOps</div>
            </div>
            <label class="switch">
              <input type="checkbox" aria-label="Aktifkan Notifikasi Webhook" bind:checked={settings.notifications.webhookEnabled} />
              <span class="slider"></span>
            </label>
          </div>

          <div class="form-group" style="margin-bottom:14px">
            <label for="wh-url">
              Webhook URL (SEC-08 Keychain)
              {#if settings.notifications.hasWebhookUrl}
                <span class="tag" style="margin-left:6px; color:var(--ok)">Tersimpan di Keychain</span>
              {/if}
            </label>
            <input
              id="wh-url"
              type="password"
              placeholder={settings.notifications.hasWebhookUrl ? '•••••••••••••••• (Kosongkan jika tidak diubah)' : 'https://discord.com/api/webhooks/... atau https://hooks.slack.com/...'}
              bind:value={settings.notifications.webhookUrl}
            />
            <div class="note">Kompatibel langsung dengan format webhook Discord, Slack, dan Mattermost.</div>
          </div>

          <div style="display:flex; justify-content:flex-end; margin-top:46px">
            <button
              class="btn"
              on:click={() => handleTestChannel('webhook')}
              disabled={testingChannel['webhook']}
            >
              {testingChannel['webhook'] ? 'Mengirim...' : 'Kirim Pesan Tes Webhook'}
            </button>
          </div>
        </div>
      </div>

      <!-- Notification Behavior Rules -->
      <div class="panel" style="padding:18px">
        <div style="font-weight:600; font-size:14px; margin-bottom:12px">Preferensi Pemicu Notifikasi (FR-26 & FR-44)</div>
        <div style="display:grid; gap:12px">
          <div style="display:flex; align-items:center; justify-content:space-between; padding-bottom:10px; border-bottom:1px solid var(--line)">
            <div>
              <div style="font-weight:500">Notifikasi Banner Real-Time di Aplikasi Desktop</div>
              <div style="font-size:12px; color:var(--dim)">Tampilkan banner pop-up di layar ketika alert kritis/warning terjadi</div>
            </div>
            <div style="display:flex; align-items:center; gap:10px">
              <button class="btn" style="height:28px; font-size:11.5px; padding:0 10px" on:click={() => handleTestChannel('desktop')}>Tes Banner</button>
              <label class="switch">
                <input type="checkbox" aria-label="Aktifkan Notifikasi Banner Real-Time Desktop" bind:checked={settings.notifications.desktopEnabled} />
                <span class="slider"></span>
              </label>
            </div>
          </div>

          <div style="display:flex; align-items:center; justify-content:space-between; padding-bottom:10px; border-bottom:1px solid var(--line)">
            <div>
              <div style="font-weight:500">Kirim Notifikasi Saat Insiden Pulih (Resolved) (FR-26)</div>
              <div style="font-size:12px; color:var(--dim)">Beri tahu ketika penggunaan CPU, RAM, Disk, SSH, atau Website kembali normal</div>
            </div>
            <label class="switch">
              <input type="checkbox" aria-label="Aktifkan Notifikasi Saat Insiden Pulih" bind:checked={settings.notifications.notifyOnRecovery} />
              <span class="slider"></span>
            </label>
          </div>

          <div style="display:flex; align-items:center; justify-content:space-between">
            <div>
              <div style="font-weight:500">Kirim Notifikasi Status Backup Database (FR-44)</div>
              <div style="font-size:12px; color:var(--dim)">Beri tahu hasil eksekusi dump backup database (sukses maupun gagal)</div>
            </div>
            <label class="switch">
              <input type="checkbox" aria-label="Aktifkan Notifikasi Status Backup Database" bind:checked={settings.notifications.notifyOnBackup} />
              <span class="slider"></span>
            </label>
          </div>
        </div>
      </div>
    </div>

  {:else if activeTab === 's3'}
    <div id="panel-s3" role="tabpanel" aria-labelledby="tab-s3" class="panel" style="padding:18px; max-width:740px">
      <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:14px">
        <div>
          <div style="font-weight:600; font-size:14.5px; color:var(--text)">Integrasi Cloud Object Storage (AWS S3 / Cloudflare R2 / MinIO)</div>
          <div style="font-size:12px; color:var(--dim)">
            Unggah arsip backup database secara otomatis ke bucket S3 menggunakan protokol AWS Signature v4 tanpa perlu memasang <code>aws-cli</code> di server target.
          </div>
        </div>
        <label class="switch">
          <input type="checkbox" aria-label="Aktifkan Integrasi Cloud S3 Backup" bind:checked={settings.s3Backup.enabled} />
          <span class="slider"></span>
        </label>
      </div>

      <div class="row3">
        <div class="form-group">
          <label for="s3-endpoint">Endpoint URL S3 / R2</label>
          <input
            id="s3-endpoint"
            type="text"
            placeholder="https://<account_id>.r2.cloudflarestorage.com atau https://s3.ap-southeast-1.amazonaws.com"
            bind:value={settings.s3Backup.endpoint}
          />
          <div class="note">Mendukung AWS S3, Cloudflare R2, MinIO, DigitalOcean Spaces, dan Wasabi</div>
        </div>
        <div class="form-group">
          <label for="s3-region">Region</label>
          <input
            id="s3-region"
            type="text"
            placeholder="auto atau ap-southeast-1"
            bind:value={settings.s3Backup.region}
          />
          <div class="note">Gunakan <code>auto</code> untuk Cloudflare R2</div>
        </div>
      </div>

      <div class="row3">
        <div class="form-group">
          <label for="s3-bucket">Nama Bucket</label>
          <input
            id="s3-bucket"
            type="text"
            placeholder="Contoh: prod-db-backups"
            bind:value={settings.s3Backup.bucket}
          />
        </div>
        <div class="form-group">
          <label for="s3-prefix">Folder / Path Prefix di Bucket</label>
          <input
            id="s3-prefix"
            type="text"
            placeholder="serverwatch-backups"
            bind:value={settings.s3Backup.pathPrefix}
          />
        </div>
      </div>

      <div class="row3">
        <div class="form-group">
          <label for="s3-ak">Access Key ID</label>
          <input
            id="s3-ak"
            type="text"
            placeholder="AKIAIOSFODNN7EXAMPLE"
            bind:value={settings.s3Backup.accessKeyId}
          />
        </div>
        <div class="form-group">
          <label for="s3-sk">
            Secret Access Key (SEC-08 Keychain)
            {#if settings.s3Backup.hasSecretKey}
              <span class="tag" style="margin-left:6px; color:var(--ok)">Tersimpan di Keychain</span>
            {/if}
          </label>
          <input
            id="s3-sk"
            type="password"
            placeholder={settings.s3Backup.hasSecretKey ? '•••••••••••••••• (Kosongkan jika tidak diubah)' : 'Masukkan Secret Access Key S3/R2'}
            bind:value={settings.s3Backup.secretAccessKey}
          />
          <div class="note">Disimpan terisolasi di Windows Credential Manager (SEC-08 & SEC-15)</div>
        </div>
      </div>

      <div style="display:flex; justify-content:space-between; align-items:center; margin-top:12px; padding-top:12px; border-top:1px solid var(--line)">
        <span style="font-size:12px; color:var(--muted)">
          Setelah disimpan, Anda dapat memilih tujuan <b>Cloud Object Storage (S3 / R2)</b> saat membuat jadwal backup atau mengunggah file dari tab Riwayat Log Backup.
        </span>
        <div style="display:flex; gap:8px">
          <button class="btn" on:click={handleTestS3} disabled={isTestingS3}>
            {isTestingS3 ? 'Mengecek Bucket...' : 'Tes Koneksi Bucket S3/R2'}
          </button>
          <button class="btn primary" on:click={handleSaveSettings} disabled={isSaving}>
            {isSaving ? 'Menyimpan...' : 'Simpan Konfigurasi S3'}
          </button>
        </div>
      </div>
    </div>

  {:else if activeTab === 'reports'}
    <div id="panel-reports" role="tabpanel" aria-labelledby="tab-reports" style="display:grid; grid-template-columns: 1fr 1fr; gap:16px">
      <!-- Report Export Card -->
      <div class="panel" style="padding:18px">
        <div style="font-weight:600; font-size:14.5px; margin-bottom:6px">Export Laporan Infrastruktur & Uptime (FR-31)</div>
        <div style="font-size:12.5px; color:var(--muted); margin-bottom:16px">
          Unduh laporan lengkap yang mencakup status seluruh server, penggunaan CPU/RAM/Disk terakhir, status endpoint Uptime & masa berlaku SSL, serta riwayat insiden alert.
        </div>

        <div style="display:flex; gap:10px; flex-wrap:wrap">
          <button
            class="btn primary"
            on:click={() => handleExportReport('html')}
            disabled={isExporting['html']}
          >
            <svg viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
            {isExporting['html'] ? 'Mengekspor...' : 'Export Laporan Eksekutif (HTML / Cetak PDF)'}
          </button>

          <button
            class="btn"
            on:click={() => handleExportReport('csv')}
            disabled={isExporting['csv']}
          >
            <svg viewBox="0 0 24 24"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
            {isExporting['csv'] ? 'Mengekspor...' : 'Export Data Tabel (.CSV)'}
          </button>
        </div>
      </div>

      <!-- Data Retention Card -->
      <div class="panel" style="padding:18px">
        <div style="font-weight:600; font-size:14.5px; margin-bottom:6px">Retensi Data Historis SQLite (FR-29)</div>
        <div style="font-size:12.5px; color:var(--muted); margin-bottom:14px">
          Atur berapa lama riwayat metrik time-series disimpan di database lokal sebelum dibersihkan otomatis.
        </div>

        <div class="form-group" style="margin-bottom:14px">
          <label for="ret-days">Batas Retensi Riwayat Metrik (Hari)</label>
          <input id="ret-days" type="number" min="1" max="365" bind:value={settings.retentionDays} />
          <div class="note">Default: 30 hari. Pembersihan otomatis berjalan setiap kali aplikasi dimulai.</div>
        </div>

        <button class="btn" on:click={handleCleanOldMetrics} disabled={isCleaning}>
          {isCleaning ? 'Membersihkan...' : 'Bersihkan Data Lama Sekarang'}
        </button>
      </div>
    </div>

  {:else if activeTab === 'desktop'}
    <div id="panel-desktop" role="tabpanel" aria-labelledby="tab-desktop" class="panel" style="padding:18px">
      <div style="font-weight:600; font-size:14.5px; margin-bottom:14px">Perilaku Aplikasi Desktop Windows (FR-32 & FR-33)</div>

      <div style="display:grid; gap:14px">
        <div style="display:flex; align-items:center; justify-content:space-between; padding-bottom:12px; border-bottom:1px solid var(--line)">
          <div>
            <div style="font-weight:500">Konfirmasi Sebelum Eksekusi Perintah Terminal SSH (SEC-06)</div>
            <div style="font-size:12px; color:var(--dim)">
              Tampilkan kotak dialog konfirmasi pop-up dari sistem operasi Windows sebelum perintah terminal SSH dijalankan. Nonaktifkan opsi ini jika Anda ingin menjalankan perintah di tab Terminal secara instan tanpa konfirmasi pop-up.
            </div>
          </div>
          <label class="switch">
            <input type="checkbox" aria-label="Konfirmasi Sebelum Eksekusi Perintah Terminal SSH" bind:checked={settings.confirmSshCommands} />
            <span class="slider"></span>
          </label>
        </div>

        <div style="display:flex; align-items:center; justify-content:space-between; padding-bottom:12px; border-bottom:1px solid var(--line)">
          <div>
            <div style="font-weight:500">Tetap Memonitor di Latar Belakang Saat Jendela Ditutup (FR-32)</div>
            <div style="font-size:12px; color:var(--dim)">
              Saat tombol tutup (X) ditekan, tawarkan opsi untuk meminimalkan aplikasi ke latar belakang agar polling SSH dan jadwal backup tetap berjalan.
            </div>
          </div>
          <label class="switch">
            <input type="checkbox" aria-label="Tetap Memonitor di Latar Belakang Saat Jendela Ditutup" bind:checked={settings.runInBackground} />
            <span class="slider"></span>
          </label>
        </div>

        <div style="display:flex; align-items:center; justify-content:space-between">
          <div>
            <div style="font-weight:500">Jalankan ServerWatch Otomatis Saat Windows Menyala (Auto-Start) (FR-33)</div>
            <div style="font-size:12px; color:var(--dim)">
              Mendaftarkan executable ServerWatch ke startup profil pengguna Windows (HKCU Run).
            </div>
          </div>
          <label class="switch">
            <input type="checkbox" aria-label="Jalankan ServerWatch Otomatis Saat Windows Menyala" bind:checked={settings.autoStartOs} />
            <span class="slider"></span>
          </label>
        </div>
      </div>

      <div style="display:flex; justify-content:flex-end; margin-top:16px; padding-top:12px; border-top:1px solid var(--line)">
        <button class="btn primary" on:click={handleSaveSettings} disabled={isSaving}>
          {isSaving ? 'Menyimpan...' : 'Simpan Pengaturan Desktop'}
        </button>
      </div>
    </div>
  {/if}
</div>
