package uptime

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"serverwatch/internal/model"
)

func TestValidateMonitorTarget(t *testing.T) {
	if err := ValidateMonitorTarget("http", "https://example.com/health"); err != nil {
		t.Errorf("expected valid https URL, got: %v", err)
	}
	if err := ValidateMonitorTarget("tcp", "127.0.0.1:6379"); err != nil {
		t.Errorf("expected valid tcp target, got: %v", err)
	}

	invalid := []struct {
		typ string
		tgt string
	}{
		{"http", "ftp://example.com"},
		{"http", "https://example.com;rm -rf /"},
		{"tcp", "127.0.0.1:99999"},
		{"tcp", "localhost"},
	}
	for _, tc := range invalid {
		if err := ValidateMonitorTarget(tc.typ, tc.tgt); err == nil {
			t.Errorf("expected %s %q to be rejected", tc.typ, tc.tgt)
		}
	}
}

func TestCheckLocalHTTPAndTCP(t *testing.T) {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer httpSrv.Close()

	checker := NewChecker(nil, nil)

	monHTTP := &model.UptimeMonitor{
		Name:   "Test HTTP",
		Type:   "http",
		Target: httpSrv.URL,
	}
	res, err := checker.CheckMonitor(context.Background(), monHTTP)
	if err != nil {
		t.Fatalf("CheckMonitor HTTP failed: %v", err)
	}
	if res.Status != "up" || res.StatusCode != 200 {
		t.Errorf("expected status=up and code=200, got %+v", res)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen tcp: %v", err)
	}
	defer ln.Close()

	monTCP := &model.UptimeMonitor{
		Name:   "Test TCP",
		Type:   "tcp",
		Target: ln.Addr().String(),
	}
	resTCP, err := checker.CheckMonitor(context.Background(), monTCP)
	if err != nil {
		t.Fatalf("CheckMonitor TCP failed: %v", err)
	}
	if resTCP.Status != "up" {
		t.Errorf("expected TCP status=up, got %+v", resTCP)
	}
}
