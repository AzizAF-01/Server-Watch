# Panduan Konfigurasi Target Server (Linux & WSL2) untuk ServerWatch

Dokumen ini adalah panduan teknis resmi bagi developer dan SysAdmin untuk menyiapkan server target (**Linux VPS / Server Fisik** maupun **WSL2 di laptop Windows lain**) agar dapat dipantau oleh aplikasi **ServerWatch** secara aman, stabil, dan memenuhi standar *least privilege* (hak akses minimal).

---

## Daftar Isi
1. [Prinsip Keamanan Dasar](#1-prinsip-keamanan-dasar)
2. [Skenario A: Konfigurasi Linux VPS / Server Fisik](#2-skenario-a-konfigurasi-linux-vps--server-fisik)
   - [2.1 Buat User Khusus Monitoring](#21-buat-user-khusus-monitoring)
   - [2.2 Pasang SSH Key (Ed25519)](#22-pasang-ssh-key-ed25519)
   - [2.3 Pengerasan (Hardening) SSH Server](#23-pengerasan-hardening-ssh-server)
   - [2.4 Konfigurasi Firewall (UFW / Firewalld)](#24-konfigurasi-firewall-ufw--firewalld)
3. [Skenario B: Konfigurasi WSL2 di Laptop Windows Lain](#3-skenario-b-konfigurasi-wsl2-di-laptop-windows-lain)
   - [3.1 Install & Jalankan OpenSSH Server di WSL](#31-install--jalankan-openssh-server-di-wsl)
   - [3.2 Auto-Start SSH Server saat WSL Menyala](#32-auto-start-ssh-server-saat-wsl-menyala)
   - [3.3 Opsi Jaringan 1: Mirrored Mode (Direkomendasikan)](#33-opsi-jaringan-1-mirrored-mode-direkomendasikan)
   - [3.4 Opsi Jaringan 2: Port Forwarding via Windows Host](#34-opsi-jaringan-2-port-forwarding-via-windows-host)
   - [3.5 Pengamanan Firewall Windows (IP Whitelisting)](#35-pengamanan-firewall-windows-ip-whitelisting)
4. [Konfigurasi Akses Monitoring & Backup Database](#4-konfigurasi-akses-monitoring--backup-database)
   - [4.1 MySQL / MariaDB](#41-mysql--mariadb)
   - [4.2 PostgreSQL](#42-postgresql)
   - [4.3 Redis](#43-redis)
5. [Tabel Periksa Kesiapan Keamanan (Security Checklist)](#5-tabel-periksa-kesiapan-keamanan-security-checklist)
6. [Pengujian Koneksi dari Laptop Admin](#6-pengujian-koneksi-dari-laptop-admin)

---

## 1. Prinsip Keamanan Dasar
ServerWatch bekerja secara **agentless** melalui koneksi SSH standar. Untuk menjamin keamanan server target:
* **Hindari menggunakan user `root`**. Gunakan user khusus non-root tanpa hak `sudo` bebas.
* **Gunakan SSH Key modern (`ed25519`)**, bukan autentikasi kata sandi biasa.
* **Batasi akses firewall (Whitelisting IP)** agar port SSH hanya bisa diakses oleh IP laptop admin ServerWatch, bukan seluruh perangkat di jaringan Wi-Fi/LAN.

---

## 2. Skenario A: Konfigurasi Linux VPS / Server Fisik

### 2.1 Buat User Khusus Monitoring
Jalankan di server target sebagai `root` atau user dengan `sudo`:
```bash
# 1. Buat user bernama 'serverwatch' tanpa shell login interaktif berbahaya (opsional)
sudo adduser --gecos "ServerWatch Monitor" --disabled-password serverwatch

# 2. Beri hak membaca log sistem (systemd journal & /var/log)
sudo usermod -aG systemd-journal,adm serverwatch
```

### 2.2 Pasang SSH Key (Ed25519)
Di laptop admin (laptop Anda):
```bash
# Generate key Ed25519 jika belum punya
ssh-keygen -t ed25519 -C "admin-serverwatch"
```
Salin public key (`id_ed25519.pub`) ke server target:
```bash
# Opsi 1: Menggunakan ssh-copy-id
ssh-copy-id -i ~/.ssh/id_ed25519.pub serverwatch@<IP_SERVER_TARGET>

# Opsi 2: Manual di server target
sudo mkdir -p /home/serverwatch/.ssh
echo "<ISI_PUBLIC_KEY_ANDA>" | sudo tee -a /home/serverwatch/.ssh/authorized_keys
sudo chown -R serverwatch:serverwatch /home/serverwatch/.ssh
sudo chmod 700 /home/serverwatch/.ssh
sudo chmod 600 /home/serverwatch/.ssh/authorized_keys
```

### 2.3 Pengerasan (Hardening) SSH Server
Edit konfigurasi SSH daemon di `/etc/ssh/sshd_config` atau tambahkan di `/etc/ssh/sshd_config.d/serverwatch.conf`:
```text
# Nonaktifkan login root langsung
PermitRootLogin no

# Wajibkan SSH Key, matikan login password
PasswordAuthentication no
PubkeyAuthentication yes

# Batasi user yang boleh login
AllowUsers serverwatch <user_admin_anda>

# Matikan X11 forwarding yang tidak diperlukan
X11Forwarding no
```
Terapkan perubahan:
```bash
sudo sshd -t && sudo systemctl restart ssh || sudo systemctl restart sshd
```

### 2.4 Konfigurasi Firewall (UFW / Firewalld)
**Jangan biarkan port SSH terbuka bebas ke publik!** Batasi hanya untuk IP laptop Anda:

* **Menggunakan UFW (Ubuntu/Debian)**:
```bash
# Izinkan port 22 HANYA dari IP laptop admin Anda
sudo ufw allow from <IP_LAPTOP_ADMIN> to any port 22 proto tcp comment 'ServerWatch SSH'
sudo ufw enable
```

* **Menggunakan Firewalld (RHEL/Rocky/AlmaLinux)**:
```bash
sudo firewall-cmd --permanent --zone=trusted --add-source=<IP_LAPTOP_ADMIN>
sudo firewall-cmd --permanent --zone=trusted --add-port=22/tcp
sudo firewall-cmd --reload
```

---

## 3. Skenario B: Konfigurasi WSL2 di Laptop Windows Lain

Arsitektur WSL2 menggunakan jaringan virtual **NAT internal**. Agar laptop admin dapat mengakses WSL di laptop lain melalui jaringan Wi-Fi/LAN, ikuti langkah berikut:

### 3.1 Install & Jalankan OpenSSH Server di WSL
Di terminal WSL laptop target:
```bash
# 1. Update paket & install OpenSSH server
sudo apt update && sudo apt install -y openssh-server

# 2. Buat host keys jika belum ada
sudo ssh-keygen -A

# 3. Jalankan service SSH
sudo service ssh start
```

### 3.2 Auto-Start SSH Server saat WSL Menyala
Agar service SSH otomatis aktif setiap kali WSL atau laptop menyala, tambahkan konfigurasi di `/etc/wsl.conf`:
```bash
sudo tee -a /etc/wsl.conf > /dev/null << 'EOF'
[boot]
command = service ssh start
EOF
```

---

### 3.3 Penanganan Jaringan di Windows 10 (Port Forwarding & Dynamic IP)
> **Catatan Windows 10**: Mode *Mirrored Networking* hanya tersedia di Windows 11. Pada Windows 10, WSL2 selalu menggunakan virtual network adapter (**NAT**), di mana alamat IP internal WSL (`172.x.x.x`) dapat berubah setelah laptop di-restart.

Untuk mengatasi ini secara praktis dan aman, gunakan script PowerShell otomatis berikut di Windows 10 target:

#### Script Otomatis Port Forwarding Windows 10 (`setup-wsl-ssh.ps1`)
Buka **PowerShell (Run as Administrator)** di laptop Windows 10 target dan jalankan perintah satu baris berikut:

```powershell
# 1. Tentukan IP laptop admin ServerWatch Anda (untuk keamanan whitelist firewall)
$ADMIN_IP = "192.168.1.15"  # <-- Ganti dengan IP Wi-Fi laptop ServerWatch Anda

# 2. Dapatkan IP internal WSL terbaru secara dinamis
$wslIp = (wsl bash -c "hostname -I").Trim().Split(' ')[0]
Write-Host "IP WSL Target terdeteksi: $wslIp" -ForegroundColor Cyan

# 3. Pastikan service SSH di dalam WSL aktif
wsl sudo service ssh start

# 4. Daftarkan port forwarding: Teruskan port 2222 Windows ke port 22 WSL
netsh interface portproxy delete v4tov4 listenport=2222 listenaddress=0.0.0.0 2>$null
netsh interface portproxy add v4tov4 listenaddress=0.0.0.0 listenport=2222 connectaddress=$wslIp connectport=22
Write-Host "Port forwarding 2222 -> $wslIp:22 berhasil diaktifkan." -ForegroundColor Green

# 5. Atur Firewall: Hanya izinkan koneksi dari IP Laptop Admin (SANGAT AMAN)
netsh advfirewall firewall delete rule name="Allow WSL SSH (ServerWatch)" 2>$null
netsh advfirewall firewall add rule name="Allow WSL SSH (ServerWatch)" dir=in action=allow protocol=TCP localport=2222 remoteip=$ADMIN_IP
Write-Host "Firewall Windows 10 berhasil dikunci: Hanya IP $ADMIN_IP yang diizinkan." -ForegroundColor Green
```

---

### 3.4 Menjalankan Otomatis saat Windows 10 Menyala (Opsional)
Jika Anda ingin port forwarding ini otomatis berjalan setiap kali laptop Windows 10 dinyalakan tanpa mengetik ulang:

1. Simpan script di atas sebagai file `C:\Scripts\wsl-portproxy.ps1`.
2. Buka **Task Scheduler** di Windows 10 (`taskschd.msc`).
3. Buat tugas baru (**Create Task**):
   * **General**: Centang *"Run with highest privileges"* (Administrator).
   * **Triggers**: Pilih *"At log on"* atau *"At startup"*.
   * **Actions**: Program: `powershell.exe`, Arguments: `-ExecutionPolicy Bypass -File C:\Scripts\wsl-portproxy.ps1`.

---

## 4. Konfigurasi Akses Monitoring & Backup Database

Jika server target menjalankan database, siapkan hak akses minimal (*least privilege*) agar ServerWatch bisa memantau metrik dan mengeksekusi backup tanpa downtime:

### 4.1 MySQL / MariaDB
Buat user database khusus `sw_monitor`:
```sql
-- Login ke MySQL: sudo mysql
CREATE USER 'sw_monitor'@'127.0.0.1' IDENTIFIED BY 'PasswordKuatDatabase123!';

-- Hak akses membaca telemetri & status server
GRANT PROCESS, SHOW DATABASES ON *.* TO 'sw_monitor'@'127.0.0.1';

-- Hak akses backup konsisten tanpa lock tabel (InnoDB single-transaction)
GRANT SELECT, LOCK TABLES, SHOW VIEW, EVENT, TRIGGER ON *.* TO 'sw_monitor'@'127.0.0.1';

FLUSH PRIVILEGES;
```

### 4.2 PostgreSQL
Buat role monitoring khusus:
```sql
-- Login ke PostgreSQL: sudo -u postgres psql
CREATE USER sw_monitor WITH PASSWORD 'PasswordKuatDatabase123!';

-- Gunakan role bawaan PostgreSQL pg_monitor (tersedia di PostgreSQL 10+)
GRANT pg_monitor TO sw_monitor;

-- Izin membaca metadata untuk backup
GRANT CONNECT ON DATABASE ecommerce_db TO sw_monitor;
```

### 4.3 Redis
* Jika Redis hanya melayani lokal, pastikan bind ke `127.0.0.1` di `/etc/redis/redis.conf`:
```text
bind 127.0.0.1 ::1
protected-mode yes
```
* Jika menggunakan `requirepass`, masukkan password tersebut di ServerWatch (disimpan terenkripsi di OS Keychain).

---

## 5. Tabel Periksa Kesiapan Keamanan (Security Checklist)

Sebelum mendaftarkan server ke ServerWatch, verifikasi checklist berikut:

| No | Komponen Keamanan | Rekomendasi Standar | Status Verifikasi |
|---|---|---|---|
| 1 | **Tipe Autentikasi** | Wajib SSH Key (`ed25519` / `rsa-4096`), `PasswordAuthentication no` | [ ] Terpasang |
| 2 | **Root SSH Login** | `PermitRootLogin no` di `/etc/ssh/sshd_config` | [ ] Terproteksi |
| 3 | **User Privilege** | User non-root (`serverwatch`), bukan `root` | [ ] Sesuai |
| 4 | **Firewall Restriction** | Port SSH hanya diizinkan dari IP laptop admin ServerWatch (`remoteip` / `ufw allow from`) | [ ] Terkunci |
| 5 | **Profil Jaringan Windows (WSL)** | Diatur ke *Private Network* | [ ] Terverifikasi |
| 6 | **Password Database** | Disimpan di Windows Credential Manager via ServerWatch, bukan plain text | [ ] Aman (SEC-15) |
| 7 | **Partisi Disk Backup** | Sisa ruang disk > 15% untuk cadangan dump (SEC-18) | [ ] Cukup |

---

## 6. Pengujian Koneksi dari Laptop Admin

Setelah semua langkah di atas selesai, lakukan verifikasi koneksi dari PowerShell laptop admin Anda:

```powershell
# Uji koneksi SSH ke target Linux VPS
ssh -i C:\Users\<NamaAnda>\.ssh\id_ed25519 -p 22 serverwatch@<IP_SERVER_TARGET>

# Uji koneksi SSH ke target WSL2 di laptop lain
ssh -i C:\Users\<NamaAnda>\.ssh\id_ed25519 -p 2222 <user_wsl>@<IP_WIFI_LAPTOP_LAIN>
```

Jika prompt shell terbuka dengan lancar tanpa meminta password, berarti server target telah **100% siap dan aman** untuk didaftarkan ke aplikasi **ServerWatch**!
