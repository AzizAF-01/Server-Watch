# Panduan Kontribusi ServerWatch

Terima kasih atas ketertarikan Anda untuk berkontribusi pada ServerWatch! Untuk menjaga konsistensi arsitektur, stabilitas, dan keamanan sistem, seluruh kontribusi wajib mengikuti alur kerja terstruktur.

---

## 🚨 Siklus Hidup Standar Kontribusi (Issue ➔ Branch ➔ PR ➔ Merge ➔ Hapus Branch)

Setiap pengerjaan tugas — **baik penambahan fitur baru, perbaikan bug, pembersihan UI, maupun refactoring** — WAJIB mengikuti 5 langkah berikut:

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

### 1. Buka Issue Terlebih Dahulu
* **Wajib**: Dilarang membuat branch atau menulis kode sebelum Issue dibuat dan didiskusikan.
* Buka tab **Issues** di repositori GitHub ini, lalu pilih template yang sesuai:
  - **Permintaan Fitur (Feature Request)**
  - **Laporan Bug / Perbaikan (Bug Report / Chore)**
* Isi latar belakang, solusi yang diusulkan, dan kriteria penerimaan (*acceptance criteria*).

### 2. Buat Branch Baru
* Dilarang melakukan commit langsung ke branch `main`.
* Buat branch baru dari `main` dengan format penamaan standar:
  - `feat/<nomor-issue>-<deskripsi-singkat>` (fitur baru)
  - `fix/<nomor-issue>-<deskripsi-singkat>` (perbaikan bug)
  - `style/<nomor-issue>-<deskripsi-singkat>` (tampilan / UI)
  - `refactor/<nomor-issue>-<deskripsi-singkat>` (pembersihan / restrukturisasi kode)
  - `docs/<nomor-issue>-<deskripsi-singkat>` (dokumentasi)
  ```bash
  git checkout -b feat/12-grafik-disk-io
  ```

### 3. Koding & Pengujian Mandiri
* Lakukan perubahan kode di branch tersebut.
* Pastikan semua unit test backend lulus:
  ```powershell
  go test ./...
  ```
* Pastikan build frontend tidak ada error:
  ```powershell
  npm --prefix frontend run build
  ```

### 4. Buka Pull Request (PR)
* Push branch ke remote: `git push origin <nama-branch>`.
* Buka PR ke branch `main`.
* Pada deskripsi PR, cantumkan kata kunci penutup issue, misalnya:
  `Closes #12` atau `Fixes #12`.

### 5. Merge & Hapus Branch
* Setelah Pull Request disetujui dan di-merge ke branch `main`:
  1. Kembali ke branch `main` dan tarik versi terbaru:
     ```bash
     git checkout main
     git pull origin main
     ```
  2. Hapus branch lokal:
     ```bash
     git branch -d <nama-branch>
     ```
  3. Hapus branch remote:
     ```bash
     git push origin --delete <nama-branch>
     ```
