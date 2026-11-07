<script lang="ts">
  import { onMount } from 'svelte';
  import {
    GetUptimeMonitors,
    SaveUptimeMonitor,
    DeleteUptimeMonitor,
    CheckUptimeMonitorNow,
    OpenExternalURL,
    ExportReport
  } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import type { model } from '../wailsjs/go/models';

  export let servers: model.Server[] = [];

  let monitors: model.UptimeMonitor[] = [];
  let isLoading = false;
  let checkingMap: Record<string, boolean> = {};
  let filterStatus: 'all' | 'up' | 'warning' | 'down' = 'all';
  let searchQuery = '';
  let feedbackMsg: { type: 'ok' | 'err'; text: string } | null = null;

  // Add/Edit Modal State
  let showModal = false;
  let isEditing = false;
  let isSaving = false;
  let formError = '';
  let formData: model.UptimeMonitor = {
    id: '',
    name: '',
    type: 'http',
    target: 'https://',
    serverId: '',
    intervalSec: 60,
    enabled: true,
    status: 'up',
    statusCode: 0,
    latencyMs: 0,
    sslExpiryDays: 0,
    sslIssuer: '',
    sslValidUntil: '',
    lastError: '',
    createdAt: new Date().toISOString()
  } as any;

  onMount(async () => {
    await loadMonitors();
    try {
      EventsOn('uptime:update', (updated: model.UptimeMonitor) => {
        if (!updated || !updated.id) return;
        const idx = monitors.findIndex(m => m.id === updated.id);
        if (idx !== -1) {
          monitors[idx] = updated;
          monitors = [...monitors];
        } else {
          monitors = [...monitors, updated];
        }
      });
    } catch (_) {}
  });

  export async function loadMonitors() {
    isLoading = true;
    try {
      monitors = (await GetUptimeMonitors()) || [];
    } catch (err: any) {
      showFeedback('err', 'Gagal memuat daftar monitor uptime: ' + (err?.message || String(err)));
    } finally {
      isLoading = false;
    }
  }

  function showFeedback(type: 'ok' | 'err', text: string) {
    feedbackMsg = { type, text };
    setTimeout(() => {
      if (feedbackMsg && feedbackMsg.text === text) {
        feedbackMsg = null;
      }
    }, 5000);
  }

  function openAddModal() {
    isEditing = false;
    formError = '';
    formData = {
      id: '',
      name: '',
      type: 'http',
      target: 'https://',
      serverId: '',
      intervalSec: 60,
      enabled: true,
      status: 'up',
      statusCode: 0,
      latencyMs: 0,
      sslExpiryDays: 0,
      sslIssuer: '',
      sslValidUntil: '',
      lastError: '',
      createdAt: new Date().toISOString()
    } as any;
    showModal = true;
  }

  function openEditModal(mon: model.UptimeMonitor) {
    isEditing = true;
    formError = '';
    formData = { ...mon } as any;
    showModal = true;
  }

  async function handleSave() {
    formError = '';
    if (!formData.name.trim()) {
      formError = 'Nama monitor wajib diisi.';
      return;
    }
    if (!formData.target.trim()) {
      formError = 'Target URL atau host:port wajib diisi.';
      return;
    }
    isSaving = true;
    try {
      formData.intervalSec = Number(formData.intervalSec) || 60;
      await SaveUptimeMonitor(formData);
      showModal = false;
      showFeedback('ok', `Monitor '${formData.name}' berhasil disimpan dan sedang diperiksa.`);
      await loadMonitors();
    } catch (err: any) {
      formError = err?.message || String(err);
    } finally {
      isSaving = false;
    }
  }

  async function handleCheckNow(mon: model.UptimeMonitor) {
    checkingMap[mon.id] = true;
    try {
      const updated = await CheckUptimeMonitorNow(mon.id);
      if (updated) {
        const idx = monitors.findIndex(m => m.id === updated.id);
        if (idx !== -1) {
          monitors[idx] = updated;
          monitors = [...monitors];
        }
      }
    } catch (err: any) {
      showFeedback('err', 'Gagal memeriksa endpoint: ' + (err?.message || String(err)));
    } finally {
      checkingMap[mon.id] = false;
    }
  }

  async function handleDelete(mon: model.UptimeMonitor) {
    if (!confirm(`Hapus monitor uptime '${mon.name}'?`)) return;
    try {
      await DeleteUptimeMonitor(mon.id);
      await loadMonitors();
    } catch (err: any) {
      showFeedback('err', 'Gagal menghapus monitor: ' + (err?.message || String(err)));
    }
  }

  async function handleOpenBrowser(urlStr: string) {
    try {
      await OpenExternalURL(urlStr);
    } catch (err: any) {
      showFeedback('err', err?.message || String(err));
    }
  }

  async function handleExport(fmt: 'csv' | 'html') {
    try {
      const savedPath = await ExportReport(fmt);
      if (savedPath) {
        showFeedback('ok', `Laporan berhasil disimpan ke: ${savedPath}`);
      }
    } catch (err: any) {
      const msg = err?.message || String(err);
      if (!msg.includes('dibatalkan')) {
        showFeedback('err', 'Gagal mengekspor laporan: ' + msg);
      }
    }
  }

  function getServerName(srvId: string): string {
    if (!srvId) return 'Aplikasi Desktop (Lokal)';
    const found = servers.find(s => s.id === srvId);
    return found ? `SSH: ${found.name}` : 'SSH Server';
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && showModal) {
      showModal = false;
    }
  }

  $: filteredMonitors = monitors.filter(m => {
    if (filterStatus !== 'all' && m.status !== filterStatus) return false;
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      return m.name.toLowerCase().includes(q) || m.target.toLowerCase().includes(q);
    }
    return true;
  });

  $: upCount = monitors.filter(m => m.status === 'up').length;
  $: warnCount = monitors.filter(m => m.status === 'warning').length;
  $: downCount = monitors.filter(m => m.status === 'down').length;
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="topbar">
  <h1>Uptime & Sertifikat SSL</h1>
  <span class="sub">{monitors.length} endpoint dipantau</span>
  <span class="spacer"></span>

  <div class="search">
    <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="6.5"/><path d="m20 20-4.2-4.2"/></svg>
    <input type="text" placeholder="Cari nama atau URL endpoint..." bind:value={searchQuery} />
  </div>

  <button class="btn" on:click={() => handleExport('html')} title="Ekspor Laporan Eksekutif HTML/PDF (FR-31)">
    <svg viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
    Export Laporan
  </button>

  <button class="btn primary" on:click={openAddModal}>
    <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
    Tambah Endpoint
  </button>
