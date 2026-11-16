# ServerWatch - AI Agent Guidelines & Project Rules

Dokumen ini berisi pedoman dan aturan **WAJIB** bagi AI Agent (Antigravity, Claude, Gemini, Copilot, dll.) yang bekerja di repositori ServerWatch.

---

## 🚨 Siklus Hidup Standar Pengerjaan Tugas (Issue ➔ Branch ➔ PR ➔ Merge ➔ Hapus Branch)

Setiap pengerjaan tugas di repositori ini — **entah bersumber dari PRD, permintaan pengguna, bug fix, refactor, pembersihan UI, maupun penambahan fitur baru** — WAJIB mengikuti siklus hidup 5 langkah tanpa terkecuali:

```text
[1. Buat GitHub Issue] 
          ↓
[2. Buat Branch Baru: <type>/<issue-number>-<slug>] 
          ↓
[3. Koding & Pengujian Mandiri] 
          ↓
[4. Buka Pull Request (Closes #XX)] 
          ↓
[5. Merge & Wajib Hapus Branch (Lokal & Remote)]
```

### Langkah 1: Wajib Buat Issue Terlebih Dahulu
* **DILARANG** langsung mengedit kode atau membuat branch sebelum Issue GitHub dibuat (atau disepakati draf dan nomor Issue-nya).
* AI Agent wajib menyusun draf Issue terstruktur untuk pengguna:
  - **Judul Issue**: `[<Type>]: <Deskripsi Singkat>` (contoh: `[Feature]: ...`, `[Fix]: ...`, `[UI]: ...`, `[Refactor]: ...`)
  - **Latar Belakang / Masalah**: Apa yang mendasari tugas ini (merujuk PRD / kebutuhan pengguna)?
  - **Rencana Solusi**: Komponen apa saja yang diubah atau ditambahkan.
  - **Kriteria Penerimaan (*Acceptance Criteria*)**: Checklist kondisi sukses.
* Dapatkan konfirmasi pengguna atau buatkan Issue tersebut di GitHub sebelum lanjut ke Langkah 2.

### Langkah 2: Buat Branch Baru untuk Pengerjaan
* **DILARANG** melakukan commit langsung ke branch `main`.
* Buat branch baru dari `main` dengan format penamaan standar:
  - `feat/<nomor-issue>-<deskripsi-singkat>` (untuk fitur baru)
  - `fix/<nomor-issue>-<deskripsi-singkat>` (untuk perbaikan bug)
  - `style/<nomor-issue>-<deskripsi-singkat>` (untuk UI/tampilan)
  - `refactor/<nomor-issue>-<deskripsi-singkat>` (untuk pembersihan/restrukturisasi)
  - `docs/<nomor-issue>-<deskripsi-singkat>` (untuk dokumentasi)
  *(Contoh: `git checkout -b style/14-clean-prd-labels`)*.

### Langkah 3: Koding & Pengujian Mandiri
* Lakukan perubahan kode hanya pada branch tersebut.
* Setiap commit wajib menyertakan referensi Issue (contoh: `style(ui): bersihkan label PRD (refs #14)`).
* Wajib validasi sebelum membuat PR:
  - `go test ./...`
  - `npm --prefix frontend run build`
  - **Wajib Sinkronisasi Graphify**: Perbarui knowledge graph dan dokumentasi arsitektur proyek:
    ```powershell
    & (Get-Content graphify-out\.graphify_python) -m graphify update .
    Copy-Item graphify-out\GRAPH_REPORT.md docs\ARCHITECTURE_GRAPH.md -Force
    ```
    Pastikan berkas `docs/ARCHITECTURE_GRAPH.md` ikut disertakan dalam commit branch sebelum membuat PR.

### Langkah 4: Buka Pull Request (PR)
* Push branch ke remote: `git push origin <nama-branch>`.
* Buat Pull Request ke branch `main`.
* Isi deskripsi PR dengan kata kunci penutup Issue: `Closes #<nomor-issue>`.

### Langkah 5: Merge & Wajib Hapus Branch
* Setelah Pull Request di-merge ke branch `main`:
  1. Pindah kembali ke branch `main` dan tarik pembaruan:
     ```powershell
     git checkout main
     git pull origin main
     ```
  2. **Hapus branch kerja lokal**:
     ```powershell
     git branch -d <nama-branch>
     ```
  3. **Hapus branch kerja di remote (GitHub)**:
     ```powershell
     git push origin --delete <nama-branch>
     ```
* Repositori lokal dan remote harus selalu bersih dari branch yang sudah selesai di-merge.

---

## Standar Kualitas & Keamanan Kode Proyek
- **Keamanan Kredensial (SEC-02, SEC-08):** Password SSH, secret S3, database password, dan token notifikasi HANYA boleh disimpan di Windows Credential Manager (OS Keychain), dilarang disimpan di SQLite atau dicetak dalam plain text.
- **Sanitasi Input (SEC-01, SEC-06):** Semua teks dari remote server/terminal/log wajib disanitasi sebelum ditampilkan di UI guna mencegah XSS dan Command Injection.
- **Pengujian Mandiri:** Setiap kode baru wajib divalidasi dengan `go test ./...` dan pengujian build frontend `npm run build`.

---

## 🗺️ Aturan Pemeliharaan Knowledge Graph (Graphify)
Repositori ini memiliki peta arsitektur dan *knowledge graph* di `graphify-out/` dan [docs/ARCHITECTURE_GRAPH.md](docs/ARCHITECTURE_GRAPH.md).
- **Wajib Update Setiap Ada Perubahan**: Setiap kali ada penambahan fitur, perbaikan bug, refactor, atau modifikasi file kode/arsitektur, AI Agent **WAJIB** memperbarui grafik pengetahuan sebelum membuka Pull Request.
- **Perintah Pembaruan Cepat (Bebas Biaya Token / AST-only)**:
  ```powershell
  & (Get-Content graphify-out\.graphify_python) -m graphify update .
  Copy-Item graphify-out\GRAPH_REPORT.md docs\ARCHITECTURE_GRAPH.md -Force
  ```
- **Kueri Arsitektur**: Sebelum memodifikasi arsitektur atau menjawab pertanyaan struktural yang kompleks, AI Agent disarankan menjalankan `graphify query "<pertanyaan>"` guna menelusuri dependensi modul dan relasi antar-komponen secara akurat.

<!-- antislop:start -->
## antislop
For UI, copy, people, mobile layout, or code comments work, read `DESIGN.md` for direction, then `antislop.md` (core) and then the skill for the task:
- UI / visual: `skills/antislop-ui/SKILL.md`
- Copy & text: `skills/antislop-copywriting/SKILL.md`
- People: `skills/antislop-human/SKILL.md`
- Mobile / responsive: `skills/antislop-layoutmobile/SKILL.md`
- Code comments: `skills/antislop-code/SKILL.md`
Before starting, ask the user when antislop applies: during the work, or after it is done.
<!-- antislop:end -->

