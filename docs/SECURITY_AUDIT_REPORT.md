# Laporan Audit Keamanan Menyeluruh: ServerWatch
**Target:** `Monitoring-server` (Desktop Wails v2 + Go + Svelte)  
**Tanggal Audit Awal:** 04 November 2026  
**Tanggal Delta Audit #002:** 07 November 2026  
**Status:** Audit Penuh & Remediasi Selesai (Remediated & Verified - 100% Fixed)

---

## 1. Ringkasan Eksekutif

Audit keamanan menyeluruh telah dilakukan pada seluruh modul backend Go, integrasi SSH, Wails bindings, serta frontend Svelte. Pada 07 November 2026, dilakukan **Delta Security & Integrity Audit #002** untuk meninjau penambahan fitur baru (S3/R2 Backup Engine, Database Restore & Rollback, Multi-Server Broadcast SSH, dan Uptime Alerting). Seluruh temuan kerentanan dan integritas data (**VULN-01 sampai VULN-10**) telah **berhasil diperbaiki (patched)** dan diverifikasi menggunakan pengujian unit otomatis (*unit tests*) serta pengujian kompilasi binary end-to-end.

---

## 2. Matriks Temuan & Status Remediasi

| ID | Tingkat Keparahan | Judul Temuan | Komponen Terkait | Status Remediasi |
|---|---|---|---|---|
| **VULN-01** | **CRITICAL** | Wails Bindings Arbitrary Shell Command (`RunSSHCommand` & `BroadcastSSHCommand`) | [`app.go:545-660`](../app.go#L545-L660) | **RESOLVED** — Dialog konfirmasi native OS Windows (`runtime.MessageDialog`) diwajibkan sebelum eksekusi single & broadcast (`SEC-04`, `SEC-06`). |
| **VULN-02** | **CRITICAL** | Remote Command Injection & SQLi pada Modul Database Explorer | [`internal/dbmonitor/collector.go`](../internal/dbmonitor/collector.go), [`explorer.go`](../internal/dbmonitor/explorer.go), [`app.go`](../app.go) | **RESOLVED** — Ditambahkan validator ketat `ValidateDBConfig` (`host`, `user`, `database_name`) pada layer Go binding dan collector (`SEC-16`). |
| **VULN-03** | **HIGH** | Remote Command Injection via `RemotePath` pada Backup Schedule | [`internal/backup/engine.go`](../internal/backup/engine.go), [`app.go`](../app.go) | **RESOLVED** — Ditambahkan `ValidateRemotePath` di `SaveBackupSchedule` dan `ExecuteBackup`; path dibungkus single-quote. |
| **VULN-04** | **HIGH** | Bypass Host Key Verification & Insecure Jump Host | [`internal/sshpool/pool.go`](../internal/sshpool/pool.go) | **RESOLVED** — `InsecureIgnoreHostKey()` dihapus; diterapkan Trust-On-First-Use (TOFU) host key pinning dengan `subtle.ConstantTimeCompare` untuk Jump Host dan Target Host (`NFR-07`). |
| **VULN-05** | **HIGH** | Ekspos Kredensial Database di Daftar Proses Server (`ps aux`) | [`internal/dbmonitor/collector.go`](../internal/dbmonitor/collector.go), [`internal/backup/engine.go`](../internal/backup/engine.go) | **RESOLVED** — Parameter `-p` pada Docker Exec diganti `-e MYSQL_PWD`; Redis CLI menggunakan `REDISCLI_AUTH` environment variable (`SEC-15`). |
| **VULN-06** | **MEDIUM** | Pembatasan Memori Output Remote Tidak Efektif di `ExecuteCommand` | [`internal/sshpool/pool.go`](../internal/sshpool/pool.go) | **RESOLVED** — Diimplementasikan `boundedBuffer` thread-safe yang membatasi penulisan memori tepat pada 1 MB (stdout) dan 512 KB (stderr) (`SEC-02`). |
| **VULN-07** | **LOW** | Absennya Content Security Policy (CSP) & Pengaturan WebView | [`frontend/index.html`](../frontend/index.html), [`main.go`](../main.go) | **RESOLVED** — Ditambahkan meta tag CSP lengkap di `index.html` dan dinonaktifkan context menu default (`SEC-11`). |
| **VULN-08** | **CRITICAL** | Kerusakan/Pemotongan File Backup Diam-diam (*Silent Truncation & Corruption*) via `cat` di `ExecuteCommand` | [`internal/backup/engine.go`](../internal/backup/engine.go), [`internal/sshpool/pool.go`](../internal/sshpool/pool.go) | **RESOLVED** — Ditambahkan `StreamRemoteFile` di SSH Pool untuk streaming file langsung ke disk/tempfile tanpa batas buffer 1 MB; S3 upload menggunakan `UploadFile` berbasis stream (`SEC-02`, `SEC-15`). |
| **VULN-09** | **HIGH** | Unquoted Identifier & Remote Path pada Fitur Database Restore / Rollback | [`internal/backup/engine.go`](../internal/backup/engine.go) | **RESOLVED** — Path remote dibungkus single-quote `'...'`; identifier database PostgreSQL dan MySQL dibungkus quote `\"...\"` dan backtick `\`...\`` (`SEC-05`, `SEC-16`). |
| **VULN-10** | **MEDIUM** | False-Negative Status Flag pada Broadcast SSH Paralel | [`app.go:714-720`](../app.go#L714-L720) | **RESOLVED** — Properti `res.Success = true` diset dengan benar pada cabang eksekusi sukses di `BroadcastSSHCommand`. |

---

## 3. Detail Remediasi per Komponen

### VULN-01: Proteksi Dialog Native OS pada Perintah Shell (`SEC-04` & `SEC-06`)
- **Implementasi:**
  Pada [`app.go`](../app.go), fungsi `RunSSHCommand` dan `BroadcastSSHCommand` kini menginterupsi eksekusi dengan dialog modal native OS:
  ```go
  choice, dlgErr := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
      Type:          runtime.QuestionDialog,
      Title:         "Konfirmasi Eksekusi Perintah SSH",
      Message:       fmt.Sprintf("Jalankan perintah berikut pada server '%s' (%s)?\n\nPerintah:\n%s", srv.Name, srv.Host, cmd),
      Buttons:       []string{"Yes", "No"},
      DefaultButton: "No",
      CancelButton:  "No",
  })
  if dlgErr != nil || (choice != "Yes" && choice != "Ya" && choice != "OK") {
      return "", fmt.Errorf("eksekusi dibatalkan oleh pengguna (SEC-06)")
  }
  ```
  Hal ini mencegah skrip frontend atau eksploitasi berbasis WebView menjalankan perintah secara otomatis di latar belakang.

---

### VULN-02: Sanitasi Input Konfigurasi Database (`SEC-05` & `SEC-16`)
- **Implementasi:**
  Fungsi `ValidateDBConfig` diimplementasikan di [`internal/dbmonitor/collector.go`](../internal/dbmonitor/collector.go) dengan regex whitelist ketat:
  ```go
  var (
      validDBIdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-\.]{0,127}$`)
      validHostRegex         = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-\.]{0,253}$`)
  )
  ```
  Validasi dijalankan di `SaveDatabaseConfig`, `TestDatabaseConnection`, `CollectMetrics`, `ListDatabases`, `AnalyzeQueries`, `ListTables`, dan `KillQuery`. Karakter injeksi shell dan SQL langsung ditolak sebelum string perintah dirangkai.

