import { mount } from 'svelte'
import './style.css'
import App from './App.svelte'

const target = document.getElementById('app')
let app: any = null

if (target) {
  try {
    app = mount(App, { target })
  } catch (err) {
    console.error("Failed to mount Svelte 5 App with mount():", err)
    // Fallback to legacy instantiation
    try {
      app = new (App as any)({ target })
    } catch (err2: any) {
      console.error("Failed legacy instantiation:", err2)
      target.innerHTML = `<div style="color:white;padding:24px;font-family:system-ui,sans-serif;background:#0d1117;border:1px solid #30363d;border-radius:8px;margin:20px;">
        <h2 style="color:#f85149;margin-top:0">Terjadi Kendala Memuat Aplikasi</h2>
        <p style="color:#8b949e">Mohon laporkan detail error berikut:</p>
        <pre style="background:#161b22;padding:12px;border-radius:6px;overflow:auto;color:#c9d1d9;font-size:12px;">${(err as any)?.stack || String(err)}</pre>
        <pre style="background:#161b22;padding:12px;border-radius:6px;overflow:auto;color:#8b949e;font-size:12px;">${err2?.stack || String(err2)}</pre>
      </div>`
    }
  }
} else {
  console.error("Element #app not found in DOM")
}

export default app
