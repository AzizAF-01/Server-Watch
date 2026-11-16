# ServerWatch: Design System & Visual Direction (`DESIGN.md`)

> **Design Read:** Desktop infrastructure telemetry & server operations console for system administrators and DevOps engineers, in a **Tactile Industrial / Hardware-Inspired** visual language, dial **ENERGY 1 / RHYTHM 2 / MOTION 1**.

---

## 1. Identity & Personality

ServerWatch is an agentless desktop instrument for inspecting, operating, and safeguarding Linux fleets over SSH. Its visual language draws inspiration from precision hardware instruments (Teenage Engineering field units, Panic Nova, rackmount telemetry modules):

- **Tactile & Physical**: Controls feel like physical switches and keycaps with crisp 1px borders and subtle top/bottom edge bevels rather than floating cards with blurry shadows.
- **Instrument-Grade Clarity**: Every screen prioritizes live operational readings, tabular monospace telemetry, and unmistakable hardware LED status indicators (`OK`, `WARN`, `CRIT`, `OFF`).
- **Calm Under Incident Pressure**: No decorative gradients, no glowing ambient orbs, and no perpetual pulsing loops. High-contrast signal colors appear only when actionable thresholds or active selections demand attention.

---

## 2. Liveliness Dials (`ENERGY 1 / RHYTHM 2 / MOTION 1`)

| Dial | Value | Implementation in ServerWatch |
|------|-------|-------------------------------|
| **ENERGY** | **1 (Calm)** | Precision 1px module frames, matte anodized or brushed aluminum surfaces, zero decorative gradients, calm hierarchy designed for long monitoring sessions. |
| **RHYTHM** | **2 (Consistent with breaks)** | Structured instrument panels that break uniformity according to content hierarchy: compact attention strip at the top, high-density fleet matrix, 2x2 telemetry rack modules, and full-bleed terminal viewports. |
| **MOTION** | **1 (Hover & micro-transitions)** | Instant 80ms–120ms tactile state changes on buttons, segmented switches, and row selection; no decorative scroll-reveals or endless pulsing animations. |

---

## 3. Dual Theme Palette (R-21, R-25, R-29, R-34)

ServerWatch ships with a first-class, persisted **Dual Theme** toggle accessible directly from the application Titlebar and Settings view:

### Theme A: Anodized Dark (`data-theme="dark"`, Default)
- **Chassis / Base (`--bg`)**: `#111316` (Anodized dark graphite chassis)
- **Module Surface (`--panel`)**: `#171a1f` (Recessed instrument bay)
- **Elevated / Keycap Surface (`--panel-2`)**: `#1e2229` (Tactile control surface)
- **Interactive Hover (`--surface-hover`)**: `#252a33`
- **Precision Seam (`--line`)**: `#2b313b` (1px machined divider)
- **Strong Seam (`--line-strong`)**: `#3a424f`
- **Primary Ink (`--text`)**: `#eceff4` (Contrast ratio > 14:1 on `--bg`)
- **Secondary Label (`--muted`)**: `#9ba3b0` (Contrast ratio > 6.2:1 on `--bg` & `--panel`)
- **Technical Metadata (`--dim`)**: `#8c95a4` (WCAG AA >= 4.5:1 across dark surfaces)
- **Primary Accent (`--accent`)**: `#f25f22` (Signal Orange)
- **Accent Ink (`--accent-ink`)**: `#ffffff` / `#111316` depending on fill weight
- **Hardware Status LEDs**:
  - **OK (`--ok`)**: `#2fd180` (Phosphor green)
  - **WARN (`--warn`)**: `#f59e0b` (Amber warning)
  - **CRIT (`--crit`)**: `#f04438` (Signal red)
  - **OFF (`--off`)**: `#848d9c` (Offline slate)