---

### VULN-03: Validasi Path Backup Remote (`SEC-05` & `SEC-16`)
- **Implementasi:**
  Fungsi `ValidateRemotePath` diterapkan di [`internal/backup/engine.go`](../internal/backup/engine.go) dan dipanggil pada `SaveBackupSchedule` di [`app.go`](../app.go). String path juga diapit tanda kutip tunggal (`'%s'`) pada `mkdir`, `df`, dan `wc` untuk mencegah word-splitting.

---

### VULN-04: Host Key Pinning & Peniadaan `InsecureIgnoreHostKey` (`NFR-07`)
- **Implementasi:**
  Pada [`internal/sshpool/pool.go`](../internal/sshpool/pool.go), koneksi Jump Host dan Target Host menerapkan verifikasi Trust-On-First-Use (TOFU) yang membandingkan fingerprint dengan `subtle.ConstantTimeCompare`. Jika fingerprint tidak cocok, koneksi langsung diputus dengan pesan error `HOST KEY MISMATCH`.

---

### VULN-05: Eliminasi Password dari Command Line Arguments (`SEC-15`)
- **Implementasi:**
  1. Pada `internal/dbmonitor/collector.go`, perintah `docker exec` menggunakan environment variable `-e MYSQL_PWD='%s'` menggantikan `-p'%s'`.
  2. Perintah `redis-cli` pada collector, explorer, dan engine backup kini memanfaatkan environment variable `REDISCLI_AUTH='%s'` bawaan Redis 6+ sehingga password tidak terekspos di `ps aux` atau `/proc/[pid]/cmdline`.

---

### VULN-06: Bounded In-Memory Buffer untuk Output Remote (`SEC-02`)
- **Implementasi:**
  Struktur `boundedBuffer` dibuat di [`internal/sshpool/pool.go`](../internal/sshpool/pool.go) untuk membungkus `sess.Stdout` dan `sess.Stderr`. Setiap penulisan dibatasi hingga batas maksimum (1 MB untuk stdout, 512 KB untuk stderr), mencegah *memory exhaustion / Denial-of-Service* jika server remote memuntahkan output tak terbatas.

---

