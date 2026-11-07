package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"serverwatch/internal/model"
	"serverwatch/internal/storage"
)

// Notifier dispatches alert, recovery, and backup notifications across configured channels.
type Notifier struct {
	storage         *storage.SQLiteStorage
	keychain        *storage.KeychainManager
	httpClient      *http.Client
	telegramBaseURL string
	onDesktopNotify func(title, body, level string)
}

func NewNotifier(store *storage.SQLiteStorage, km *storage.KeychainManager) *Notifier {
	return &Notifier{
		storage:         store,
		keychain:        km,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		telegramBaseURL: "https://api.telegram.org",
	}
}

func (n *Notifier) SetDesktopCallback(fn func(title, body, level string)) {
	n.onDesktopNotify = fn
}

// DispatchAlert sends an alert or recovery notification according to user preferences.
func (n *Notifier) DispatchAlert(ctx context.Context, serverName, level, message string, isRecovery bool) {
	if n.storage == nil {
		return
	}
	settings := n.storage.GetAppSettings().Notifications
	if isRecovery && !settings.NotifyOnRecovery {
		return
	}

	statusPrefix := "[ALERT " + strings.ToUpper(level) + "]"
	if isRecovery {
		statusPrefix = "[RESOLVED / PULIH]"
		level = "ok"
	}

	timeStr := time.Now().Format("2006-01-02 15:04:05")

	var telegramText string
	if isRecovery {
		telegramText = fmt.Sprintf("✅ <b>SERVERWATCH: INSIDEN PULIH</b>\n━━━━━━━━━━━━━━━━━━━━\n🖥 <b>Server:</b> <code>%s</code>\n✨ <b>Kondisi:</b> %s\n⏱ <b>Waktu:</b> %s WIB",
			html.EscapeString(serverName), html.EscapeString(message), timeStr)
	} else if strings.EqualFold(level, "critical") {
		telegramText = fmt.Sprintf("🚨 <b>SERVERWATCH ALERT: KRITIS</b>\n━━━━━━━━━━━━━━━━━━━━\n🖥 <b>Server:</b> <code>%s</code>\n⚠️ <b>Kendala:</b> %s\n⏱ <b>Waktu:</b> %s WIB",
			html.EscapeString(serverName), html.EscapeString(message), timeStr)
	} else {
		telegramText = fmt.Sprintf("⚠️ <b>SERVERWATCH ALERT: PERINGATAN</b>\n━━━━━━━━━━━━━━━━━━━━\n🖥 <b>Server:</b> <code>%s</code>\n⚡ <b>Perhatian:</b> %s\n⏱ <b>Waktu:</b> %s WIB",
			html.EscapeString(serverName), html.EscapeString(message), timeStr)
	}

	title := fmt.Sprintf("ServerWatch %s - %s", statusPrefix, serverName)
	fullText := fmt.Sprintf("%s\nServer: %s\nDetail: %s\nWaktu: %s WIB",
		statusPrefix, serverName, message, timeStr)

	if settings.DesktopEnabled && n.onDesktopNotify != nil {
		n.onDesktopNotify(title, message, level)
	}

	if settings.TelegramEnabled && settings.TelegramChatID != "" && n.keychain != nil {
		if token, err := n.keychain.GetNotificationSecret("telegram_token"); err == nil && token != "" {
			go func() {
				_ = n.SendTelegram(ctx, token, settings.TelegramChatID, telegramText)
			}()
		}
	}

	if settings.WebhookEnabled && n.keychain != nil {
		if webhookURL, err := n.keychain.GetNotificationSecret("webhook_url"); err == nil && webhookURL != "" {
			go func() {
				_ = n.SendWebhook(ctx, webhookURL, fullText)
			}()
		}
	}
}

