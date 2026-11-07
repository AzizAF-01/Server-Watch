# ServerWatch

Aplikasi Desktop Windows untuk Monitoring Server & Database Berbasis SSH Agentless (`.exe`).

## Fitur Utama
- **Agentless SSH Monitoring**: Memantau metrik Linux & WSL (CPU, RAM, Disk, Jaringan, Top Process) tanpa memasang software agent di server.
- **DevOps & Ekosistem Server**:
  - **Docker & Container Manager**: Memantau status, CPU, RAM, port mapping, log container, serta aksi `start`/`stop`/`restart` terproteksi.
  - **Systemd Service Manager**: Memantau status daemon infrastruktur utama dan unit gagal (`failed`), dilengkapi pembaca log dan aksi kontrol dengan dialog konfirmasi native OS (SEC-06).
  - **Port & Firewall Watch**: Inspeksi soket listening (`ss`/`netstat`) dengan klasifikasi eksposur publik (`0.0.0.0`/`::`) vs lokal (`127.0.0.1`), serta deteksi status firewall (`UFW`, `firewalld`, `iptables`).
  - **Audit Keamanan & Update OS**: Deteksi jumlah paket/security update tertunda (`apt`, `dnf`, `yum`, `apk`), deteksi percobaan brute-force SSH beserta daftar IP penyerang teratas, sesi login aktif (`who`), dan audit konfigurasi `sshd`/`fail2ban`.
- **Uptime & Sertifikat SSL Monitor (FR-14 & FR-15)**: Pemantauan HTTP(S) status code, response latency, koneksi port TCP, dan sisa hari masa berlaku sertifikat SSL/TLS (baik langsung dari aplikasi desktop maupun via SSH untuk endpoint internal).
- **Notifikasi Multi-Channel (FR-25, FR-26, SEC-08)**: Pengiriman alert real-time dan notifikasi pemulihan (*recovery*) ke Bot Telegram, Webhook Discord/Slack, dan banner desktop dengan penyimpanan token di Windows Credential Manager.
- **Export Laporan & Background Mode (FR-29, FR-31, FR-32, FR-33)**: Ekspor laporan infrastruktur ke CSV dan Laporan Eksekutif HTML/PDF siap cetak, pengaturan retensi data SQLite, opsi tetap memonitor di latar belakang saat jendela ditutup, serta Auto-Start Windows.
- **Monitoring Database**: Pemantauan metrik MySQL/MariaDB, PostgreSQL, dan Redis secara real-time.
- **Auto-Discovery Database**: Deteksi otomatis instans database yang aktif di server target.
- **Engine Backup Otomatis**: Dump database terjadwal dengan kompresi stream (`gzip`), zero-lock table (`--single-transaction`), rotasi retensi, dan perlindungan disk space.
- **Terminal & Quick Log Viewer**: Konsol SSH interaktif dan tail log remote terintegrasi.
- **Standar Keamanan Tinggi (SEC-01 s.d. SEC-18)**: Kredensial tersimpan di Windows Credential Manager, verifikasi host key fingerprint, dan sanitasi input anti-injection.

## Dokumentasi Teknis
- [**Product Requirements Document (PRD v0.4)**](docs/PRD.md): Spesifikasi fungsional, arsitektur, dan standar keamanan lengkap.
- [**Panduan Konfigurasi Target Server (Linux & WSL2)**](docs/TARGET_SETUP_GUIDE.md): Panduan langkah demi langkah menyiapkan server target (Linux VPS & WSL2 di laptop lain) dengan konfigurasi keamanan terbaik.
- [**Laporan Audit Keamanan (Security Audit Report)**](docs/SECURITY_AUDIT_REPORT.md): Laporan lengkap audit keamanan sistem, matriks kerentanan, dan panduan remediasi.

## Development & Build
- Mode Dev: `wails dev`
- Build Executable: `wails build -s`
- Binary Output: `build/bin/ServerWatch.exe`
