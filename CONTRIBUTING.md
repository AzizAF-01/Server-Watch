# Panduan Kontribusi ServerWatch

Terima kasih atas ketertarikan Anda untuk berkontribusi pada ServerWatch! Untuk menjaga konsistensi arsitektur, stabilitas, dan keamanan sistem, kami menerapkan alur kerja terstruktur.

---

## 🚨 Aturan Utama: Wajib Membuat Issue Sebelum Request Fitur

> **PENTING**: Semua usulan fitur baru **WAJIB diawali dengan membuat Issue (Feature Request)** terlebih dahulu sebelum kode mulai ditulis atau Pull Request dibuka.

### Mengapa Aturan Ini Wajib?
1. **Penyelarasan Arsitektur**: Memastikan fitur baru tidak melanggar batasan arsitektur (seperti model *agentless*, keamanan kredensial di OS Keychain, dan batas memori SQLite).
2. **Diskusi Terbuka**: Memberi ruang bagi maintainer dan kontributor untuk mendiskusikan pendekatan teknis, desain antarmuka, dan prioritas sebelum ada waktu koding yang terbuang sia-sia.
3. **Pelacakan (*Traceability*)**: Memastikan setiap rilis memiliki riwayat perubahan (*changelog*) yang terhubung langsung ke kebutuhan pengguna.

---

## Alur Kerja Penambahan Fitur Baru

```text
[1. Buat Issue Feature Request]
              ↓
[2. Diskusi & Persetujuan Desain]
              ↓
[3. Buat Branch Baru: feat/issue-nomor-deskripsi]
              ↓
[4. Koding & Pengujian Mandiri (Unit Test & Build)]
              ↓
[5. Buka Pull Request (Hubungkan: Closes #XX)]
              ↓
[6. Review & Merge]
```

1. **Buka Issue Baru**:
   * Buka tab **Issues** di repositori GitHub ini.
   * Pilih template **Permintaan Fitur (Feature Request)**.
   * Isi latar belakang, solusi yang diusulkan, dan kriteria penerimaan (*acceptance criteria*).
2. **Tunggu Konfirmasi / Diskusi**:
   * Tim atau maintainer akan meninjau dan memberi label persetujuan.
3. **Buat Branch**:
   * Buat branch baru dari `main` dengan format:
     ```bash
     git checkout -b feat/<nomor-issue>-<deskripsi-singkat>
     # Contoh: git checkout -b feat/12-grafik-disk-io
     ```
4. **Validasi & Pengujian Lokal**:
   * Pastikan semua unit test backend lulus:
     ```powershell
     go test ./...
     ```
   * Pastikan build frontend tidak ada error:
     ```powershell
     cd frontend
     npm run build
     cd ..
     ```
5. **Buat Pull Request (PR)**:
   * Buat PR ke branch `main`.
   * Pada deskripsi PR, cantumkan kata kunci penutup issue, misalnya:
     `Closes #12` atau `Fixes #12`.
