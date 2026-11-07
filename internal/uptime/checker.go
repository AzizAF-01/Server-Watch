package uptime

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"serverwatch/internal/model"
	"serverwatch/internal/sshpool"
	"serverwatch/internal/storage"
)

var safeHostPortRegex = regexp.MustCompile(`^[a-zA-Z0-9.-]+:\d{1,5}$`)

// Checker executes HTTP(S), SSL expiry, and TCP port checks locally or via remote SSH (FR-14 & FR-15).
type Checker struct {
	pool              *sshpool.SSHPool
	storage           *storage.SQLiteStorage
	onAlertTransition func(monitorName, level, message string, isRecovery bool)
}

func NewChecker(pool *sshpool.SSHPool, store *storage.SQLiteStorage) *Checker {
	return &Checker{
		pool:    pool,
		storage: store,
	}
}

func (c *Checker) SetAlertCallback(fn func(monitorName, level, message string, isRecovery bool)) {
	c.onAlertTransition = fn
}

// ValidateMonitorTarget validates URL or host:port targets and prevents shell metacharacters (SEC-05).
func ValidateMonitorTarget(monType, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("target monitor tidak boleh kosong")
	}
	if strings.ContainsAny(target, "'\"`;$&|<>\\\n\r\t ") {
		return fmt.Errorf("target mengandung karakter berbahaya yang tidak diizinkan (SEC-05)")
	}

	switch monType {
	case "http":
		u, err := url.Parse(target)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("target HTTP harus berupa URL valid dengan skema http:// atau https://")
		}
		return nil
	case "tcp":
		if !safeHostPortRegex.MatchString(target) {
			return fmt.Errorf("target TCP harus berformat host:port (contoh: 127.0.0.1:6379)")
		}
		_, portStr, err := net.SplitHostPort(target)
		if err != nil {
			return fmt.Errorf("format host:port tidak valid: %w", err)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			return fmt.Errorf("nomor port TCP harus antara 1 sampai 65535")
		}
		return nil
	default:
		return fmt.Errorf("tipe monitor '%s' tidak didukung (gunakan http atau tcp)", monType)
	}
}

// CheckMonitor runs a single uptime/SSL check, updates SQLite, and fires alert transitions if needed.
func (c *Checker) CheckMonitor(ctx context.Context, mon *model.UptimeMonitor) (*model.UptimeMonitor, error) {
	if err := ValidateMonitorTarget(mon.Type, mon.Target); err != nil {
		return nil, err
	}

	now := time.Now()
	mon.LastCheck = &now
	mon.LastError = ""

	if mon.ServerID != "" {
		c.checkViaSSH(ctx, mon)
	} else {
		if mon.Type == "tcp" {
			c.checkLocalTCP(ctx, mon)
		} else {
			c.checkLocalHTTP(ctx, mon)
		}
	}

	if c.storage != nil && mon.ID != "" {
		_ = c.storage.SaveUptimeMonitor(mon)
		ruleID := "uptime_" + mon.ID
		switch mon.Status {
		case "down":
			msg := fmt.Sprintf("Endpoint %s tidak dapat diakses (%s)", mon.Target, mon.LastError)
			if triggered, _ := c.storage.RecordAlertEventTransition(mon.ID, mon.Name, ruleID, "critical", msg); triggered && c.onAlertTransition != nil {
				c.onAlertTransition(mon.Name, "critical", msg, false)
			}
		case "warning":
			msg := fmt.Sprintf("Endpoint %s peringatan: %s", mon.Target, mon.LastError)
			if triggered, _ := c.storage.RecordAlertEventTransition(mon.ID, mon.Name, ruleID, "warning", msg); triggered && c.onAlertTransition != nil {
				c.onAlertTransition(mon.Name, "warning", msg, false)
			}
		case "up":
			if resolved, _ := c.storage.ResolveAlertEventTransition(mon.ID, ruleID); resolved && c.onAlertTransition != nil {
				c.onAlertTransition(mon.Name, "ok", fmt.Sprintf("Endpoint %s kembali normal (%d ms)", mon.Target, mon.LatencyMs), true)
			}
		}
	}

	return mon, nil
}

