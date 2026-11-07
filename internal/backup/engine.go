package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"serverwatch/internal/model"
	"serverwatch/internal/sshpool"
	"serverwatch/internal/storage"
)

var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)
var validRemotePathRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\./~]+$`)

// BackupProgressCallback delivers real-time stage progress updates for active backup executions
type BackupProgressCallback func(progress model.BackupProgressEvent)

type BackupEngine struct {
	pool       *sshpool.SSHPool
	storage    *storage.SQLiteStorage
	keychain   *storage.KeychainManager
	s3Client   *S3Client
	onProgress BackupProgressCallback
}

func NewBackupEngine(pool *sshpool.SSHPool, store *storage.SQLiteStorage, km *storage.KeychainManager) *BackupEngine {
	return &BackupEngine{
		pool:     pool,
		storage:  store,
		keychain: km,
		s3Client: NewS3Client(),
	}
}

// SetProgressCallback registers a listener for real-time backup execution events
func (e *BackupEngine) SetProgressCallback(cb BackupProgressCallback) {
	e.onProgress = cb
}

// ValidateIdentifier enforces SEC-16: only alphanumeric, underscore, dot, and hyphen are allowed
func ValidateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("nama tidak boleh kosong")
	}
	if !validIdentifierRegex.MatchString(name) {
		return fmt.Errorf("karakter tidak valid pada nama '%s': hanya diizinkan huruf, angka, underscore, dan hyphen (SEC-16)", name)
	}
	return nil
}

// ValidateRemotePath enforces SEC-05 & SEC-16: checks path characters and forbids directory traversal
func ValidateRemotePath(path string) error {
	p := strings.TrimSpace(path)
	if p == "" {
		return fmt.Errorf("path direktori remote tidak boleh kosong")
	}
	if !validRemotePathRegex.MatchString(p) || strings.Contains(p, "..") {
		return fmt.Errorf("path direktori remote mengandung karakter tidak valid: %s (SEC-16)", p)
	}
	return nil
}

// ExecuteBackup runs database dump, compression, safety check, and retention cleanup
func (e *BackupEngine) ExecuteBackup(ctx context.Context, srv *model.Server, dbCfg *model.DatabaseConfig, sched *model.BackupSchedule) (*model.BackupJobLog, error) {
	startTime := time.Now()
	schedID := ""
	if sched != nil {
		schedID = sched.ID
	}

	targetDB := dbCfg.DatabaseName
	if sched != nil && sched.DatabaseName != "" {
		targetDB = sched.DatabaseName
	}
	if targetDB == "" {
		targetDB = "all_databases"
	}
	if targetDB != "all_databases" {
		if err := ValidateIdentifier(targetDB); err != nil {
			return nil, err
		}
	}

	jobLog := &model.BackupJobLog{
		ScheduleID:       schedID,
		ServerID:         srv.ID,
		DatabaseConfigID: dbCfg.ID,
		DatabaseName:     targetDB,
		StartedAt:        startTime,
		Status:           "running",
	}

	logID, _ := e.storage.RecordBackupJobStart(jobLog)
	jobLog.ID = logID

	remoteDir := "/var/backups/serverwatch"
	retentionDays := 7
	storageTarget := "remote"

	if sched != nil {
		if sched.RemotePath != "" {
			if err := ValidateRemotePath(sched.RemotePath); err != nil {
				return nil, err
			}
			remoteDir = sched.RemotePath
		}
		if sched.RetentionDays > 0 {
			retentionDays = sched.RetentionDays
		}
		if sched.StorageTarget != "" {
			storageTarget = sched.StorageTarget
		}
	}

	emitProgress := func(stage, message string, pct int, fileSize int64, filePath, errMsg string) {
		if e.onProgress != nil {
			e.onProgress(model.BackupProgressEvent{
				JobID:            logID,
				ServerID:         srv.ID,
				ServerName:       srv.Name,
				DatabaseConfigID: dbCfg.ID,
				DatabaseName:     targetDB,
				StorageTarget:    storageTarget,
				Stage:            stage,
				Message:          message,
				ProgressPercent:  pct,
				StartedAt:        startTime,
				ElapsedSec:       time.Since(startTime).Seconds(),
				FileSizeBytes:    fileSize,
				FilePath:         filePath,
				ErrorMessage:     errMsg,
			})
		}
	}

	emitProgress("started", fmt.Sprintf("Menyiapkan backup database '%s' di server %s...", targetDB, srv.Name), 5, 0, "", "")

	// Pre-flight check: ensure destination directory exists and disk has sufficient space (SEC-18)
	emitProgress("preflight", "Pre-flight safety check: memeriksa direktori & ruang disk server...", 15, 0, "", "")
	ensureDirCmd := fmt.Sprintf("mkdir -p '%s' 2>/dev/null && [ -w '%s' ] && echo 'OK' || (mkdir -p ~/backups/serverwatch && echo 'FALLBACK')", remoteDir, remoteDir)
	dirRes, _ := e.pool.ExecuteCommand(ctx, srv, ensureDirCmd)
	if strings.Contains(dirRes, "FALLBACK") {
		homeOut, _ := e.pool.ExecuteCommand(ctx, srv, "echo $HOME")
		homeDir := strings.TrimSpace(homeOut)
		if homeDir != "" {
			remoteDir = fmt.Sprintf("%s/backups/serverwatch", strings.TrimRight(homeDir, "/"))
		} else {
			remoteDir = "~/backups/serverwatch"
		}
	}

	checkDiskCmd := fmt.Sprintf("df -P '%s' | tail -1", remoteDir)
	dfOut, err := e.pool.ExecuteCommand(ctx, srv, checkDiskCmd)
	if err == nil && strings.TrimSpace(dfOut) != "" {
		fields := strings.Fields(dfOut)
		if len(fields) >= 5 {
			pctStr := strings.TrimSuffix(fields[4], "%")
			if usedPct, convErr := strconv.Atoi(pctStr); convErr == nil {
				if usedPct >= 90 {
					errMsg := fmt.Sprintf("Pre-flight Safety Guard: Partisi penyimpanan disk hampir penuh (%d%% terpakai). Backup dibatalkan otomatis demi menjaga stabilitas server (SEC-18).", usedPct)
					_ = e.storage.UpdateBackupJobFinish(logID, "failed", 0, "", time.Since(startTime).Seconds(), errMsg)
					jobLog.Status = "failed"
					jobLog.ErrorMessage = errMsg
					emitProgress("failed", errMsg, 100, 0, "", errMsg)
					return jobLog, fmt.Errorf("%s", errMsg)
				}
			}
		}
	}

	// Retrieve DB password from Keychain (SEC-15)
	password, _ := e.keychain.GetDBPassword(dbCfg.ID)

	timestampStr := startTime.Format("20060102_150405")
	fileName := fmt.Sprintf("backup_%s_%s_%s.sql.gz", dbCfg.Type, targetDB, timestampStr)
	remoteFilePath := fmt.Sprintf("%s/%s", strings.TrimRight(remoteDir, "/"), fileName)

	// Construct dump command with gzip compression
	var dumpCmd string
	switch dbCfg.Type {
	case model.DBTypeMySQL:
		pwdEnv := ""
		if password != "" {
			escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
			pwdEnv = fmt.Sprintf("MYSQL_PWD='%s' ", escapedPwd)
		}
		host := dbCfg.Host
		if host == "" {
			host = "127.0.0.1"
		}
		port := dbCfg.Port
		if port <= 0 {
			port = 3306
		}
		user := dbCfg.User
		if user == "" {
			user = "root"
		}

		dbTargetArg := targetDB
		if targetDB == "all_databases" {
			dbTargetArg = "--all-databases"
		}

		// Use --single-transaction --quick to ensure zero-lock on InnoDB
		dumpCmd = fmt.Sprintf("%snice -n 19 ionice -c 3 mysqldump -h %s -P %d -u %s --single-transaction --quick --skip-lock-tables %s | gzip -9 > '%s'",
			pwdEnv, host, port, user, dbTargetArg, remoteFilePath)

	case model.DBTypePostgres:
		pwdEnv := ""
		if password != "" {
			escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
			pwdEnv = fmt.Sprintf("PGPASSWORD='%s' ", escapedPwd)
		}
		host := dbCfg.Host
		if host == "" {
			host = "127.0.0.1"
		}
		port := dbCfg.Port
		if port <= 0 {
			port = 5432
		}
		user := dbCfg.User
		if user == "" {
			user = "postgres"
		}

		dumpCmd = fmt.Sprintf("%snice -n 19 pg_dump -h %s -p %d -U %s -d %s --no-owner --clean | gzip -9 > '%s'",
			pwdEnv, host, port, user, targetDB, remoteFilePath)

	case model.DBTypeRedis:
		// SEC-15: Pass password via REDISCLI_AUTH environment variable instead of -a flag
		authEnv := ""
		if password != "" {
			escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
			authEnv = fmt.Sprintf("REDISCLI_AUTH='%s' ", escapedPwd)
		}
		host := dbCfg.Host
		if host == "" {
			host = "127.0.0.1"
		}
		port := dbCfg.Port
		if port <= 0 {
			port = 6379
		}
		dumpCmd = fmt.Sprintf("%sredis-cli -h %s -p %d --rdb '%s'", authEnv, host, port, remoteFilePath)

	default:
		errMsg := fmt.Sprintf("tipe database %s tidak didukung untuk backup", dbCfg.Type)
		_ = e.storage.UpdateBackupJobFinish(logID, "failed", 0, "", time.Since(startTime).Seconds(), errMsg)
		emitProgress("failed", errMsg, 100, 0, "", errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	emitProgress("dumping", fmt.Sprintf("Mengeksekusi dump database '%s' (%s) & kompresi stream gzip...", targetDB, strings.ToUpper(string(dbCfg.Type))), 35, 0, "", "")

	_, err = e.pool.ExecuteCommand(ctx, srv, dumpCmd)
	if err != nil {
		errMsg := fmt.Sprintf("Eksekusi perintah dump gagal: %v", err)
		if strings.Contains(err.Error(), "Access denied") || strings.Contains(err.Error(), "1044") {
			errMsg += fmt.Sprintf("\nPetunjuk: Akun database '%s' membutuhkan hak akses SELECT & LOCK TABLES pada database target. Jalankan di MySQL server:\nGRANT SELECT, LOCK TABLES, SHOW VIEW, TRIGGER ON `%s`.* TO '%s'@'%%'; FLUSH PRIVILEGES;", dbCfg.User, targetDB, dbCfg.User)
		}
		_ = e.storage.UpdateBackupJobFinish(logID, "failed", 0, "", time.Since(startTime).Seconds(), errMsg)
		jobLog.Status = "failed"
		jobLog.ErrorMessage = errMsg
		emitProgress("failed", errMsg, 100, 0, "", errMsg)
		return jobLog, fmt.Errorf("%s", errMsg)
	}

	emitProgress("verifying", "Memverifikasi integritas file arsip hasil dump di server remote...", 65, 0, remoteFilePath, "")

	verifyCmd := fmt.Sprintf("wc -c '%s' 2>/dev/null", remoteFilePath)
	sizeOut, err := e.pool.ExecuteCommand(ctx, srv, verifyCmd)
	if err != nil || strings.TrimSpace(sizeOut) == "" {
		errMsg := "Gagal memverifikasi file backup di server remote."
		_ = e.storage.UpdateBackupJobFinish(logID, "failed", 0, "", time.Since(startTime).Seconds(), errMsg)
		jobLog.Status = "failed"
		jobLog.ErrorMessage = errMsg
		emitProgress("failed", errMsg, 100, 0, remoteFilePath, errMsg)
		return jobLog, fmt.Errorf("%s", errMsg)
	}

	fields := strings.Fields(sizeOut)
	fileSizeBytes, _ := strconv.ParseInt(fields[0], 10, 64)
	if fileSizeBytes <= 0 {
		errMsg := "File backup kosong (0 bytes). Terjadi kendala saat dumping database."
		_ = e.storage.UpdateBackupJobFinish(logID, "failed", 0, remoteFilePath, time.Since(startTime).Seconds(), errMsg)
		jobLog.Status = "failed"
		jobLog.ErrorMessage = errMsg
		emitProgress("failed", errMsg, 100, 0, remoteFilePath, errMsg)
		return jobLog, fmt.Errorf("%s", errMsg)
	}

	finalFilePath := remoteFilePath

	// Stream copy if storage target is local or S3
	if storageTarget == "local" {
		emitProgress("transferring", "Mentransfer dan menyimpan arsip ke folder laptop lokal...", 80, fileSizeBytes, remoteFilePath, "")
		localDir := ""
		if sched != nil {
			localDir = sched.LocalPath
		}
		if localDir == "" {
			userHome, _ := os.UserHomeDir()
			localDir = filepath.Join(userHome, "ServerWatch", "Backups")
		}
		_ = os.MkdirAll(localDir, 0755)
		localFilePath := filepath.Join(localDir, fileName)

		f, err := os.Create(localFilePath)
		if err == nil {
			streamErr := e.pool.StreamRemoteFile(ctx, srv, remoteFilePath, f)
			_ = f.Close()
			if streamErr == nil {
				finalFilePath = fmt.Sprintf("Remote: %s | Local: %s", remoteFilePath, localFilePath)
			} else {
				_ = os.Remove(localFilePath)
				errMsg := fmt.Sprintf("Gagal streaming arsip remote ke disk lokal: %v", streamErr)
				_ = e.storage.UpdateBackupJobFinish(logID, "failed", fileSizeBytes, remoteFilePath, time.Since(startTime).Seconds(), errMsg)
				jobLog.Status = "failed"
				jobLog.ErrorMessage = errMsg
				emitProgress("failed", errMsg, 100, fileSizeBytes, remoteFilePath, errMsg)
				return jobLog, fmt.Errorf("%s", errMsg)
			}
		} else {
			errMsg := fmt.Sprintf("Gagal membuat file backup lokal (%s): %v", localFilePath, err)
			_ = e.storage.UpdateBackupJobFinish(logID, "failed", fileSizeBytes, remoteFilePath, time.Since(startTime).Seconds(), errMsg)
			jobLog.Status = "failed"
			jobLog.ErrorMessage = errMsg
			emitProgress("failed", errMsg, 100, fileSizeBytes, remoteFilePath, errMsg)
			return jobLog, fmt.Errorf("%s", errMsg)
		}
	} else if storageTarget == "s3" && e.storage != nil && e.keychain != nil {
		emitProgress("transferring", "Mengunggah arsip ke Cloud Object Storage (S3 / R2)...", 80, fileSizeBytes, remoteFilePath, "")
		appCfg := e.storage.GetAppSettings()
		secretKey, _ := e.keychain.GetNotificationSecret("s3_secret_key")

		tmpFile, err := os.CreateTemp("", "serverwatch-s3-*.sql.gz")
		if err != nil {
			errMsg := fmt.Sprintf("Gagal membuat temporary file untuk upload S3: %v", err)
			_ = e.storage.UpdateBackupJobFinish(logID, "failed", fileSizeBytes, remoteFilePath, time.Since(startTime).Seconds(), errMsg)
			jobLog.Status = "failed"
			jobLog.ErrorMessage = errMsg
			emitProgress("failed", errMsg, 100, fileSizeBytes, remoteFilePath, errMsg)
			return jobLog, fmt.Errorf("%s", errMsg)
		}
		defer os.Remove(tmpFile.Name())

		if streamErr := e.pool.StreamRemoteFile(ctx, srv, remoteFilePath, tmpFile); streamErr != nil {
			_ = tmpFile.Close()
			errMsg := fmt.Sprintf("Gagal streaming arsip remote untuk diunggah ke S3: %v", streamErr)
			_ = e.storage.UpdateBackupJobFinish(logID, "failed", fileSizeBytes, remoteFilePath, time.Since(startTime).Seconds(), errMsg)
			jobLog.Status = "failed"
			jobLog.ErrorMessage = errMsg
			emitProgress("failed", errMsg, 100, fileSizeBytes, remoteFilePath, errMsg)
			return jobLog, fmt.Errorf("%s", errMsg)
		}
		_ = tmpFile.Close()

		s3URI, s3Err := e.s3Client.UploadFile(ctx, appCfg.S3Backup, secretKey, fileName, tmpFile.Name())
		if s3Err != nil {
			errMsg := fmt.Sprintf("Dump berhasil di server, namun upload ke S3/R2 gagal: %v", s3Err)
			_ = e.storage.UpdateBackupJobFinish(logID, "failed", fileSizeBytes, remoteFilePath, time.Since(startTime).Seconds(), errMsg)
			jobLog.Status = "failed"
			jobLog.ErrorMessage = errMsg
			emitProgress("failed", errMsg, 100, fileSizeBytes, remoteFilePath, errMsg)
			return jobLog, fmt.Errorf("%s", errMsg)
		}
		finalFilePath = fmt.Sprintf("Remote: %s | S3: %s", remoteFilePath, s3URI)
	}

	// Purge dumps exceeding retention threshold
	if retentionDays > 0 {
		emitProgress("retention", "Membersihkan arsip lama sesuai masa retensi...", 95, fileSizeBytes, finalFilePath, "")
		purgeCmd := fmt.Sprintf("find '%s' -maxdepth 1 -name \"backup_%s_%s_*.sql.gz\" -type f -mtime +%d -delete 2>/dev/null",
			strings.ReplaceAll(remoteDir, "'", "'\\''"), dbCfg.Type, targetDB, retentionDays)
		_, _ = e.pool.ExecuteCommand(ctx, srv, purgeCmd)
	}

	duration := time.Since(startTime).Seconds()
	_ = e.storage.UpdateBackupJobFinish(logID, "success", fileSizeBytes, finalFilePath, duration, "")

	jobLog.Status = "success"
	jobLog.FileSizeBytes = fileSizeBytes
	jobLog.FilePath = finalFilePath
	jobLog.DurationSec = duration
	completed := time.Now()
	jobLog.CompletedAt = &completed

	emitProgress("completed", fmt.Sprintf("Backup sukses! (%.2f MB dalam %.1f detik)", float64(fileSizeBytes)/(1024*1024), duration), 100, fileSizeBytes, finalFilePath, "")

	return jobLog, nil
}

// ExtractRemoteFilePath extracts and validates the remote path from a BackupJobLog.FilePath entry.
func ExtractRemoteFilePath(filePath string) (string, error) {
	raw := strings.TrimSpace(filePath)
	if raw == "" {
		return "", fmt.Errorf("lokasi file backup kosong")
	}
	if strings.Contains(raw, "|") {
		parts := strings.Split(raw, "|")
		raw = strings.TrimSpace(parts[0])
	}
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "Remote:"))
	if raw == "" || !validRemotePathRegex.MatchString(raw) || strings.Contains(raw, "..") {
		return "", fmt.Errorf("path file backup remote tidak valid: %s", raw)
	}
	return raw, nil
}

// VerifyBackupArchive checks archive existence, gzip integrity, and SHA-256 checksum on the remote server.
func (e *BackupEngine) VerifyBackupArchive(ctx context.Context, srv *model.Server, log *model.BackupJobLog) (*model.BackupVerifyResult, error) {
	remotePath, err := ExtractRemoteFilePath(log.FilePath)
	if err != nil {
		return nil, err
	}

	res := &model.BackupVerifyResult{
		BackupLogID: log.ID,
	}

	escapedPath := strings.ReplaceAll(remotePath, "'", "'\\''")
	var checkCmd string
	if strings.HasSuffix(remotePath, ".gz") {
		checkCmd = fmt.Sprintf("if [ ! -f '%s' ]; then echo 'MISSING'; else sz=$(wc -c < '%s' 2>/dev/null || echo 0); gzip -t '%s' 2>/dev/null && gz='OK' || gz='CORRUPT'; sha=$(sha256sum '%s' 2>/dev/null | awk '{print $1}'); echo \"FOUND|$sz|$gz|$sha\"; fi",
			escapedPath, escapedPath, escapedPath, escapedPath)
	} else {
		checkCmd = fmt.Sprintf("if [ ! -f '%s' ]; then echo 'MISSING'; else sz=$(wc -c < '%s' 2>/dev/null || echo 0); sha=$(sha256sum '%s' 2>/dev/null | awk '{print $1}'); echo \"FOUND|$sz|OK|$sha\"; fi",
			escapedPath, escapedPath, escapedPath)
	}

	out, err := e.pool.ExecuteCommand(ctx, srv, checkCmd)
	if err != nil {
		return nil, fmt.Errorf("gagal memverifikasi file di server: %w", err)
	}

	out = strings.TrimSpace(out)
	if out == "MISSING" || !strings.HasPrefix(out, "FOUND|") {
		res.FileExists = false
		res.Valid = false
		res.Message = fmt.Sprintf("File arsip %s sudah tidak ditemukan di server (mungkin telah terhapus oleh rotasi retensi).", remotePath)
		return res, nil
	}

	parts := strings.Split(out, "|")
	res.FileExists = true
	if len(parts) >= 2 {
		res.FileSizeBytes, _ = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	}
	gzState := ""
	if len(parts) >= 3 {
		gzState = strings.TrimSpace(parts[2])
	}
	if len(parts) >= 4 {
		res.SHA256 = strings.TrimSpace(parts[3])
	}

	if res.FileSizeBytes <= 0 {
		res.Valid = false
		res.Message = "File arsip ditemukan tetapi berukuran 0 bytes."
		return res, nil
	}
	if gzState != "OK" {
		res.Valid = false
		res.Message = "Peringatan: Arsip .gz gagal uji integritas (gzip -t corrupt)."
		return res, nil
	}

	res.Valid = true
	res.Message = fmt.Sprintf("Arsip valid & utuh (%d bytes). Integritas terverifikasi.", res.FileSizeBytes)
	return res, nil
}

// UploadBackupLogToS3 uploads an existing remote backup archive to the configured S3/R2/MinIO bucket.
func (e *BackupEngine) UploadBackupLogToS3(ctx context.Context, srv *model.Server, log *model.BackupJobLog) (string, error) {
	if e.storage == nil || e.keychain == nil {
		return "", fmt.Errorf("storage belum siap")
	}
	remotePath, err := ExtractRemoteFilePath(log.FilePath)
	if err != nil {
		return "", err
	}

	appCfg := e.storage.GetAppSettings()
	secretKey, _ := e.keychain.GetNotificationSecret("s3_secret_key")

	tmpFile, err := os.CreateTemp("", "serverwatch-s3-*.sql.gz")
	if err != nil {
		return "", fmt.Errorf("gagal membuat temporary file untuk upload S3: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := e.pool.StreamRemoteFile(ctx, srv, remotePath, tmpFile); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("gagal streaming file backup di server (%s): %w", remotePath, err)
	}
	_ = tmpFile.Close()

	parts := strings.Split(remotePath, "/")
	fileName := parts[len(parts)-1]

	s3URI, err := e.s3Client.UploadFile(ctx, appCfg.S3Backup, secretKey, fileName, tmpFile.Name())
	if err != nil {
		return "", err
	}

	if !strings.Contains(log.FilePath, "S3:") {
		updatedPath := fmt.Sprintf("%s | S3: %s", log.FilePath, s3URI)
		_ = e.storage.UpdateBackupJobFinish(log.ID, log.Status, log.FileSizeBytes, updatedPath, log.DurationSec, log.ErrorMessage)
	}

	return s3URI, nil
}

// TestS3Connection tests S3 / Cloudflare R2 / MinIO bucket connectivity.
func (e *BackupEngine) TestS3Connection(ctx context.Context, cfg model.S3BackupSettings, secretKey string) error {
	return e.s3Client.TestConnection(ctx, cfg, secretKey)
}

// RestoreDatabase restores a verified backup archive into the target database (SEC-15 & SEC-16).
func (e *BackupEngine) RestoreDatabase(ctx context.Context, srv *model.Server, dbCfg *model.DatabaseConfig, log *model.BackupJobLog, targetDBName string) error {
	remotePath, err := ExtractRemoteFilePath(log.FilePath)
	if err != nil {
		return err
	}

	targetDBName = strings.TrimSpace(targetDBName)
	if targetDBName == "" {
		targetDBName = log.DatabaseName
	}
	if targetDBName == "" {
		targetDBName = dbCfg.DatabaseName
	}
	if targetDBName != "all_databases" {
		if err := ValidateIdentifier(targetDBName); err != nil {
			return err
		}
	}

	verifyRes, err := e.VerifyBackupArchive(ctx, srv, log)
	if err != nil {
		return err
	}
	if !verifyRes.FileExists || !verifyRes.Valid {
		return fmt.Errorf("restorasi dibatalkan: %s", verifyRes.Message)
	}

	password, _ := e.keychain.GetDBPassword(dbCfg.ID)
	host := dbCfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := dbCfg.Port
	user := dbCfg.User

	escapedRemotePath := strings.ReplaceAll(remotePath, "'", "'\\''")
	var restoreCmd string
	switch dbCfg.Type {
	case model.DBTypeMySQL:
		if port <= 0 {
			port = 3306
		}
		if user == "" {
			user = "root"
		}
		pwdEnv := ""
		if password != "" {
			escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
			pwdEnv = fmt.Sprintf("MYSQL_PWD='%s' ", escapedPwd)
		}
		if targetDBName == "all_databases" {
			restoreCmd = fmt.Sprintf("gzip -dc '%s' | %smysql -h %s -P %d -u %s 2>&1", escapedRemotePath, pwdEnv, host, port, user)
		} else {
			createCmd := fmt.Sprintf("%smysql -h %s -P %d -u %s -e \"CREATE DATABASE IF NOT EXISTS \\`%s\\`;\" 2>&1", pwdEnv, host, port, user, targetDBName)
			_, _ = e.pool.ExecuteCommand(ctx, srv, createCmd)
			restoreCmd = fmt.Sprintf("gzip -dc '%s' | %smysql -h %s -P %d -u %s \\`%s\\` 2>&1", escapedRemotePath, pwdEnv, host, port, user, targetDBName)
		}

	case model.DBTypePostgres:
		if port <= 0 {
			port = 5432
		}
		if user == "" {
			user = "postgres"
		}
		if targetDBName == "" || targetDBName == "all_databases" {
			targetDBName = "postgres"
		}
		pwdEnv := ""
		if password != "" {
			escapedPwd := strings.ReplaceAll(password, "'", "'\\''")
			pwdEnv = fmt.Sprintf("PGPASSWORD='%s' ", escapedPwd)
		}
		createCmd := fmt.Sprintf("%spsql -h %s -p %d -U %s -d postgres -tc \"SELECT 1 FROM pg_database WHERE datname = '%s'\" | grep -q 1 || %spsql -h %s -p %d -U %s -d postgres -c \"CREATE DATABASE \\\"%s\\\";\" 2>/dev/null",
			pwdEnv, host, port, user, targetDBName, pwdEnv, host, port, user, targetDBName)
		_, _ = e.pool.ExecuteCommand(ctx, srv, createCmd)
		restoreCmd = fmt.Sprintf("gzip -dc '%s' | %spsql -h %s -p %d -U %s -d \"%s\" 2>&1", escapedRemotePath, pwdEnv, host, port, user, targetDBName)

	default:
		return fmt.Errorf("restore otomatis via wizard saat ini didukung untuk MySQL/MariaDB dan PostgreSQL")
	}

	out, err := e.pool.ExecuteCommand(ctx, srv, restoreCmd)
	if err != nil {
		return fmt.Errorf("proses restore gagal: %v (%s)", err, strings.TrimSpace(out))
	}
	if strings.Contains(out, "ERROR ") || strings.Contains(out, "FATAL:") {
		return fmt.Errorf("database mengembalikan error saat restore: %s", strings.TrimSpace(out))
	}

	return nil
}

// DownloadRemoteFile streams a remote backup file to local filesystem without memory truncation (SEC-02, SEC-15).
func (e *BackupEngine) DownloadRemoteFile(ctx context.Context, srv *model.Server, remotePath string, localDestination string) error {
	cleanPath, err := ExtractRemoteFilePath(remotePath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(localDestination)
	if dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	f, err := os.Create(localDestination)
	if err != nil {
		return fmt.Errorf("gagal membuat file tujuan lokal (%s): %w", localDestination, err)
	}
	defer f.Close()

	if err := e.pool.StreamRemoteFile(ctx, srv, cleanPath, f); err != nil {
		_ = os.Remove(localDestination)
		return err
	}
	return nil
}
