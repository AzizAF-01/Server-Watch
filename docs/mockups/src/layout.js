// ServerWatch: Shared Titlebar & Sidebar layout for mockups & portofolio captures
const ICONS = {
  server: '<rect x="3" y="4" width="18" height="7" rx="1.5"/><rect x="3" y="13" width="18" height="7" rx="1.5"/><circle cx="7" cy="7.5" r="1"/><circle cx="7" cy="16.5" r="1"/>',
  database: '<ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>',
  devops: '<rect x="2" y="7" width="20" height="13" rx="2"/><path d="M6 7V4h4v3M10 7V4h4v3"/>',
  uptime: '<circle cx="12" cy="12" r="9"/><path d="M2.5 12h19M12 2.5a14.5 14.5 0 0 1 0 19M12 2.5a14.5 14.5 0 0 0 0 19"/>',
  alert: '<path d="M12 4 2.5 20h19L12 4z"/><path d="M12 10v4M12 17h.01"/>',
  term: '<rect x="3" y="4" width="18" height="16" rx="1.5"/><path d="m7 9 3 3-3 3M13 15h4"/>',
  log: '<path d="M6 3h9l4 4v14H6z"/><path d="M9 11h7M9 15h7"/>',
  gear: '<circle cx="12" cy="12" r="3"/><path d="M12 2.5v3M12 18.5v3M2.5 12h3M18.5 12h3M5.3 5.3l2.1 2.1M16.6 16.6l2.1 2.1M5.3 18.7l2.1-2.1M16.6 7.4l2.1-2.1"/>',
};

function icon(n) {
  return `<svg viewBox="0 0 24 24">${ICONS[n] || ""}</svg>`;
}

function renderShell(active) {
  // Check theme
  const currentTheme = document.documentElement.getAttribute("data-theme") || "dark";
  const themeLabel = currentTheme === "light" ? "THEME: ALUMINUM LIGHT" : "THEME: ANODIZED DARK";

  const nav = [
    ["server", "Server (Fleet)", ""],
    ["database", "Database & Backup", '<span class="count num" style="background:var(--ok-soft); color:var(--ok); border-color:var(--ok-border)">3 DB</span>'],
    ["devops", "DevOps & Ekosistem", ""],
    ["uptime", "Uptime & SSL", ""],
    ["alert", "Alert", '<span class="count num">3</span>'],
    ["term", "Quick Terminal", ""],
    ["log", "Log Viewer", ""],
    ["gear", "Pengaturan", ""],
  ].map(([k, label, extra]) =>
    `<a class="${k === active ? "active" : ""}" href="${k === 'server' ? '01-dashboard.html' : k === 'devops' ? '06-devops.html' : k === 'uptime' ? '07-uptime-ssl.html' : k === 'gear' ? '08-settings.html' : k === 'term' ? '05-terminal-log.html' : k === 'alert' ? '04-alert.html' : '#'}">${icon(k)}<span>${label}</span>${extra}</a>`
  ).join("");

  const titlebarEl = document.getElementById("titlebar");
  if (titlebarEl) {
    titlebarEl.outerHTML = `
    <div class="titlebar">
      <span class="unit-code">SW // UNIT-01</span>
      <b>ServerWatch</b>
      <span class="num ver">v1.0.0</span>
      <span class="sep">|</span>
      <span class="sub">SSH AGENTLESS TELEMETRY CONSOLE</span>
      <span class="spacer"></span>
      <div class="theme-badge">
        <span class="led"></span>
        <span>${themeLabel}</span>
      </div>
      <span class="winbtn">–</span>
      <span class="winbtn">□</span>
      <span class="winbtn">✕</span>
    </div>`;
  }

  const sideEl = document.getElementById("side");
  if (sideEl) {
    sideEl.outerHTML = `
    <aside class="side">
      <div class="brand">
        <span class="brand-mark">${icon("server")}</span>
        <div>
          <div>SERVERWATCH</div>
          <div class="brand-sub">FLEET MONITOR</div>
        </div>
      </div>
      <div class="side-sec">Modules</div>
      <nav class="nav">${nav}</nav>
      <div class="side-sec">Fleet Tags</div>
      <nav class="nav tagnav">
        <a><i style="background:#e05252"></i>production<span class="dim num" style="margin-left:auto">6</span></a>
        <a><i style="background:#8f9aa6"></i>staging<span class="dim num" style="margin-left:auto">3</span></a>
        <a><i style="background:#c99e38"></i>klien-a<span class="dim num" style="margin-left:auto">3</span></a>
      </nav>
      <div class="side-foot">
        <div class="row"><span>Server Aktif</span><span class="num">11 / 12</span></div>
        <div class="row"><span>Interval Polling</span><span class="num">5 dtk</span></div>
        <div class="row"><span>Koneksi Engine</span><span class="num" style="color:var(--ok)">SSH Agentless</span></div>
      </div>
    </aside>`;
  }
}