func (c *Checker) checkLocalHTTP(ctx context.Context, mon *model.UptimeMonitor) {
	start := time.Now()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, mon.Target, nil)
	if err != nil {
		mon.Status = "down"
		mon.StatusCode = 0
		mon.LastError = err.Error()
		return
	}
	req.Header.Set("User-Agent", "ServerWatch-UptimeMonitor/1.0")

	resp, err := client.Do(req)
	mon.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		mon.Status = "down"
		mon.StatusCode = 0
		mon.LastError = err.Error()
		return
	}
	defer resp.Body.Close()

	mon.StatusCode = resp.StatusCode

	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		days := int(math.Floor(time.Until(cert.NotAfter).Hours() / 24.0))
		mon.SSLExpiryDays = days
		mon.SSLValidUntil = cert.NotAfter.Format("2006-01-02")
		issuer := cert.Issuer.CommonName
		if issuer == "" && len(cert.Issuer.Organization) > 0 {
			issuer = cert.Issuer.Organization[0]
		}
		mon.SSLIssuer = issuer
	}

	if resp.StatusCode >= 500 {
		mon.Status = "down"
		mon.LastError = fmt.Sprintf("HTTP %d", resp.StatusCode)
	} else if resp.StatusCode >= 400 {
		mon.Status = "warning"
		mon.LastError = fmt.Sprintf("HTTP %d", resp.StatusCode)
	} else if strings.HasPrefix(mon.Target, "https://") && mon.SSLExpiryDays > 0 && mon.SSLExpiryDays <= 14 {
		mon.Status = "warning"
		mon.LastError = fmt.Sprintf("Sertifikat SSL kedaluwarsa dalam %d hari", mon.SSLExpiryDays)
	} else {
		mon.Status = "up"
	}
}

func (c *Checker) checkLocalTCP(ctx context.Context, mon *model.UptimeMonitor) {
	start := time.Now()
	d := net.Dialer{Timeout: 8 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", mon.Target)
	mon.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		mon.Status = "down"
		mon.StatusCode = 0
		mon.LastError = err.Error()
		return
	}
	_ = conn.Close()
	mon.Status = "up"
	mon.StatusCode = 200
}

// checkViaSSH executes the uptime check from the target SSH server (FR-15) for internal services.
func (c *Checker) checkViaSSH(ctx context.Context, mon *model.UptimeMonitor) {
	if c.pool == nil || c.storage == nil {
		mon.Status = "down"
		mon.LastError = "SSH pool belum siap"
		return
	}
	srv, err := c.storage.GetServer(mon.ServerID)
	if err != nil {
		mon.Status = "down"
		mon.LastError = "Server SSH perantara tidak ditemukan"
		return
	}

	execCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	start := time.Now()
	if mon.Type == "tcp" {
		host, port, _ := net.SplitHostPort(mon.Target)
		cmd := fmt.Sprintf(
			"if command -v nc >/dev/null 2>&1; then nc -z -w 5 '%s' %s; else timeout 5 sh -c 'cat < /dev/null > /dev/tcp/%s/%s' 2>/dev/null; fi",
			host, port, host, port,
		)
		_, err := c.pool.ExecuteCommand(execCtx, srv, cmd)
		mon.LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			mon.Status = "down"
			mon.StatusCode = 0
			mon.LastError = fmt.Sprintf("Port TCP %s tertutup dari server %s", mon.Target, srv.Name)
			return
		}
		mon.Status = "up"
		mon.StatusCode = 200
		return
	}

	// HTTP(S) check via curl on remote server
	cmd := fmt.Sprintf("curl -s -o /dev/null -w '%%{http_code}\t%%{time_total}' --max-time 8 '%s'", mon.Target)
	out, err := c.pool.ExecuteCommand(execCtx, srv, cmd)
	mon.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		mon.Status = "down"
		mon.StatusCode = 0
		mon.LastError = fmt.Sprintf("curl gagal dari %s: %v", srv.Name, err)
		return
	}

	parts := strings.Split(strings.TrimSpace(out), "\t")
	if len(parts) >= 1 {
		code, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		mon.StatusCode = code
	}
	if len(parts) >= 2 {
		if sec, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil && sec > 0 {
			mon.LatencyMs = int64(sec * 1000)
		}
	}

	if mon.StatusCode >= 200 && mon.StatusCode < 400 {
		mon.Status = "up"
	} else if mon.StatusCode >= 400 && mon.StatusCode < 500 {
		mon.Status = "warning"
		mon.LastError = fmt.Sprintf("HTTP %d (via %s)", mon.StatusCode, srv.Name)
	} else {
		mon.Status = "down"
		mon.LastError = fmt.Sprintf("HTTP %d (via %s)", mon.StatusCode, srv.Name)
	}
}
