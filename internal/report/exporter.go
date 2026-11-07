package report

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"serverwatch/internal/model"
	"serverwatch/internal/storage"
)

// Exporter builds CSV and printable HTML/PDF reports from SQLite data (FR-31).
type Exporter struct {
	storage *storage.SQLiteStorage
}

func NewExporter(store *storage.SQLiteStorage) *Exporter {
	return &Exporter{storage: store}
}

// GenerateCSV builds a comprehensive CSV report of servers, latest metrics, uptime monitors, and alert events.
func (e *Exporter) GenerateCSV() ([]byte, error) {
	if e.storage == nil {
		return nil, fmt.Errorf("storage belum siap")
	}

	servers, _ := e.storage.ListServers()
	monitors, _ := e.storage.ListUptimeMonitors()
	alerts, _ := e.storage.ListAlertEvents(100)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{"LAPORAN MONITORING INFRASTRUKTUR SERVERWATCH"})
	_ = w.Write([]string{"Waktu Cetak", time.Now().Format("2006-01-02 15:04:05")})
	_ = w.Write([]string{})

	_ = w.Write([]string{"1. DAFTAR SERVER & METRIK TERAKHIR"})
	_ = w.Write([]string{"Nama Server", "Host", "Port", "Status", "CPU (%)", "RAM (%)", "Disk Maks (%)", "Load 1m", "Uptime (Hari)", "OS"})
	for _, s := range servers {
		cpuStr, ramStr, diskStr, loadStr, upStr, osStr := "-", "-", "-", "-", "-", "-"
		if m, err := e.storage.GetLatestMetrics(s.ID); err == nil && m != nil {
			cpuStr = fmt.Sprintf("%.1f", m.CPU.TotalPercent)
			ramStr = fmt.Sprintf("%.1f", m.Memory.UsedPercent)
			maxDisk := 0.0
			for _, d := range m.Disks {
				if d.UsedPercent > maxDisk {
					maxDisk = d.UsedPercent
				}
			}
			diskStr = fmt.Sprintf("%.1f", maxDisk)
			loadStr = fmt.Sprintf("%.2f", m.CPU.Load1)
			upStr = strconv.FormatUint(m.UptimeSeconds/86400, 10)
			if m.OSInfo != "" {
				osStr = m.OSInfo
			}
		}
		_ = w.Write([]string{
			s.Name,
			s.Host,
			strconv.Itoa(s.Port),
			string(s.Status),
			cpuStr,
			ramStr,
			diskStr,
			loadStr,
			upStr,
			osStr,
		})
	}
	_ = w.Write([]string{})

	_ = w.Write([]string{"2. UPTIME & SERTIFIKAT SSL ENDPOINT"})
	_ = w.Write([]string{"Nama Monitor", "Tipe", "Target", "Status", "HTTP Code", "Latency (ms)", "Sisa SSL (Hari)", "Berlaku Sampai"})
	for _, mon := range monitors {
		sslDays := "-"
		if strings.HasPrefix(mon.Target, "https://") && mon.SSLExpiryDays > 0 {
			sslDays = strconv.Itoa(mon.SSLExpiryDays)
		}
		_ = w.Write([]string{
			mon.Name,
			strings.ToUpper(mon.Type),
			mon.Target,
			strings.ToUpper(mon.Status),
			strconv.Itoa(mon.StatusCode),
			strconv.FormatInt(mon.LatencyMs, 10),
			sslDays,
			mon.SSLValidUntil,
		})
	}
	_ = w.Write([]string{})

	_ = w.Write([]string{"3. RIWAYAT INSIDEN ALERT TERBARU"})
	_ = w.Write([]string{"Status", "Tingkat", "Server / Target", "Pesan Insiden", "Waktu Mulai", "Waktu Pulih"})
	for _, ev := range alerts {
		resolved := "-"
		if ev.ResolvedAt != nil {
			resolved = ev.ResolvedAt.Format("2006-01-02 15:04:05")
		}
		_ = w.Write([]string{
			ev.Status,
			ev.Level,
			ev.ServerName,
			ev.Message,
			ev.StartedAt.Format("2006-01-02 15:04:05"),
			resolved,
		})
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// GenerateHTML builds a standalone, print-ready HTML/PDF Executive Report with strict HTML escaping (SEC-01).
func (e *Exporter) GenerateHTML() ([]byte, error) {
	if e.storage == nil {
		return nil, fmt.Errorf("storage belum siap")
	}

	servers, _ := e.storage.ListServers()
	monitors, _ := e.storage.ListUptimeMonitors()
	alerts, _ := e.storage.ListAlertEvents(50)

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<title>Laporan Eksekutif ServerWatch</title>
<style>
  body { font-family: "Segoe UI", system-ui, sans-serif; color: #1a1d21; margin: 32px; line-height: 1.45; font-size: 13px; }
  .header { display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #2e333a; padding-bottom: 14px; margin-bottom: 22px; }
  h1 { margin: 0; font-size: 20px; }
  h2 { font-size: 15px; margin: 24px 0 10px; border-bottom: 1px solid #d0d5dd; padding-bottom: 5px; }
  .meta { color: #555; font-size: 12px; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 18px; }
  th, td { border: 1px solid #d0d5dd; padding: 8px 10px; text-align: left; }
  th { background: #f3f4f6; font-weight: 600; font-size: 12px; }
  .num { font-family: Consolas, monospace; }
  .ok { color: #1b7f4b; font-weight: 600; }
  .warn { color: #b26b00; font-weight: 600; }
  .crit { color: #c92a2a; font-weight: 600; }
  .print-btn { background: #0f766e; color: #fff; border: none; padding: 8px 14px; border-radius: 5px; cursor: pointer; font-weight: 600; }
  @media print { .no-print { display: none; } body { margin: 12px; } }
</style>
</head>
<body>
<div class="header">
  <div>
    <h1>Laporan Kesehatan & Uptime ServerWatch</h1>
    <div class="meta">Waktu Cetak: ` + html.EscapeString(time.Now().Format("02 Jan 2006 15:04:05 WIB")) + `</div>
  </div>
  <button class="print-btn no-print" onclick="window.print()">Cetak / Simpan PDF</button>
</div>
`)

	b.WriteString(`<h2>1. Status Armada Server (` + strconv.Itoa(len(servers)) + ` Server)</h2>
<table>
<thead><tr><th>Nama Server</th><th>Host</th><th>Status</th><th>CPU</th><th>RAM</th><th>Disk Maks</th><th>Load 1m</th><th>Uptime</th></tr></thead>
<tbody>`)

	if len(servers) == 0 {
		b.WriteString(`<tr><td colspan="8">Belum ada server yang terdaftar.</td></tr>`)
	}
	for _, s := range servers {
		cpuStr, ramStr, diskStr, loadStr, upStr := "-", "-", "-", "-", "-"
		if m, err := e.storage.GetLatestMetrics(s.ID); err == nil && m != nil {
			cpuStr = fmt.Sprintf("%.1f%%", m.CPU.TotalPercent)
			ramStr = fmt.Sprintf("%.1f%%", m.Memory.UsedPercent)
			maxDisk := 0.0
			for _, d := range m.Disks {
				if d.UsedPercent > maxDisk {
					maxDisk = d.UsedPercent
				}
			}
			diskStr = fmt.Sprintf("%.1f%%", maxDisk)
			loadStr = fmt.Sprintf("%.2f", m.CPU.Load1)
			upStr = fmt.Sprintf("%d hari", m.UptimeSeconds/86400)
		}
		stClass := "ok"
		if s.Status == model.StatusCritical || s.Status == model.StatusUnreachable {
			stClass = "crit"
		} else if s.Status == model.StatusWarning {
			stClass = "warn"
		}
		b.WriteString(fmt.Sprintf(`<tr><td><b>%s</b></td><td class="num">%s:%d</td><td class="%s">%s</td><td class="num">%s</td><td class="num">%s</td><td class="num">%s</td><td class="num">%s</td><td class="num">%s</td></tr>`,
			html.EscapeString(s.Name),
			html.EscapeString(s.Host),
			s.Port,
			stClass,
			html.EscapeString(strings.ToUpper(string(s.Status))),
			cpuStr, ramStr, diskStr, loadStr, upStr,
		))
	}
	b.WriteString(`</tbody></table>`)

	b.WriteString(`<h2>2. Monitoring Uptime & Sertifikat SSL (` + strconv.Itoa(len(monitors)) + ` Endpoint)</h2>
<table>
<thead><tr><th>Nama Endpoint</th><th>Tipe</th><th>Target</th><th>Status</th><th>HTTP Code</th><th>Latency</th><th>Masa Berlaku SSL</th></tr></thead>
<tbody>`)
	if len(monitors) == 0 {
		b.WriteString(`<tr><td colspan="7">Belum ada endpoint uptime yang dipantau.</td></tr>`)
	}
	for _, mon := range monitors {
		stClass := "ok"
		if mon.Status == "down" {
			stClass = "crit"
		} else if mon.Status == "warning" {
			stClass = "warn"
		}
		sslInfo := "-"
		if strings.HasPrefix(mon.Target, "https://") && mon.SSLExpiryDays > 0 {
			sslInfo = fmt.Sprintf("%d hari (%s)", mon.SSLExpiryDays, html.EscapeString(mon.SSLValidUntil))
		}
		b.WriteString(fmt.Sprintf(`<tr><td><b>%s</b></td><td class="num">%s</td><td class="num">%s</td><td class="%s">%s</td><td class="num">%d</td><td class="num">%d ms</td><td class="num">%s</td></tr>`,
			html.EscapeString(mon.Name),
			html.EscapeString(strings.ToUpper(mon.Type)),
			html.EscapeString(mon.Target),
			stClass,
			html.EscapeString(strings.ToUpper(mon.Status)),
			mon.StatusCode,
			mon.LatencyMs,
			sslInfo,
		))
	}
	b.WriteString(`</tbody></table>`)

	b.WriteString(`<h2>3. Riwayat Insiden Alert Terbaru (` + strconv.Itoa(len(alerts)) + ` Insiden)</h2>
<table>
<thead><tr><th>Status</th><th>Tingkat</th><th>Server / Endpoint</th><th>Deskripsi</th><th>Waktu Mulai</th><th>Waktu Pulih</th></tr></thead>
<tbody>`)
	if len(alerts) == 0 {
		b.WriteString(`<tr><td colspan="6">Tidak ada insiden yang tercatat.</td></tr>`)
	}
	for _, ev := range alerts {
		resolved := "-"
		if ev.ResolvedAt != nil {
			resolved = ev.ResolvedAt.Format("2006-01-02 15:04:05")
		}
		b.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td><b>%s</b></td><td>%s</td><td class="num">%s</td><td class="num">%s</td></tr>`,
			html.EscapeString(strings.ToUpper(ev.Status)),
			html.EscapeString(strings.ToUpper(ev.Level)),
			html.EscapeString(ev.ServerName),
			html.EscapeString(ev.Message),
			ev.StartedAt.Format("2006-01-02 15:04:05"),
			resolved,
		))
	}
	b.WriteString(`</tbody></table></body></html>`)

	return []byte(b.String()), nil
}
