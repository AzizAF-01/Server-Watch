# Changelog

Semua perubahan penting pada proyek **ServerWatch** akan didokumentasikan di berkas ini.
Format ini didasarkan pada [Keep a Changelog](https://keepachangelog.com/id/1.0.0/) dan proyek ini mematuhi [Semantic Versioning](https://semver.org/lang/id/).

## [1.0.1] - 2026-11-15

### Diubah & Ditingkatkan (UI Polish)
- **Pembersihan Antarmuka**: Menghilangkan label kode spesifikasi internal (FR-xx dan SEC-xx) dari tab Pengaturan (*Settings*), Ekspor Laporan, dan Uptime agar tampilan lebih ramah pengguna dan profesional.
- **Dokumentasi & Arsitektur**: Menambahkan dokumen audit grafik arsitektur sistem (`docs/ARCHITECTURE_GRAPH.md`).
- **Aturan Kontribusi & Issue Template**: Menerapkan alur wajib pembuatan Issue sebelum penambahan fitur baru melalui `AGENTS.md`, `CONTRIBUTING.md`, dan formulir GitHub Issue Template.

### Keamanan & CI/CD Hardening
- **Pipeline Release 2-Stage**: Menerapkan approval gate (`release-approval`) pada GitHub Actions untuk mencegah eksekusi rilis liar jika token PAT bocor.
- **GitHub Artifact Attestation**: Penandatanganan biner secara kriptografis menggunakan Sigstore OIDC untuk membuktikan keaslian berkas installer.
- **Workflow CI Test**: Menambahkan pengujian otomatis terpisah untuk Pull Request.

## [1.0.0] - 2026-11-07

### Ditambahkan
- **Agentless SSH Monitoring**: Telemetri CPU, RAM, Disk, Network I/O, Uptime, dan Top Process untuk Linux & WSL2 tanpa install agent di server.
- **Docker & Container Manager**: Pemantauan kontainer aktif, penggunaan CPU/RAM, log tailing, serta aksi kontrol (`start`/`stop`/`restart`) terproteksi.
- **Systemd Service Manager**: Inspeksi unit systemd aktif maupun gagal (`failed`), pembaca log journald, dan kontrol daemon dengan konfirmasi native OS (SEC-06).
- **Port & Firewall Watch**: Inspeksi soket listening dengan klasifikasi eksposur publik (`0.0.0.0`/`::`) vs lokal (`127.0.0.1`), serta deteksi status firewall (`UFW`, `firewalld`, `iptables`).
- **Uptime & SSL Expiry Checker**: Monitoring HTTP(S) latency, status code, TCP port check, dan sisa hari masa berlaku sertifikat SSL/TLS.
- **Database Engine & Auto-Discovery**: Monitoring MySQL/MariaDB, PostgreSQL, dan Redis, dilengkapi penjelajah skema dan query explorer.
- **Automated Backup & Streaming Direct-to-S3**: Backup dump database terjadwal dengan kompresi gzip stream, zero-lock table (`--single-transaction`), rotasi retensi, dan streaming langsung ke AWS S3/MinIO tanpa membebani disk lokal.
- **Interactive SSH Terminal**: Konsol SSH interaktif dan tailing log remote berbasis WebSocket/Wails IPC.
- **Multi-Channel Alerts**: Notifikasi otomatis kondisi server ke Bot Telegram, Webhook Discord/Slack, dan Windows Toast Notification.
- **Report Exporter & Background Mode**: Ekspor metrik ke CSV dan Laporan Eksekutif HTML/PDF siap cetak, konfigurasi retensi database SQLite lokal, opsi minimize-to-tray, dan auto-start Windows.
- **CI/CD Automation**: GitHub Actions workflow untuk kompilasi biner otomatis dan publikasi aset rilis (`Setup.exe`, `Portable.zip`, `SHA256SUMS.txt`).

### Keamanan (Security Hardening)
- Penyimpanan seluruh kredensial (password SSH, passphrase key, password database, bot token, AWS S3 secret) di Windows Credential Manager via DPAPI (SEC-01 s.d. SEC-18).
- Remediasi lengkap kerentanan VULN-01 hingga VULN-10, termasuk pemisahan streaming I/O untuk file backup besar (>1MB) guna menghindari silent truncation buffer.
- Sanitasi input anti-command injection dan quoting ketat untuk identifier database PostgreSQL/MySQL.

### Aksesibilitas & UI (Anti-Slop Compliance)
- Rasio kontras warna status kritis (`--crit`) ditingkatkan menjadi 4.68:1 (memenuhi standar WCAG AA 4.5:1).
- Target sentuh/klik tombol aksi dan close banner distandarisasi minimal 28px.
- Dukungan navigasi keyboard dan screen reader lengkap (`role="tablist"`, `aria-selected`, `aria-label`, dan keyboard listener `Escape`).
