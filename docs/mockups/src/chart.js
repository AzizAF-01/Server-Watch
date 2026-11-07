// Data contoh deterministik untuk mockup (bukan data server nyata).
function rng(seed) {
  let s = seed >>> 0;
  return () => ((s = (s * 1664525 + 1013904223) >>> 0) / 4294967296);
}
function series(seed, n, base, amp, min = 0, max = 100) {
  const r = rng(seed); const out = []; let v = base;
  for (let i = 0; i < n; i++) {
    v += (r() - 0.5) * amp;
    v += (base - v) * 0.08;
    out.push(Math.max(min, Math.min(max, v)));
  }
  return out;
}
function path(data, w, h, max) {
  const step = w / (data.length - 1);
  return data.map((v, i) => `${i ? "L" : "M"}${(i * step).toFixed(1)},${(h - (v / max) * h).toFixed(1)}`).join("");
}
// <svg class="spark" data-seed data-base data-amp data-color>
document.querySelectorAll("svg[data-spark]").forEach(el => {
  const d = series(+el.dataset.seed, 40, +el.dataset.base, +el.dataset.amp);
  el.setAttribute("viewBox", "0 0 90 22");
  el.innerHTML = `<path d="${path(d, 90, 22, 100)}" fill="none" stroke="${el.dataset.color || "#8b939e"}" stroke-width="1.4"/>`;
});
// <svg data-chart data-lines='[{"seed":1,"base":40,"amp":10,"color":"#..","fill":true}]' data-max="100" data-thr="90">
document.querySelectorAll("svg[data-chart]").forEach(el => {
  const W = 600, H = 150, max = +el.dataset.max || 100;
  const lines = JSON.parse(el.dataset.lines);
  el.setAttribute("viewBox", `0 0 ${W} ${H}`);
  el.setAttribute("preserveAspectRatio", "none");
  let g = "";
  for (let i = 1; i < 4; i++) g += `<line x1="0" x2="${W}" y1="${(H / 4) * i}" y2="${(H / 4) * i}" stroke="#2a2f35" stroke-width="1"/>`;
  if (el.dataset.thr) {
    const y = H - (+el.dataset.thr / max) * H;
    g += `<line x1="0" x2="${W}" y1="${y}" y2="${y}" stroke="#e5534b" stroke-width="1" stroke-dasharray="4 4" opacity=".7"/>`;
  }
  for (const l of lines) {
    const d = series(l.seed, 120, l.base, l.amp, 0, max);
    if (l.spike) for (let i = l.spike[0]; i < l.spike[1]; i++) d[i] = Math.min(max, d[i] + l.spike[2]);
    const p = path(d, W, H, max);
    if (l.fill) g += `<path d="${p}L${W},${H}L0,${H}Z" fill="${l.color}" opacity=".12"/>`;
    g += `<path d="${p}" fill="none" stroke="${l.color}" stroke-width="1.6" vector-effect="non-scaling-stroke"/>`;
  }
  el.innerHTML = g;
});