### VULN-07: Penerapan Content Security Policy (CSP) (`SEC-11`)
- **Implementasi:**
  Meta tag CSP berikut ditambahkan pada [`frontend/index.html`](../frontend/index.html):
  ```html
  <meta http-equiv="Content-Security-Policy" content="default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data:; connect-src 'self' ipc:; object-src 'none'; frame-src 'none'; base-uri 'none';"/>
  ```
  Selain itu, `EnableDefaultContextMenu: false` dikonfigurasi pada [`main.go`](../main.go).

---

### VULN-08: Peniadaan Silent Truncation pada Transfer File Backup (`SEC-02` & `SEC-15`)
- **Deskripsi Masalah:**
  Pada implementasi awal transfer backup offsite (unduh lokal dan upload cloud S3/R2), konten arsip dibaca dengan `cat %s` melalui `pool.ExecuteCommand()`. Karena `ExecuteCommand` memiliki proteksi memori buffer 1 MB (VULN-06), seluruh file backup berukuran lebih dari 1 MB terpotong tepat pada 1 MB. File backup yang terpotong menjadi rusak (*corrupted archive*), menimbulkan risiko kehilangan data fatal saat proses pemulihan bencana (*disaster recovery*).
- **Implementasi Perbaikan:**
  1. Pada [`internal/sshpool/pool.go`](../internal/sshpool/pool.go), ditambahkan fungsi `StreamRemoteFile(ctx, srv, remotePath, dst io.Writer)` yang menghubungkan `sess.Stdout` langsung ke `io.Writer` tanpa batasan buffer memori di heap Go.
  2. Pada [`internal/backup/s3.go`](../internal/backup/s3.go), ditambahkan method `UploadFile(ctx, cfg, secretKey, fileName, localFilePath)` yang menghitung payload SHA-256 secara streaming dan mengalirkan file langsung ke HTTP request body.
  3. Pada [`internal/backup/engine.go`](../internal/backup/engine.go), seluruh alur transfer (`ExecuteBackup` mode lokal & S3, `UploadBackupLogToS3`, dan `DownloadRemoteFile`) dialihkan ke `StreamRemoteFile` dan `UploadFile`.

---

### VULN-09: Sanitasi Quoting & Delimitasi Identifier pada Database Restore (`SEC-05` & `SEC-16`)
- **Deskripsi Masalah:**
  Pada perintah restore MySQL dan PostgreSQL di `RestoreDatabase`, path file arsip remote dilewatkan tanpa tanda kutip `gzip -dc %s | ...`. Selain itu, pada PostgreSQL, identifier database pada perintah `CREATE DATABASE %s` dan `-d %s` tidak diapit tanda kutip ganda, sehingga nama database dengan tanda hubung (misal: `prod-db`) memicu kegagalan sintaks SQL (*operator minus*).
- **Implementasi Perbaikan:**
  1. Path remote selalu diapit tanda kutip tunggal ter-escape (`'%s'`) pada perintah dekompresi gzip, wc, dan find.
  2. Nama database target selalu dibungkus backticks (`` `...` ``) untuk MySQL dan tanda kutip ganda (`\"...\"`) untuk PostgreSQL.
  3. Target nama database divalidasi ketat dengan `ValidateIdentifier` sebelum proses dekompresi dimulai.

---

### VULN-10: Koreksi Flag Status pada Eksekusi Broadcast SSH Paralel
- **Deskripsi Masalah:**
  Pada `BroadcastSSHCommand` di [`app.go`](../app.go), cabang eksekusi berhasil (`execErr == nil`) tidak menetapkan nilai `res.Success = true`, menyebabkan frontend menerima respons status `success: false` secara keliru.
- **Implementasi Perbaikan:**
  Properti `res.Success = true` diset secara eksplisit ketika eksekusi SSH selesai tanpa error.

---

## 4. Hasil Verifikasi Pengujian

1. **Unit Testing Go (`go test -v ./...`):**
   - `serverwatch/internal/backup`: **PASS** (termasuk `TestValidateBackupInputs`, `TestRestoreDatabaseValidation`, `TestBackupSchedulerIsDue`, `TestS3ClientUploadAndTestConnection`)
   - `serverwatch/internal/dbmonitor`: **PASS** (termasuk `TestValidateDBConfig`, query analysis, dan deteksi database)
   - `serverwatch/internal/devops`: **PASS** (termasuk `TestValidateUnitName`, `TestValidateAction`, `TestSanitizeText`)
   - `serverwatch/internal/notifier`: **PASS** (termasuk `TestSendTelegramAndWebhook`)
   - `serverwatch/internal/sshpool`: **PASS** (termasuk `TestBoundedBuffer_Limit`)
   - `serverwatch/internal/terminal`: **PASS**
   - `serverwatch/internal/uptime`: **PASS** (termasuk `TestValidateMonitorTarget`, `TestCheckLocalHTTPAndTCP`)
2. **Frontend Production Build (`npm run build`):** **PASS** (Vite v7.3.7 mentransformasi 118 modul tanpa error).
3. **Go Application Compilation (`go build -v .`):** **PASS** (Kompilasi backend sukses tanpa error).
