<script lang="ts">
  import { onMount } from 'svelte';
  import {
    GetDevOpsSnapshot,
    CancelDevOpsScan,
    ManageDockerContainer,
    GetDockerContainerLogs,
    ManageSystemService,
    TailLog
  } from '../wailsjs/go/main/App';
  import type { model } from '../wailsjs/go/models';

  export let servers: model.Server[] = [];
  export let selectedServerId: string = '';

  let activeTab: 'docker' | 'services' | 'ports' | 'security' = 'docker';
  let snapshot: model.DevOpsSnapshot | null = null;
  let snapshotCache: Record<string, model.DevOpsSnapshot> = {};
  let scanReqId = 0;
  let isLoading = false;
  let errorMessage = '';
  let actionInProgress: Record<string, boolean> = {};
  let feedbackBanner: { type: 'ok' | 'err'; text: string } | null = null;

  // Filters
  let containerFilter: 'all' | 'running' | 'stopped' = 'all';
  let containerSearch = '';

  let serviceFilter: 'all' | 'key' | 'running' | 'failed' = 'all';
  let serviceSearch = '';

  let portFilter: 'all' | 'public' | 'local' = 'all';
  let portSearch = '';

  // Log Inspector Modal
  let showLogModal = false;
  let logModalTitle = '';
  let logModalContent = '';
  let isLoadingModalLog = false;

  $: if (!selectedServerId && servers.length > 0) {
    selectedServerId = servers[0].id;
  }

  onMount(() => {
    if (selectedServerId) {
      loadSnapshot();
    }
  });

  function handleServerChange() {
    if (isLoading) {
      scanReqId++;
      isLoading = false;
      CancelDevOpsScan().catch(() => {});
    }
    errorMessage = '';
    snapshot = snapshotCache[selectedServerId] || null;
    loadSnapshot();
  }

  function handleCancelScan() {
    scanReqId++;
    isLoading = false;
    CancelDevOpsScan().catch(() => {});
  }

  export async function loadSnapshot() {
    if (!selectedServerId) return;
    const targetServerId = selectedServerId;
    const reqId = ++scanReqId;
    isLoading = true;
    errorMessage = '';
    try {
      const res = await GetDevOpsSnapshot(targetServerId);
      if (reqId !== scanReqId || selectedServerId !== targetServerId) {
        return;
      }
      snapshot = res;
      if (res) {
        snapshotCache[targetServerId] = res;
      }
    } catch (err: any) {
      if (reqId !== scanReqId || selectedServerId !== targetServerId) {
        return;
      }
      const msg = err?.message || String(err);
      if (!msg.toLowerCase().includes('canceled')) {
        errorMessage = msg;
      }
    } finally {
      if (reqId === scanReqId) {
        isLoading = false;
      }
    }
  }

  function showFeedback(type: 'ok' | 'err', text: string) {
    feedbackBanner = { type, text };
    setTimeout(() => {
      if (feedbackBanner && feedbackBanner.text === text) {
        feedbackBanner = null;
      }
    }, 5000);
  }

  async function handleContainerAction(containerName: string, action: 'start' | 'stop' | 'restart') {
    const key = `c_${containerName}_${action}`;
    actionInProgress[key] = true;
    feedbackBanner = null;
    try {
      await ManageDockerContainer(selectedServerId, containerName, action);
      showFeedback('ok', `Aksi '${action}' pada container '${containerName}' berhasil dijalankan.`);
      await loadSnapshot();
    } catch (err: any) {
      const msg = err?.message || String(err);
      if (!msg.includes('dibatalkan')) {
        showFeedback('err', msg);
      }
    } finally {
      actionInProgress[key] = false;
    }
  }

  async function handleViewContainerLogs(containerName: string) {
    logModalTitle = `Log Container: ${containerName}`;
    logModalContent = '';
    showLogModal = true;
    isLoadingModalLog = true;
    try {
      logModalContent = await GetDockerContainerLogs(selectedServerId, containerName);
    } catch (err: any) {
      logModalContent = 'Gagal memuat log container: ' + (err?.message || String(err));
    } finally {
      isLoadingModalLog = false;
    }
  }

  async function handleServiceAction(serviceName: string, action: 'start' | 'stop' | 'restart') {
    const key = `s_${serviceName}_${action}`;
    actionInProgress[key] = true;
    feedbackBanner = null;
    try {
      await ManageSystemService(selectedServerId, serviceName, action);
      showFeedback('ok', `Service '${serviceName}' berhasil di-${action}.`);
      await loadSnapshot();
    } catch (err: any) {
      const msg = err?.message || String(err);
      if (!msg.includes('dibatalkan')) {
        showFeedback('err', msg);
      }
    } finally {
      actionInProgress[key] = false;
    }
  }

  async function handleViewServiceLogs(serviceName: string) {
    logModalTitle = `Log Systemd: ${serviceName}.service`;
    logModalContent = '';
    showLogModal = true;
    isLoadingModalLog = true;
    try {
      logModalContent = await TailLog(selectedServerId, serviceName);
    } catch (err: any) {
      logModalContent = 'Gagal memuat log service: ' + (err?.message || String(err));
    } finally {
      isLoadingModalLog = false;
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && showLogModal) {
      showLogModal = false;
    }
  }

  $: filteredContainers = (snapshot?.containers || []).filter(c => {
    if (containerFilter === 'running' && c.state !== 'running') return false;
    if (containerFilter === 'stopped' && c.state === 'running') return false;
    if (containerSearch.trim()) {
      const q = containerSearch.toLowerCase();
      return c.name.toLowerCase().includes(q) || c.image.toLowerCase().includes(q) || c.id.toLowerCase().includes(q);
    }
    return true;
  });

  $: filteredServices = (snapshot?.services || []).filter(s => {
    if (serviceFilter === 'key' && !s.isKeyService) return false;
    if (serviceFilter === 'running' && s.subState !== 'running') return false;
    if (serviceFilter === 'failed' && s.activeState !== 'failed') return false;
    if (serviceSearch.trim()) {
      const q = serviceSearch.toLowerCase();
      return s.name.toLowerCase().includes(q) || s.description.toLowerCase().includes(q);
    }
    return true;
  });

  $: filteredPorts = (snapshot?.listeningPorts || []).filter(p => {
    if (portFilter === 'public' && p.exposure !== 'public') return false;
    if (portFilter === 'local' && p.exposure !== 'local') return false;
    if (portSearch.trim()) {
      const q = portSearch.toLowerCase();
      return String(p.port).includes(q) || p.process.toLowerCase().includes(q) || p.localAddress.toLowerCase().includes(q);
    }
    return true;
  });

  $: publicPortsCount = (snapshot?.listeningPorts || []).filter(p => p.exposure === 'public').length;
  $: failedServicesCount = (snapshot?.services || []).filter(s => s.activeState === 'failed').length;
  $: runningContainersCount = (snapshot?.containers || []).filter(c => c.state === 'running').length;
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="topbar">
  <h1>DevOps & Ekosistem Server</h1>
  <span class="sub">Container, Systemd Service, Port Jaringan, dan Audit Keamanan</span>
  <span class="spacer"></span>

  <div style="display:flex; align-items:center; gap:8px">
    <label for="devops-server-select" style="font-size:12px; color:var(--muted)">Target Server:</label>
    <select
      id="devops-server-select"
      style="height:32px; background:var(--bg); border:1px solid var(--line); color:var(--text); border-radius:5px; padding:0 10px; font-weight:500"
      bind:value={selectedServerId}
      on:change={handleServerChange}
    >
      {#if servers.length === 0}
        <option value="">Belum ada server terdaftar</option>
      {/if}
      {#each servers as s}
        <option value={s.id}>{s.name} ({s.user}@{s.host})</option>
      {/each}
    </select>
  </div>

  <button
    class="btn {isLoading ? '' : 'primary'}"
    style={isLoading ? 'border-color:var(--warn); color:var(--warn)' : ''}
    on:click={isLoading ? handleCancelScan : loadSnapshot}
    disabled={!selectedServerId}
    title={isLoading ? 'Klik untuk membatalkan pemindaian ekosistem' : 'Pindai ekosistem server target'}
  >
    <svg viewBox="0 0 24 24"><path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"/></svg>
    {isLoading ? 'Memindai... (Batalkan)' : 'Pindai Ekosistem'}
  </button>
</div>

<div class="content">
  {#if servers.length === 0}
    <div class="panel" style="padding:48px 20px; text-align:center; color:var(--muted)">
      <div style="font-size:15px; color:var(--text); font-weight:600">Belum ada server yang terdaftar</div>
      <div style="font-size:12.5px; margin-top:6px">
        Tambahkan server Linux terlebih dahulu di menu <b>Server</b> untuk memantau container Docker, service systemd, dan port jaringan.
      </div>
    </div>
  {:else}
    {#if feedbackBanner}
      <div
        class="panel"
        style="padding:10px 14px; margin-bottom:14px; display:flex; align-items:center; justify-content:space-between; border-color:{feedbackBanner.type === 'ok' ? 'var(--ok)' : 'var(--crit)'}; color:{feedbackBanner.type === 'ok' ? 'var(--ok)' : 'var(--crit)'}"
      >
        <span>{feedbackBanner.text}</span>
        <button class="btn ghost" style="height:24px; padding:0 8px" on:click={() => feedbackBanner = null}>Tutup</button>
      </div>
    {/if}

    {#if errorMessage}
      <div class="panel" style="padding:16px 18px; margin-bottom:16px; border-color:var(--crit); display:flex; align-items:center; justify-content:space-between; gap:12px">
        <div>
          <div style="font-weight:600; color:var(--crit)">Gagal memindai ekosistem server</div>
          <div style="font-size:12px; color:var(--muted); margin-top:3px">{errorMessage}</div>
        </div>
        <button class="btn" on:click={loadSnapshot}>Coba Lagi</button>
      </div>
    {/if}

    <!-- Sub-navigation Tabs -->
    <div class="subnav-tabs">
      <button class={activeTab === 'docker' ? 'active' : ''} on:click={() => activeTab = 'docker'}>
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="2" y="7" width="20" height="13" rx="2"/><path d="M6 7V4h4v3M10 7V4h4v3"/>
        </svg>
        Docker & Container ({snapshot ? snapshot.containers.length : 0})
      </button>

      <button class={activeTab === 'services' ? 'active' : ''} on:click={() => activeTab = 'services'}>
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M4.93 4.93l2.12 2.12M16.95 16.95l2.12 2.12M2 12h3M19 12h3M4.93 19.07l2.12-2.12M16.95 7.05l2.12-2.12"/>
        </svg>
        Systemd Service ({snapshot ? snapshot.services.length : 0})
        {#if failedServicesCount > 0}
          <span class="tag" style="background:var(--crit); color:#fff; font-weight:600; padding:1px 6px">{failedServicesCount} gagal</span>
        {/if}
      </button>

      <button class={activeTab === 'ports' ? 'active' : ''} on:click={() => activeTab = 'ports'}>
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
        </svg>
        Port & Firewall ({snapshot ? snapshot.listeningPorts.length : 0})
      </button>

      <button class={activeTab === 'security' ? 'active' : ''} on:click={() => activeTab = 'security'}>
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
        Audit Keamanan & Update OS
        {#if snapshot && snapshot.security.securityUpdates > 0}
          <span class="tag" style="background:var(--warn); color:#15171a; font-weight:600; padding:1px 6px">{snapshot.security.securityUpdates} patch</span>
        {/if}
      </button>
    </div>

    {#if isLoading && !snapshot}
      <div class="panel" style="padding:48px 20px; text-align:center; color:var(--muted)">
        <div style="font-size:14px; color:var(--text); font-weight:600">Memindai ekosistem server via SSH...</div>
        <div style="font-size:12px; margin-top:6px">
          Mengumpulkan status Docker container, unit systemd, soket jaringan aktif, dan log autentikasi SSH.
        </div>
      </div>

    {:else if snapshot}
      <!-- TAB 1: DOCKER & CONTAINERS -->
      {#if activeTab === 'docker'}
        <div class="panel" style="padding:12px 16px; margin-bottom:14px; display:flex; align-items:center; gap:14px; flex-wrap:wrap">
          <div>
            <span style="font-size:12px; color:var(--dim)">Engine Runtime:</span>
            <b style="margin-left:6px; color:var(--text)">{snapshot.dockerAvailable ? (snapshot.dockerVersion || 'Docker / Podman Aktif') : 'Tidak Terpasang'}</b>
          </div>
          <span style="color:var(--line)">|</span>
          <div>
            <span style="font-size:12px; color:var(--dim)">Container Berjalan:</span>
            <b class="num" style="margin-left:6px; color:var(--ok)">{runningContainersCount}</b>
            <span style="color:var(--dim)">/ {snapshot.containers.length} total</span>
          </div>
          <span class="spacer"></span>
          <div class="seg">
            <button class={containerFilter === 'all' ? 'on' : ''} on:click={() => containerFilter = 'all'}>Semua ({snapshot.containers.length})</button>
            <button class={containerFilter === 'running' ? 'on' : ''} on:click={() => containerFilter = 'running'}>Running ({runningContainersCount})</button>
            <button class={containerFilter === 'stopped' ? 'on' : ''} on:click={() => containerFilter = 'stopped'}>Stopped ({snapshot.containers.length - runningContainersCount})</button>
          </div>
          <div class="search" style="width:210px">
            <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="6.5"/><path d="m20 20-4.2-4.2"/></svg>
            <input type="text" placeholder="Cari container / image..." bind:value={containerSearch} />
          </div>
        </div>

        <div class="panel">
          {#if !snapshot.dockerAvailable}
            <div style="padding:40px 20px; text-align:center; color:var(--muted)">
              <div style="font-size:14px; color:var(--text); font-weight:600">Docker atau Podman tidak terdeteksi di server ini</div>
              <div style="font-size:12px; margin-top:6px; max-width:460px; margin-left:auto; margin-right:auto">
                Server target belum memiliki runtime container atau user SSH belum memiliki akses ke soket Docker.
              </div>
            </div>
          {:else if filteredContainers.length === 0}
            <div style="padding:36px 20px; text-align:center; color:var(--muted)">
              Tidak ada container yang cocok dengan filter saat ini.
            </div>
          {:else}
            <table>
              <thead>
                <tr>
                  <th style="width:105px">State</th>
                  <th>Container</th>
                  <th>Image</th>
                  <th>CPU</th>
                  <th>Memori</th>
                  <th>Port Mapping</th>
                  <th>Uptime / Status</th>
                  <th class="r" style="width:210px">Aksi</th>
                </tr>
              </thead>
              <tbody>
                {#each filteredContainers as c}
                  <tr>
                    <td>
                      <span class="st {c.state === 'running' ? 'ok' : c.state === 'restarting' ? 'warn' : 'off'}">
                        {c.state}
                      </span>
                    </td>
                    <td>
                      <div style="font-weight:600; color:var(--text)">{c.name}</div>
                      <div class="num" style="font-size:11px; color:var(--dim)">{c.id}</div>
                    </td>
                    <td class="num" style="font-size:12px; color:var(--muted)">{c.image}</td>
                    <td class="num">
                      {#if c.state === 'running'}
                        {c.cpuPercent.toFixed(1)}%
                      {:else}
                        <span style="color:var(--dim)">-</span>
                      {/if}
                    </td>
                    <td>
                      {#if c.state === 'running' && c.memUsage}
                        <div class="num" style="font-size:12px">{c.memUsage}</div>
                        <div class="num" style="font-size:11px; color:var(--dim)">{c.memPercent.toFixed(1)}%</div>
                      {:else}
                        <span style="color:var(--dim)">-</span>
                      {/if}
                    </td>
                    <td class="num" style="font-size:11.5px; color:var(--muted); max-width:190px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap" title={c.ports}>
                      {c.ports || '-'}
                    </td>
                    <td style="font-size:12px; color:var(--muted)">{c.status}</td>
                    <td class="r">
                      <button class="btn ghost" style="height:26px; padding:0 8px; font-size:11.5px" on:click={() => handleViewContainerLogs(c.name)}>
                        Log
                      </button>
                      {#if c.state === 'running'}
                        <button
                          class="btn ghost"
                          style="height:26px; padding:0 8px; font-size:11.5px"
                          disabled={actionInProgress[`c_${c.name}_restart`]}
                          on:click={() => handleContainerAction(c.name, 'restart')}
                        >
                          Restart
                        </button>
                        <button
                          class="btn ghost danger"
                          style="height:26px; padding:0 8px; font-size:11.5px"
                          disabled={actionInProgress[`c_${c.name}_stop`]}
                          on:click={() => handleContainerAction(c.name, 'stop')}
                        >
                          Stop
                        </button>
                      {:else}
                        <button
                          class="btn ghost"
                          style="height:26px; padding:0 8px; font-size:11.5px; color:var(--ok)"
                          disabled={actionInProgress[`c_${c.name}_start`]}
                          on:click={() => handleContainerAction(c.name, 'start')}
                        >
                          Start
                        </button>
                      {/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          {/if}
        </div>

      <!-- TAB 2: SYSTEMD SERVICES -->
      {:else if activeTab === 'services'}
        <div class="panel" style="padding:12px 16px; margin-bottom:14px; display:flex; align-items:center; gap:12px; flex-wrap:wrap">
          <div class="seg">
            <button class={serviceFilter === 'all' ? 'on' : ''} on:click={() => serviceFilter = 'all'}>Semua ({snapshot.services.length})</button>
            <button class={serviceFilter === 'key' ? 'on' : ''} on:click={() => serviceFilter = 'key'}>Infrastruktur Utama ({snapshot.services.filter(s => s.isKeyService).length})</button>
            <button class={serviceFilter === 'running' ? 'on' : ''} on:click={() => serviceFilter = 'running'}>Running ({snapshot.services.filter(s => s.subState === 'running').length})</button>
            <button class={serviceFilter === 'failed' ? 'on' : ''} on:click={() => serviceFilter = 'failed'}>Failed ({failedServicesCount})</button>
          </div>
          <span class="spacer"></span>
          <div class="search" style="width:230px">
            <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="6.5"/><path d="m20 20-4.2-4.2"/></svg>
            <input type="text" placeholder="Cari nama service atau deskripsi..." bind:value={serviceSearch} />
          </div>
        </div>

        <div class="panel">
          {#if filteredServices.length === 0}
            <div style="padding:36px 20px; text-align:center; color:var(--muted)">
              Tidak ada unit service systemd yang cocok dengan filter.
            </div>
          {:else}
            <table>
              <thead>
                <tr>
                  <th style="width:120px">Status</th>
                  <th>Unit Service</th>
                  <th>Sub-State</th>
                  <th>Deskripsi</th>
                  <th class="r" style="width:220px">Aksi (Konfirmasi OS)</th>
                </tr>
              </thead>
              <tbody>
                {#each filteredServices as svc}
                  <tr>
                    <td>
                      <span class="st {svc.activeState === 'active' ? 'ok' : svc.activeState === 'failed' ? 'crit' : 'off'}">
                        {svc.activeState}
                      </span>
                    </td>
                    <td>
                      <span class="num" style="font-weight:600; color:var(--text)">{svc.name}</span>
                      {#if svc.isKeyService}
                        <span class="tag" style="margin-left:6px; color:var(--accent)">Utama</span>
                      {/if}
                    </td>
                    <td class="num" style="font-size:12px; color:var(--muted)">{svc.subState}</td>
                    <td style="color:var(--muted); font-size:12.5px">{svc.description || '-'}</td>
                    <td class="r">
                      <button class="btn ghost" style="height:26px; padding:0 8px; font-size:11.5px" on:click={() => handleViewServiceLogs(svc.name)}>
                        Log
                      </button>
                      <button
                        class="btn ghost"
                        style="height:26px; padding:0 8px; font-size:11.5px"
                        disabled={actionInProgress[`s_${svc.name}_restart`]}
                        on:click={() => handleServiceAction(svc.name, 'restart')}
                      >
                        Restart
                      </button>
                      {#if svc.activeState === 'active'}
                        <button
                          class="btn ghost danger"
                          style="height:26px; padding:0 8px; font-size:11.5px"
                          disabled={actionInProgress[`s_${svc.name}_stop`]}
                          on:click={() => handleServiceAction(svc.name, 'stop')}
                        >
                          Stop
                        </button>
                      {:else}
                        <button
                          class="btn ghost"
                          style="height:26px; padding:0 8px; font-size:11.5px; color:var(--ok)"
                          disabled={actionInProgress[`s_${svc.name}_start`]}
                          on:click={() => handleServiceAction(svc.name, 'start')}
                        >
                          Start
                        </button>
                      {/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          {/if}
        </div>

      <!-- TAB 3: LISTENING PORTS & FIREWALL -->
      {:else if activeTab === 'ports'}
        <div class="panel" style="padding:12px 16px; margin-bottom:14px; display:flex; align-items:center; gap:16px; flex-wrap:wrap">
          <div style="display:flex; align-items:center; gap:8px">
            <span style="font-size:12px; color:var(--dim)">Firewall Host:</span>
            <span class="st {snapshot.firewall.active ? 'ok' : 'warn'}">
              {snapshot.firewall.summary}
            </span>
          </div>
          <span style="color:var(--line)">|</span>
          <div>
            <span style="font-size:12px; color:var(--dim)">Port Terbuka ke Publik (0.0.0.0 / ::):</span>
            <b class="num" style="margin-left:6px; color:{publicPortsCount > 0 ? 'var(--warn)' : 'var(--ok)'}">{publicPortsCount} port</b>
          </div>
          <span class="spacer"></span>
          <div class="seg">
            <button class={portFilter === 'all' ? 'on' : ''} on:click={() => portFilter = 'all'}>Semua ({snapshot.listeningPorts.length})</button>
            <button class={portFilter === 'public' ? 'on' : ''} on:click={() => portFilter = 'public'}>Publik ({publicPortsCount})</button>
            <button class={portFilter === 'local' ? 'on' : ''} on:click={() => portFilter = 'local'}>Lokal ({snapshot.listeningPorts.length - publicPortsCount})</button>
          </div>
          <div class="search" style="width:200px">
            <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="6.5"/><path d="m20 20-4.2-4.2"/></svg>
            <input type="text" placeholder="Cari port atau proses..." bind:value={portSearch} />
          </div>
        </div>

        <div class="panel">
          {#if filteredPorts.length === 0}
            <div style="padding:36px 20px; text-align:center; color:var(--muted)">
              Tidak ada port listening yang sesuai filter.
            </div>
          {:else}
            <table>
              <thead>
                <tr>
                  <th style="width:120px">Eksposur</th>
                  <th style="width:90px">Protokol</th>
                  <th style="width:100px">Port</th>
                  <th>Alamat Bind Lokal</th>
                  <th>Proses / Service</th>
                  <th class="r" style="width:100px">PID</th>
                </tr>
              </thead>
              <tbody>
                {#each filteredPorts as p}
                  <tr>
                    <td>
                      {#if p.exposure === 'public'}
                        <span class="tag" style="background:rgba(224,164,58,0.15); color:var(--warn); border:1px solid rgba(224,164,58,0.4)">PUBLIK</span>
                      {:else}
                        <span class="tag" style="color:var(--ok)">LOKAL</span>
                      {/if}
                    </td>
                    <td class="num" style="text-transform:uppercase; color:var(--muted)">{p.protocol}</td>
                    <td class="num" style="font-weight:600; font-size:13.5px; color:var(--text)">{p.port}</td>
                    <td class="num" style="color:var(--muted)">{p.localAddress}:{p.port}</td>
                    <td style="font-weight:500">{p.process}</td>
                    <td class="r num" style="color:var(--dim)">{p.pid > 0 ? p.pid : '-'}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          {/if}
        </div>

      <!-- TAB 4: SECURITY AUDIT & OS UPDATES -->
      {:else if activeTab === 'security'}
        <!-- Security Summary Strip -->
        <div class="panel" style="padding:14px 18px; margin-bottom:16px; display:flex; align-items:center; justify-content:space-between; flex-wrap:wrap; gap:16px">
          <div>
            <div style="font-size:11.5px; color:var(--dim)">Paket Update OS ({snapshot.security.packageManager})</div>
            <div style="font-size:15px; font-weight:600; margin-top:2px">
              <span class="num" style="color:{snapshot.security.pendingUpdates > 0 ? 'var(--warn)' : 'var(--ok)'}">{snapshot.security.pendingUpdates}</span> paket tersedia
              {#if snapshot.security.securityUpdates > 0}
                <span class="tag" style="margin-left:6px; background:rgba(229,83,75,0.15); color:var(--crit); border:1px solid var(--crit)">
                  {snapshot.security.securityUpdates} patch keamanan
                </span>
              {/if}
            </div>
          </div>

          <div>
            <div style="font-size:11.5px; color:var(--dim)">Percobaan Login SSH Gagal (Log Terbaru)</div>
            <div class="num" style="font-size:15px; font-weight:600; margin-top:2px; color:{snapshot.security.failedSshCount > 20 ? 'var(--crit)' : snapshot.security.failedSshCount > 0 ? 'var(--warn)' : 'var(--ok)'}">
              {snapshot.security.failedSshCount} percobaan gagal
            </div>
          </div>

          <div>
            <div style="font-size:11.5px; color:var(--dim)">PermitRootLogin (sshd)</div>
            <div class="num" style="font-size:13.5px; font-weight:600; margin-top:2px; color:{snapshot.security.permitRootLogin === 'no' ? 'var(--ok)' : 'var(--warn)'}">
              {snapshot.security.permitRootLogin}
            </div>
          </div>

          <div>
            <div style="font-size:11.5px; color:var(--dim)">PasswordAuth (sshd)</div>
            <div class="num" style="font-size:13.5px; font-weight:600; margin-top:2px; color:{snapshot.security.passwordAuth === 'no' ? 'var(--ok)' : 'var(--warn)'}">
              {snapshot.security.passwordAuth}
            </div>
          </div>

          <div>
            <div style="font-size:11.5px; color:var(--dim)">Proteksi Fail2ban</div>
            <div style="margin-top:2px">
              <span class="st {snapshot.security.fail2banActive ? 'ok' : 'off'}">
                {snapshot.security.fail2banActive ? 'Aktif' : 'Tidak Aktif'}
              </span>
            </div>
          </div>
        </div>

        <div style="display:grid; grid-template-columns: 1fr 1fr; gap:16px">
          <!-- Brute-Force Top Attackers -->
          <div class="panel">
            <div class="panel-h">
              <h2>Sumber IP Percobaan Brute-Force SSH Teratas</h2>
            </div>
            {#if snapshot.security.topAttackers.length === 0}
              <div style="padding:28px 18px; text-align:center; color:var(--muted)">
                Tidak ditemukan percobaan login gagal dari IP eksternal pada sampel log terbaru.
              </div>
            {:else}
              <table>
                <thead>
                  <tr>
                    <th>Alamat IP Sumber</th>
                    <th class="r">Jumlah Gagal</th>
                  </tr>
                </thead>
                <tbody>
                  {#each snapshot.security.topAttackers as atk}
                    <tr>
                      <td class="num" style="font-weight:600; color:var(--text)">{atk.ip}</td>
                      <td class="r num" style="color:{atk.attempts >= 10 ? 'var(--crit)' : 'var(--warn)'}; font-weight:600">
                        {atk.attempts}x
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {/if}
          </div>

          <!-- Active Login Sessions -->
          <div class="panel">
            <div class="panel-h">
              <h2>Sesi User Login Aktif (who)</h2>
            </div>
            {#if snapshot.security.activeSessions.length === 0}
              <div style="padding:28px 18px; text-align:center; color:var(--muted)">
                Tidak ada sesi terminal interaktif yang sedang login saat ini.
              </div>
            {:else}
              <table>
                <thead>
                  <tr>
                    <th>User</th>
                    <th>TTY</th>
                    <th>IP Asal</th>
                    <th class="r">Waktu Login</th>
                  </tr>
                </thead>
                <tbody>
                  {#each snapshot.security.activeSessions as sess}
                    <tr>
                      <td style="font-weight:600">{sess.user}</td>
                      <td class="num" style="color:var(--muted)">{sess.terminal}</td>
                      <td class="num" style="color:var(--accent)">{sess.fromIp}</td>
                      <td class="r num" style="color:var(--dim); font-size:12px">{sess.loginTime}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {/if}
          </div>
        </div>
      {/if}
    {/if}
  {/if}
</div>

{#if showLogModal}
  <div
    class="modal-overlay"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    on:click|self={() => showLogModal = false}
    on:keydown={handleKeydown}
  >
    <div class="modal-box" style="width:760px; max-width:92vw">
      <div class="m-h">
        <h2>{logModalTitle}</h2>
        <span class="spacer"></span>
        <button class="btn ghost" style="padding:2px 8px" on:click={() => showLogModal = false}>✕</button>
      </div>
      <div class="m-b">
        {#if isLoadingModalLog}
          <div style="padding:30px; text-align:center; color:var(--muted)">Mengambil output log dari server via SSH...</div>
        {:else}
          <pre style="background:#0f1114; color:var(--text); border:1px solid var(--line); border-radius:5px; padding:12px; font-family:var(--mono); font-size:11.5px; max-height:420px; overflow:auto; white-space:pre-wrap; word-break:break-word">{logModalContent}</pre>
        {/if}
      </div>
      <div class="m-f">
        <span class="spacer"></span>
        <button class="btn" on:click={() => showLogModal = false}>Tutup (Esc)</button>
      </div>
    </div>
  </div>
{/if}
