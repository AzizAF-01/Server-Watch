package backup

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"serverwatch/internal/model"
)

// S3Client provides agentless, pure-Go S3 Signature Version 4 uploads for AWS S3, Cloudflare R2, and MinIO.
type S3Client struct {
	httpClient *http.Client
}

func NewS3Client() *S3Client {
	return &S3Client{
		httpClient: &http.Client{
			Timeout: 3 * time.Minute,
		},
	}
}

func ValidateS3Settings(cfg model.S3BackupSettings, secretKey string) error {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return fmt.Errorf("endpoint S3/R2 harus diisi (contoh: https://<account>.r2.cloudflarestorage.com)")
	}
	u, err := url.Parse(strings.TrimSpace(cfg.Endpoint))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("format URL endpoint S3 tidak valid")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return fmt.Errorf("nama bucket S3/R2 harus diisi")
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" {
		return fmt.Errorf("Access Key ID S3/R2 harus diisi")
	}
	if strings.TrimSpace(secretKey) == "" {
		return fmt.Errorf("Secret Access Key S3/R2 belum dikonfigurasi di OS Keychain")
	}
	return nil
}

// TestConnection verifies bucket access using a signed HEAD or ListObjectsV2 request.
func (c *S3Client) TestConnection(ctx context.Context, cfg model.S3BackupSettings, secretKey string) error {
	if err := ValidateS3Settings(cfg, secretKey); err != nil {
		return err
	}

	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	bucket := strings.Trim(strings.TrimSpace(cfg.Bucket), "/")
	reqURL := fmt.Sprintf("%s/%s?list-type=2&max-keys=1", endpoint, bucket)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	if err := signS3Request(req, nil, strings.TrimSpace(cfg.AccessKeyID), strings.TrimSpace(secretKey), region, time.Now().UTC()); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal menghubungi endpoint S3: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("bucket mengembalikan HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

// UploadObject uploads a backup archive to S3/R2/MinIO using AWS Signature V4.
func (c *S3Client) UploadObject(ctx context.Context, cfg model.S3BackupSettings, secretKey string, fileName string, payload []byte) (string, error) {
	if err := ValidateS3Settings(cfg, secretKey); err != nil {
		return "", err
	}

	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	bucket := strings.Trim(strings.TrimSpace(cfg.Bucket), "/")
	prefix := strings.Trim(strings.TrimSpace(cfg.PathPrefix), "/")

	objectKey := fileName
	if prefix != "" {
		objectKey = prefix + "/" + fileName
	}

	reqURL := fmt.Sprintf("%s/%s/%s", endpoint, bucket, objectKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.ContentLength = int64(len(payload))
	req.Header.Set("Content-Type", "application/gzip")

	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	if err := signS3Request(req, payload, strings.TrimSpace(cfg.AccessKeyID), strings.TrimSpace(secretKey), region, time.Now().UTC()); err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload S3 gagal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("S3 menolak upload (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return fmt.Sprintf("s3://%s/%s", bucket, objectKey), nil
}

// UploadFile streams a file from local disk directly to S3/R2/MinIO using AWS Signature V4 without loading the entire payload into RAM.
func (c *S3Client) UploadFile(ctx context.Context, cfg model.S3BackupSettings, secretKey string, fileName string, localFilePath string) (string, error) {
	if err := ValidateS3Settings(cfg, secretKey); err != nil {
		return "", err
	}

	f, err := os.Open(localFilePath)
	if err != nil {
		return "", fmt.Errorf("gagal membuka file lokal untuk upload S3: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	size, err := io.Copy(hasher, f)
	if err != nil {
		return "", fmt.Errorf("gagal menghitung checksum file: %w", err)
	}
	payloadHash := hex.EncodeToString(hasher.Sum(nil))

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("gagal rewind stream file: %w", err)
	}

	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	bucket := strings.Trim(strings.TrimSpace(cfg.Bucket), "/")
	prefix := strings.Trim(strings.TrimSpace(cfg.PathPrefix), "/")

	objectKey := fileName
	if prefix != "" {
		objectKey = prefix + "/" + fileName
	}

	reqURL := fmt.Sprintf("%s/%s/%s", endpoint, bucket, objectKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, f)
	if err != nil {
		return "", err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/gzip")

	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	if err := signS3RequestWithHash(req, payloadHash, strings.TrimSpace(cfg.AccessKeyID), strings.TrimSpace(secretKey), region, time.Now().UTC()); err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload S3 gagal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("S3 menolak upload (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return fmt.Sprintf("s3://%s/%s", bucket, objectKey), nil
}

func signS3Request(req *http.Request, payload []byte, accessKey, secretKey, region string, now time.Time) error {
	return signS3RequestWithHash(req, sha256Hex(payload), accessKey, secretKey, region, now)
}

func signS3RequestWithHash(req *http.Request, payloadHash string, accessKey, secretKey, region string, now time.Time) error {
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	host := req.URL.Host
	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalQuery := canonicalQueryString(req.URL.Query())

	signedHeaders := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	sort.Strings(signedHeaders)

	var canonicalHeaders strings.Builder
	for _, h := range signedHeaders {
		var val string
		if h == "host" {
			val = host
		} else {
			val = strings.TrimSpace(req.Header.Get(h))
		}
		canonicalHeaders.WriteString(h)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(val)
		canonicalHeaders.WriteString("\n")
	}

	signedHeadersStr := strings.Join(signedHeaders, ";")
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders.String(),
		signedHeadersStr,
		payloadHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	signature := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeadersStr, signature,
	)
	req.Header.Set("Authorization", authHeader)
	return nil
}

func canonicalQueryString(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		vals := q[k]
		sort.Strings(vals)
		escapedKey := strings.ReplaceAll(url.QueryEscape(k), "+", "%20")
		for _, v := range vals {
			escapedVal := strings.ReplaceAll(url.QueryEscape(v), "+", "%20")
			parts = append(parts, escapedKey+"="+escapedVal)
		}
	}
	return strings.Join(parts, "&")
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
