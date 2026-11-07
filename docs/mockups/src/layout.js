// Title bar + sidebar dipakai bersama semua halaman mockup.
const ICONS = {
  server: '<rect x="3" y="4" width="18" height="7" rx="1.5"/><rect x="3" y="13" width="18" height="7" rx="1.5"/><path d="M7 7.5h.01M7 16.5h.01"/>',
  alert: '<path d="M12 4 2.5 20h19L12 4z"/><path d="M12 10v4M12 17h.01"/>',
  uptime: '<circle cx="12" cy="12" r="8.5"/><path d="M12 7v5l3 2"/>',
  term: '<rect x="3" y="4" width="18" height="16" rx="1.5"/><path d="m7 9 3 3-3 3M13 15h4"/>',
  log: '<path d="M6 3h9l4 4v14H6z"/><path d="M9 11h7M9 15h7"/>',
  gear: '<circle cx="12" cy="12" r="3"/><path d="M12 2.5v3M12 18.5v3M2.5 12h3M18.5 12h3M5.3 5.3l2.1 2.1M16.6 16.6l2.1 2.1M5.3 18.7l2.1-2.1M16.6 7.4l2.1-2.1"/>',
};
function icon(n) { return `<svg viewBox="0 0 24 24">${ICONS[n]}</svg>`; }

function renderShell(active) {
  const nav = [
    ["server", "Server", ""],
    ["alert", "Alert", '<span class="count num">3</span>'],
    ["uptime", "Uptime check", ""],
    ["term", "Terminal", ""],
    ["log", "Log", ""],
    ["gear", "Pengaturan", ""],
  ].map(([k, label, extra]) =>
    `<a class="${k === active ? "active" : ""}">${icon(k)}${label}${extra}</a>`).join("");

  document.getElementById("titlebar").outerHTML = `
  <div class="titlebar">
    <b>ServerWatch</b><span>0.1.0</span><span class="mock">Mockup, data contoh</span>
    <span class="spacer"></span>
    <span class="winbtn">&#x2013;</span><span class="winbtn">&#x25A1;</span><span class="winbtn">&#x2715;</span>
  </div>`;

  document.getElementById("side").outerHTML = `
  <aside class="side">
    <div class="brand"><span class="brand-mark">${icon("server")}</span>ServerWatch</div>
    <nav class="nav">${nav}</nav>
    <div class="side-sec">Tag</div>
    <nav class="nav tagnav">
      <a><i style="background:#c9705f"></i>production<span class="dim num" style="margin-left:auto">6</span></a>
      <a><i style="background:#8f9aa6"></i>staging<span class="dim num" style="margin-left:auto">3</span></a>
      <a><i style="background:#b39a5c"></i>klien-a<span class="dim num" style="margin-left:auto">3</span></a>
    </nav>
    <div class="side-foot">
      <div class="row"><span>Koneksi SSH aktif</span><span class="num">11 / 12</span></div>
      <div class="row"><span>Polling</span><span class="num">10 dtk</span></div>
    </div>
  </aside>`;
}