</div>

<div class="content">
  {#if feedbackMsg}
    <div
      class="panel"
      style="padding:10px 14px; margin-bottom:14px; display:flex; align-items:center; justify-content:space-between; border-color:{feedbackMsg.type === 'ok' ? 'var(--ok)' : 'var(--crit)'}; color:{feedbackMsg.type === 'ok' ? 'var(--ok)' : 'var(--crit)'}"
    >
      <span>{feedbackMsg.text}</span>
      <button class="btn ghost" style="height:28px; padding:0 12px; font-size:12px" on:click={() => feedbackMsg = null} aria-label="Tutup notifikasi">Tutup</button>
    </div>
  {/if}

  <!-- Filter & Summary Strip -->
  <div style="display:flex; gap:8px; align-items:center; margin-bottom:14px; flex-wrap:wrap">
    <button class="chip {filterStatus === 'all' ? 'on' : ''}" on:click={() => filterStatus = 'all'}>
      Semua <span class="n">{monitors.length}</span>
    </button>
    <button class="chip {filterStatus === 'up' ? 'on' : ''}" on:click={() => filterStatus = 'up'}>
      <span class="st ok"></span> Online (UP) <span class="n">{upCount}</span>
    </button>
    <button class="chip {filterStatus === 'warning' ? 'on' : ''}" on:click={() => filterStatus = 'warning'}>
      <span class="st warn"></span> Warning / SSL Kedaluwarsa <span class="n">{warnCount}</span>
    </button>
    <button class="chip {filterStatus === 'down' ? 'on' : ''}" on:click={() => filterStatus = 'down'}>
      <span class="st crit"></span> Down <span class="n">{downCount}</span>
    </button>
    <span class="spacer"></span>
    <button class="btn" on:click={loadMonitors} disabled={isLoading}>
      {isLoading ? 'Memuat...' : 'Refresh Status'}
    </button>
  </div>

  <div class="panel">
    {#if isLoading && monitors.length === 0}
      <div style="padding:40px 20px; text-align:center; color:var(--muted)">
        Memuat daftar endpoint uptime dan status sertifikat SSL...
      </div>
    {:else if filteredMonitors.length === 0}
      <div style="padding:44px 20px; text-align:center; color:var(--muted)">
        <div style="font-size:15px; color:var(--text); font-weight:600">Belum ada endpoint website atau TCP port yang dipantau</div>
        <div style="font-size:12.5px; margin-top:6px; max-width:480px; margin-left:auto; margin-right:auto">
          Tambahkan URL HTTPS, API healthcheck, atau port TCP internal untuk memantau response time, kode HTTP, dan sisa hari masa berlaku sertifikat SSL secara berkala.
        </div>
        <div style="margin-top:16px">
          <button class="btn primary" on:click={openAddModal}>+ Tambah Endpoint Pertama</button>
        </div>
      </div>
    {:else}
      <table>
        <thead>
          <tr>
            <th style="width:110px">Status</th>
            <th>Nama & Target Endpoint</th>
            <th>Lokasi Cek (FR-15)</th>
            <th>Respons / Kode</th>
            <th>Latency</th>
            <th>Sertifikat SSL</th>
            <th class="r" style="width:190px">Aksi</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredMonitors as mon}
            <tr>
              <td>
                <span class="st {mon.status === 'up' ? 'ok' : mon.status === 'warning' ? 'warn' : 'crit'}">
                  {mon.status.toUpperCase()}
                </span>
              </td>
              <td>
                <div style="font-weight:600; color:var(--text); display:flex; align-items:center; gap:6px">
                  {mon.name}
                  <span class="tag" style="font-size:10.5px">{mon.type.toUpperCase()}</span>
                </div>
                <div class="num" style="font-size:11.5px; color:var(--dim); display:flex; align-items:center; gap:6px; margin-top:2px">
                  <span>{mon.target}</span>
                  {#if mon.type === 'http'}
                    <button
                      class="btn ghost"
                      style="height:28px; padding:0 8px; font-size:11px; color:var(--accent)"
                      on:click={() => handleOpenBrowser(mon.target)}
                      title="Buka {mon.target} di browser default OS (SEC-12)"
                      aria-label="Buka {mon.name} di browser default"
                    >
                      Buka Browser
                    </button>
                  {/if}
                </div>
                {#if mon.lastError}
                  <div style="font-size:11px; color:{mon.status === 'down' ? 'var(--crit)' : 'var(--warn)'}; margin-top:2px">
                    {mon.lastError}
                  </div>
                {/if}
              </td>
              <td>
                <span class="tag" style="color:{mon.serverId ? 'var(--accent)' : 'var(--muted)'}">
                  {getServerName(mon.serverId)}
                </span>
              </td>
              <td class="num">
                {#if mon.type === 'http'}
                  {mon.statusCode > 0 ? `HTTP ${mon.statusCode}` : '-'}
                {:else}
                  {mon.status === 'up' ? 'TCP Terbuka' : 'Tertutup'}
                {/if}
              </td>
              <td class="num">
                {mon.latencyMs > 0 ? `${mon.latencyMs} ms` : '-'}
              </td>
              <td>
                {#if mon.target.startsWith('https://') && mon.sslExpiryDays > 0}
                  <div class="num" style="font-weight:600; color:{mon.sslExpiryDays <= 14 ? 'var(--warn)' : 'var(--ok)'}">
                    {mon.sslExpiryDays} hari lagi
                  </div>
                  <div style="font-size:11px; color:var(--dim)">
                    s.d. {mon.sslValidUntil} {mon.sslIssuer ? `(${mon.sslIssuer})` : ''}
                  </div>
                {:else if mon.target.startsWith('https://') && mon.serverId}
                  <span style="font-size:11.5px; color:var(--dim)">Via SSH curl</span>
                {:else}
                  <span style="color:var(--dim)">-</span>
                {/if}
              </td>
              <td class="r">
                <button
                  class="btn ghost"
                  style="height:28px; padding:0 10px; font-size:12px"
                  disabled={checkingMap[mon.id]}
                  on:click={() => handleCheckNow(mon)}
                  title="Periksa respons {mon.name} ({mon.target}) sekarang"
                  aria-label="Cek status {mon.name}"
                >
                  {checkingMap[mon.id] ? 'Mengecek...' : 'Cek'}
                </button>
                <button
                  class="btn ghost"
                  style="height:28px; padding:0 10px; font-size:12px"
                  on:click={() => openEditModal(mon)}
                  title="Edit konfigurasi monitor {mon.name}"
                  aria-label="Edit monitor {mon.name}"
                >
                  Edit
                </button>
                <button
                  class="btn ghost danger"
                  style="height:28px; padding:0 10px; font-size:12px"
                  on:click={() => handleDelete(mon)}
                  title="Hapus monitor {mon.name}"
                  aria-label="Hapus monitor {mon.name}"
                >
                  ✕
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>

{#if showModal}
  <div
    class="modal-overlay"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    on:click|self={() => showModal = false}
    on:keydown={handleKeydown}
  >
    <div class="modal-box" style="width:560px">
      <div class="m-h">
        <h2>{isEditing ? 'Edit Endpoint Monitor' : 'Tambah Endpoint Uptime & SSL'}</h2>
        <span class="spacer"></span>
        <button class="btn ghost" on:click={() => showModal = false} aria-label="Tutup modal" title="Tutup">✕</button>
      </div>

      <div class="m-b">
        {#if formError}
          <div style="padding:8px 12px; border-radius:var(--radius-md); background:rgba(229,83,75,0.12); border:1px solid var(--crit); color:var(--crit); font-size:12px">
            {formError}
          </div>
        {/if}

        <div class="row3">
          <div class="form-group">
            <label for="up-name">Nama Layanan / Website</label>
            <input id="up-name" type="text" placeholder="Contoh: Portal Produksi" bind:value={formData.name} />
          </div>
          <div class="form-group">
            <label for="up-type">Tipe Pengecekan</label>
            <select
              id="up-type"
              bind:value={formData.type}
              on:change={() => {
                if (formData.type === 'tcp' && formData.target.startsWith('http')) {
                  formData.target = '127.0.0.1:8080';
                } else if (formData.type === 'http' && !formData.target.startsWith('http')) {
                  formData.target = 'https://';
                }
              }}
            >
              <option value="http">HTTP(S) Status & Sertifikat SSL</option>
              <option value="tcp">Koneksi Port TCP</option>
            </select>
          </div>
        </div>

        <div class="form-group">
          <label for="up-target">
            {formData.type === 'http' ? 'URL Target (http:// atau https://)' : 'Host & Port TCP (host:port)'}
          </label>
          <input
            id="up-target"
            type="text"
            placeholder={formData.type === 'http' ? 'https://domain-anda.com/health' : '127.0.0.1:6379'}
            bind:value={formData.target}
          />
          <div class="note">
            {formData.type === 'http'
              ? 'Untuk URL https://, masa berlaku sertifikat SSL akan diperiksa otomatis.'
              : 'Menguji apakah soket TCP menerima koneksi pada host dan port tersebut.'}
          </div>
        </div>

        <div class="row3">
          <div class="form-group">
            <label for="up-server">Lokasi Eksekusi Pengecekan (FR-15)</label>
            <select id="up-server" bind:value={formData.serverId}>
              <option value="">Langsung dari Aplikasi Desktop</option>
              {#each servers as s}
                <option value={s.id}>Via SSH dari Server: {s.name} ({s.host})</option>
              {/each}
            </select>
            <div class="note">Pilih server SSH jika ingin mengecek service internal (misal 127.0.0.1:8080).</div>
          </div>

          <div class="form-group">
            <label for="up-interval">Interval Cek (Detik)</label>
            <input id="up-interval" type="number" min="10" max="3600" bind:value={formData.intervalSec} />
            <div class="note">Default 60 detik (minimal 10 detik).</div>
          </div>
        </div>
      </div>

      <div class="m-f">
        <span class="spacer"></span>
        <button class="btn" on:click={() => showModal = false}>Batal</button>
        <button class="btn primary" on:click={handleSave} disabled={isSaving}>
          {isSaving ? 'Menyimpan...' : 'Simpan & Cek'}
        </button>
      </div>
    </div>
  </div>
{/if}
