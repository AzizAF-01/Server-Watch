# Skrip Build & Packaging Rilis Produksi ServerWatch (Windows AMD64)
$ErrorActionPreference = "Stop"

$nsisPath = "$env:LOCALAPPDATA\Programs\nsis-3.10"
if ((Test-Path $nsisPath) -and ($env:PATH -notlike "*$nsisPath*")) {
    $env:PATH = "$env:PATH;$nsisPath"
}

Write-Host "[1/4] Memaketkan frontend Svelte + Vite..." -ForegroundColor Cyan
npm --prefix frontend run build

Write-Host "[2/4] Mengompilasi biner produksi & NSIS Windows Installer..." -ForegroundColor Cyan
wails build -s -nsis -ldflags "-s -w"

$binDir = Join-Path $PSScriptRoot "..\build\bin"
$exeFile = Join-Path $binDir "ServerWatch.exe"
$setupFile = Join-Path $binDir "ServerWatch-v1.0.0-amd64-Setup.exe"
$portableZip = Join-Path $binDir "ServerWatch-v1.0.0-windows-amd64-Portable.zip"
$readmeFile = Join-Path $binDir "CARA_PAKAI.txt"
$checksumFile = Join-Path $binDir "SHA256SUMS.txt"

# Bersihkan file artefak sementara jika ada
Remove-Item -Path (Join-Path $binDir "ServerWatch.exe~") -ErrorAction SilentlyContinue
Remove-Item -Path (Join-Path $binDir "wails_init_serverwatch.exe") -ErrorAction SilentlyContinue

Write-Host "[3/4] Membuat paket Portable (.zip)..." -ForegroundColor Cyan
@"
======================================================================
ServerWatch v1.0.0 - Multi-Server SSH, Database & DevOps Monitoring
======================================================================

OPSI PENGGUNAAN:
1. Menggunakan Installer (Disarankan):
   Jalankan 'ServerWatch-v1.0.0-amd64-Setup.exe' untuk menginstal
   ServerWatch ke sistem beserta shortcut Desktop & Start Menu.

2. Menggunakan Versi Portable:
   Jalankan 'ServerWatch.exe' secara langsung tanpa instalasi.

PERSYARATAN SISTEM:
- Windows 10 (1809+) atau Windows 11 (64-bit / AMD64).
- Microsoft Edge WebView2 Runtime (sudah terpasang bawaan di Windows 10/11,
  atau otomatis diunduh oleh Setup Installer bila belum ada).

LOKASI DATA & KEAMANAN:
- Database konfigurasi & histori metrik disimpan secara lokal di:
  %APPDATA%\ServerWatch\serverwatch.db (atau folder kerja aplikasi).
- Password SSH, passphrase key, password Database, token Telegram,
  dan secret S3 disimpan terenkripsi di Windows Credential Manager (OS Keychain).
- Lokasi default hasil backup database lokal:
  %USERPROFILE%\ServerWatch\Backups (dapat diubah melalui tombol Browse).
"@ | Set-Content -Path $readmeFile -Encoding UTF8

if (Test-Path $portableZip) {
    Remove-Item $portableZip -Force
}
Compress-Archive -Path $exeFile, $readmeFile -DestinationPath $portableZip -Force

Write-Host "[4/4] Menghasilkan SHA-256 Checksum..." -ForegroundColor Cyan
$hashes = @()
foreach ($f in @($setupFile, $portableZip, $exeFile)) {
    if (Test-Path $f) {
        $h = Get-FileHash -Path $f -Algorithm SHA256
        $name = Split-Path $f -Leaf
        $hashes += "$($h.Hash.ToLower())  $name"
    }
}
$hashes | Set-Content -Path $checksumFile -Encoding UTF8

Write-Host "`nBuild Rilis Selesai! Artefak tersedia di build\bin:" -ForegroundColor Green
Get-ChildItem -Path $binDir -File | Select-Object Name, @{Name="Ukuran (MB)";Expression={[math]::Round($_.Length / 1MB, 2)}}, LastWriteTime | Format-Table -AutoSize