### Theme B: Brushed Aluminum Light (`data-theme="light"`)
- **Chassis / Base (`--bg`)**: `#e6e9ef` (Anodized silver/aluminum chassis)
- **Module Surface (`--panel`)**: `#f4f6f9` (Technical paper / instrument faceplate)
- **Elevated / Keycap Surface (`--panel-2`)**: `#ffffff` (Crisp tactile keycap surface)
- **Interactive Hover (`--surface-hover`)**: `#eaeef4`
- **Precision Seam (`--line`)**: `#cbd2dd` (1px machined seam)
- **Strong Seam (`--line-strong`)**: `#aeb8c7`
- **Primary Ink (`--text`)**: `#14181f` (Contrast ratio > 13:1 on `--panel`)
- **Secondary Label (`--muted`)**: `#465061` (Contrast ratio > 6.8:1 on `--panel`)
- **Technical Metadata (`--dim`)**: `#556070` (WCAG AA >= 4.8:1 on `--panel` & `--bg`)
- **Primary Accent (`--accent`)**: `#d84b16` (Industrial Signal Orange, WCAG AA compliant with white ink)
- **Hardware Status LEDs**:
  - **OK (`--ok`)**: `#0f7647` (High-contrast instrument green)
  - **WARN (`--warn`)**: `#b45309` (High-contrast technical amber)
  - **CRIT (`--crit`)**: `#c81e1e` (High-contrast alarm red)
  - **OFF (`--off`)**: `#525d6d` (Offline steel)

---

## 4. Typography & Identity Motifs (R-06, R-20)

1. **UI & Technical Sans (`--sans`)**: `"Inter", "Segoe UI Variable Display", "Segoe UI", system-ui, sans-serif`. Clean geometric legibility for dense operational labels and navigation.
2. **Telemetry & Instrument Mono (`--mono`)**: `"JetBrains Mono", "Cascadia Mono", "Consolas", monospace` with `font-variant-numeric: tabular-nums`. Used on all sensor readouts, IP addresses, PIDs, latency deltas, and module index codes so numbers never jitter during 5-second live polling.
3. **Identity Motifs**:
   - **Tactile Keycap Controls**: Buttons and segmented controls feature a crisp 1px border and a 1px bottom structural edge (`box-shadow: 0 1px 0 var(--bevel-shadow), inset 0 1px 0 var(--bevel-highlight)`) that depresses by 1px on `:active`.
   - **Module Registration Headers**: Panel headers sit in a distinct machined strip (`--panel-header`) with a subtle left-hand technical marker and right-aligned tabular telemetry metadata.
   - **Precision Scale Telemetry Bars**: CPU, RAM, and Disk bars include subtle 25%/50%/75% instrument scale ticks overlaid on the track so operators can judge capacity thresholds at a glance.
   - **Tight Machined Radii (R-11)**: `--radius-sm: 2px`, `--radius-md: 4px`, `--radius-lg: 6px`. Pill shapes (`999px`) are forbidden so every component retains a machined hardware silhouette.

---

## 5. Written Purpose Log (R-31: One-Line Reason per Decision)

- **Why Dual Theme (Anodized Dark & Brushed Aluminum Light)?** Operators use ServerWatch both in dim NOC/terminal setups and bright daytime engineering environments, and hardware instruments naturally pair dark graphite with anodized silver.
- **Why Signal Orange (`#f25f22` / `#d84b16`) as Primary Accent?** It separates interactive selection and primary operator actions from green/yellow/red health indicators so status colors are never confused with navigation state.
- **Why Tight 2px–6px Border Radii?** Tight radii reinforce the machined rackmount instrument identity and maximize usable data density in tables and terminal panes.
- **Why Tabular Monospace for All Numeric Readouts?** Live 1-to-5 second SSH polling updates CPU, RAM, network rates, and query durations continuously; tabular numerals prevent horizontal layout jitter.
- **Why Tactile 1px Beveled Buttons Instead of Soft Drop Shadows?** Physical keycap borders communicate clickability clearly in both dark and light themes without floating every surface off the chassis.
- **Why Precision Scale Ticks on Resource Bars?** Scale ticks let sysadmins read 50%, 75%, and 90% alert thresholds directly on compact progress bars without hovering.