// DispatchBackupStatus sends a backup job completion or failure notification (FR-44).
func (n *Notifier) DispatchBackupStatus(ctx context.Context, serverName string, log *model.BackupJobLog) {
	if n.storage == nil || log == nil {
		return
	}
	settings := n.storage.GetAppSettings().Notifications
	if !settings.NotifyOnBackup {
		return
	}

	level := "ok"
	header := "[BACKUP SUKSES]"
	timeStr := time.Now().Format("2006-01-02 15:04:05")

	var telegramText string
	if log.Status == "success" {
		targetLabel := "Server Remote"
		if strings.Contains(log.FilePath, "Local:") {
			targetLabel = "Laptop Lokal (Offsite Stream)"
		} else if strings.Contains(log.FilePath, "S3:") {
			targetLabel = "Cloud S3 / R2"
		}
		telegramText = fmt.Sprintf("💾 <b>SERVERWATCH: BACKUP BERHASIL</b>\n━━━━━━━━━━━━━━━━━━━━\n🖥 <b>Server:</b> <code>%s</code>\n📦 <b>Database:</b> <code>%s</code>\n📊 <b>Ukuran:</b> <code>%.2f MB</code>\n⏱ <b>Durasi:</b> <code>%.1fs</code>\n📁 <b>Tujuan:</b> <code>%s</code>\n🕒 <b>Waktu Selesai:</b> %s WIB",
			html.EscapeString(serverName), html.EscapeString(log.DatabaseName), float64(log.FileSizeBytes)/(1024*1024), log.DurationSec, html.EscapeString(targetLabel), timeStr)
	} else {
		level = "critical"
		header = "[BACKUP GAGAL]"
		telegramText = fmt.Sprintf("❌ <b>SERVERWATCH: BACKUP GAGAL</b>\n━━━━━━━━━━━━━━━━━━━━\n🖥 <b>Server:</b> <code>%s</code>\n📦 <b>Database:</b> <code>%s</code>\n⚠️ <b>Kendala:</b> <code>%s</code>\n⏱ <b>Waktu:</b> %s WIB",
			html.EscapeString(serverName), html.EscapeString(log.DatabaseName), html.EscapeString(log.ErrorMessage), timeStr)
	}

	detail := fmt.Sprintf("Database: %s | Ukuran: %.2f MB | Durasi: %.1fs",
		log.DatabaseName, float64(log.FileSizeBytes)/(1024*1024), log.DurationSec)
	if log.Status != "success" {
		detail = fmt.Sprintf("Database: %s | Error: %s", log.DatabaseName, log.ErrorMessage)
	}

	fullText := fmt.Sprintf("%s\nServer: %s\n%s\nWaktu: %s WIB", header, serverName, detail, timeStr)
	if settings.DesktopEnabled && n.onDesktopNotify != nil {
		n.onDesktopNotify("ServerWatch "+header, detail, level)
	}

	if settings.TelegramEnabled && settings.TelegramChatID != "" && n.keychain != nil {
		if token, err := n.keychain.GetNotificationSecret("telegram_token"); err == nil && token != "" {
			go func() {
				_ = n.SendTelegram(ctx, token, settings.TelegramChatID, telegramText)
			}()
		}
	}

	if settings.WebhookEnabled && n.keychain != nil {
		if webhookURL, err := n.keychain.GetNotificationSecret("webhook_url"); err == nil && webhookURL != "" {
			go func() {
				_ = n.SendWebhook(ctx, webhookURL, fullText)
			}()
		}
	}
}

// SendTelegram posts a message to the Telegram Bot API.
func (n *Notifier) SendTelegram(ctx context.Context, botToken, chatID, text string) error {
	botToken = strings.TrimSpace(botToken)
	chatID = strings.TrimSpace(chatID)
	if botToken == "" || chatID == "" {
		return fmt.Errorf("bot token dan chat ID Telegram wajib diisi")
	}

	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", strings.TrimRight(n.telegramBaseURL, "/"), botToken)
	payload := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal menghubungi API Telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("Telegram mengembalikan HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}
	return nil
}

// SendWebhook posts a message to a Discord, Slack, or generic JSON webhook URL.
func (n *Notifier) SendWebhook(ctx context.Context, webhookURL, text string) error {
	webhookURL = strings.TrimSpace(webhookURL)
	if !strings.HasPrefix(webhookURL, "https://") && !strings.HasPrefix(webhookURL, "http://") {
		return fmt.Errorf("URL webhook harus diawali dengan https:// atau http://")
	}

	// Provide both "content" (Discord) and "text" (Slack / Mattermost / Generic)
	payload := map[string]string{
		"username": "ServerWatch",
		"content":  text,
		"text":     text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal mengirim ke webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook mengembalikan HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}
	return nil
}
