# ServerWatch - AI Agent Guidelines & Project Rules

Dokumen ini berisi pedoman dan aturan wajib bagi AI Agent (Antigravity, Claude, Gemini, Copilot, dll.) yang bekerja di repositori ServerWatch.

---

## 🚨 Aturan Wajib: Penambahan Fitur Baru (Feature Request Rule)

Setiap kali pengguna meminta penambahan fitur baru (*feature request* / request tambah fitur):

1. **DILARANG langsung menulis kode implementasi sebelum Issue GitHub dibuat.**
2. **Prosedur Wajib Penanganan Request Fitur:**
   - **Tahap 1 - Analisis Kebutuhan:** Analisis kebutuhan fitur, batasan teknis (*agentless* SSH, Windows Credential Manager, Wails binding), dan dampaknya terhadap arsitektur (merujuk ke [docs/PRD.md](docs/PRD.md) dan [docs/ARCHITECTURE_GRAPH.md](docs/ARCHITECTURE_GRAPH.md)).
   - **Tahap 2 - Susun Draf Issue:** Buatkan draf GitHub Issue terstruktur dan sajikan kepada pengguna untuk ditinjau:
     - **Judul Issue**: `[Feature]: <Deskripsi Singkat Fitur>`
     - **Latar Belakang / Masalah / User Story**: Mengapa fitur ini dibutuhkan?
     - **Solusi yang Diusulkan**: Bagaimana cara kerja dan implementasinya?
     - **Dampak Arsitektur & Keamanan**: Komponen apa saja yang terpengaruh dan aturan keamanan (SEC) apa yang harus dipatuhi?
     - **Kriteria Penerimaan (*Acceptance Criteria*)**: Daftar checklist pengujian agar fitur dinyatakan selesai.
   - **Tahap 3 - Buat / Konfirmasi Issue:** Minta konfirmasi pengguna untuk membuat Issue tersebut di GitHub (atau bantu buatkan via GitHub CLI `gh issue create` jika tersedia).
   - **Tahap 4 - Implementasi Kode:** Hanya setelah Issue terdaftar di GitHub (atau nomor issue telah ditentukan/disepakati pengguna), AI Agent diperbolehkan mulai mengedit kode atau membuat branch baru.
   - **Tahap 5 - Referensi Commit & PR:** Setiap commit dan PR yang berkaitan wajib menyertakan nomor Issue (contoh: `feat(db): tambahkan visualisasi slow query (closes #XX)`).

---

## Standar Kualitas & Keamanan Kode Proyek
- **Keamanan Kredensial (SEC-02, SEC-08):** Password SSH, secret S3, database password, dan token notifikasi HANYA boleh disimpan di Windows Credential Manager (OS Keychain), dilarang disimpan di SQLite atau dicetak dalam plain text.
- **Sanitasi Input (SEC-01, SEC-06):** Semua teks dari remote server/terminal/log wajib disanitasi sebelum ditampilkan di UI guna mencegah XSS dan Command Injection.
- **Pengujian Mandiri:** Setiap kode baru wajib divalidasi dengan `go test ./...` dan pengujian build frontend `npm run build`.
