package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendTelegramAndWebhook(t *testing.T) {
	var receivedTelegram map[string]string
	var receivedWebhook map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bottest-token/sendMessage" {
			_ = json.NewDecoder(r.Body).Decode(&receivedTelegram)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		if r.URL.Path == "/webhook" {
			_ = json.NewDecoder(r.Body).Decode(&receivedWebhook)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	n := NewNotifier(nil, nil)
	n.telegramBaseURL = srv.URL

	if err := n.SendTelegram(context.Background(), "test-token", "123456", "CPU tinggi 95%"); err != nil {
		t.Fatalf("SendTelegram failed: %v", err)
	}
	if receivedTelegram["chat_id"] != "123456" || receivedTelegram["text"] != "CPU tinggi 95%" || receivedTelegram["parse_mode"] != "HTML" {
		t.Errorf("unexpected telegram payload: %+v", receivedTelegram)
	}

	if err := n.SendWebhook(context.Background(), srv.URL+"/webhook", "Disk penuh 92%"); err != nil {
		t.Fatalf("SendWebhook failed: %v", err)
	}
	if receivedWebhook["content"] != "Disk penuh 92%" || receivedWebhook["text"] != "Disk penuh 92%" {
		t.Errorf("unexpected webhook payload: %+v", receivedWebhook)
	}
}
