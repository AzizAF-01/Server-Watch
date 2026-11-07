package backup

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"serverwatch/internal/model"
)

func TestS3ClientUploadAndTestConnection(t *testing.T) {
	var receivedAuth string
	var receivedPayload []byte
	var receivedPath string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedPath = r.URL.Path
		if r.Method == http.MethodPut {
			b, _ := io.ReadAll(r.Body)
			receivedPayload = b
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult></ListBucketResult>`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	client := NewS3Client()
	cfg := model.S3BackupSettings{
		Enabled:     true,
		Endpoint:    ts.URL,
		Region:      "auto",
		Bucket:      "prod-backups",
		PathPrefix:  "serverwatch",
		AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
	}
	secret := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

	if err := client.TestConnection(context.Background(), cfg, secret); err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
	if !strings.HasPrefix(receivedAuth, "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/") {
		t.Fatalf("expected AWS4-HMAC-SHA256 header, got %q", receivedAuth)
	}

	s3URI, err := client.UploadObject(context.Background(), cfg, secret, "backup_mysql_app_20261103.sql.gz", []byte("dummy-gzip-content"))
	if err != nil {
		t.Fatalf("UploadObject failed: %v", err)
	}
	if s3URI != "s3://prod-backups/serverwatch/backup_mysql_app_20261103.sql.gz" {
		t.Fatalf("unexpected s3URI: %s", s3URI)
	}
	if receivedPath != "/prod-backups/serverwatch/backup_mysql_app_20261103.sql.gz" {
		t.Fatalf("unexpected request path: %s", receivedPath)
	}
	if string(receivedPayload) != "dummy-gzip-content" {
		t.Fatalf("unexpected payload: %q", string(receivedPayload))
	}

	// Test streaming file upload
	tmpFile, err := os.CreateTemp("", "test-s3-stream-*.sql.gz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	largeDummyData := strings.Repeat("ABCDEF1234567890", 1024) // 16 KB payload
	if _, err := tmpFile.WriteString(largeDummyData); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	_ = tmpFile.Close()

	s3URIFile, err := client.UploadFile(context.Background(), cfg, secret, "large_backup.sql.gz", tmpFile.Name())
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	if s3URIFile != "s3://prod-backups/serverwatch/large_backup.sql.gz" {
		t.Fatalf("unexpected s3URIFile: %s", s3URIFile)
	}
	if string(receivedPayload) != largeDummyData {
		t.Fatalf("payload mismatch in UploadFile")
	}
}

func TestExtractRemoteFilePathAndValidateIdentifier(t *testing.T) {
	p1, err := ExtractRemoteFilePath("/var/backups/serverwatch/backup_mysql_db_20261103.sql.gz")
	if err != nil || p1 != "/var/backups/serverwatch/backup_mysql_db_20261103.sql.gz" {
		t.Fatalf("unexpected p1: %v, %v", p1, err)
	}

	p2, err := ExtractRemoteFilePath("Remote: /var/backups/serverwatch/backup_mysql_db.sql.gz | S3: s3://bucket/key")
	if err != nil || p2 != "/var/backups/serverwatch/backup_mysql_db.sql.gz" {
		t.Fatalf("unexpected p2: %v, %v", p2, err)
	}

	if _, err := ExtractRemoteFilePath("/var/backups/../etc/passwd"); err == nil {
		t.Fatalf("expected path traversal rejection")
	}

	if _, err := ExtractRemoteFilePath("/var/backups/test.sql.gz; rm -rf /"); err == nil {
		t.Fatalf("expected command injection rejection")
	}

	if err := ValidateIdentifier("my_app_db_01"); err != nil {
		t.Fatalf("expected valid identifier, got %v", err)
	}
	if err := ValidateIdentifier("db; DROP TABLE users"); err == nil {
		t.Fatalf("expected invalid identifier rejection")
	}

	if err := ValidateRemotePath("/var/backups/serverwatch"); err != nil {
		t.Fatalf("expected valid remote path, got %v", err)
	}
	if err := ValidateRemotePath("/var/backups/../etc"); err == nil {
		t.Fatalf("expected traversal rejection in ValidateRemotePath")
	}
	if err := ValidateRemotePath("/var/backups; id; #"); err == nil {
		t.Fatalf("expected command injection rejection in ValidateRemotePath")
	}
}

