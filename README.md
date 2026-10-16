# Sistem Autentikasi Internal FTI

Sistem autentikasi berbasis web terpusat untuk kebutuhan portal internal PT Kuliner Kreasindo Inovasi (FTI). Aplikasi ini dibangun secara penuh menggunakan **Go** pada backend, basis data **PostgreSQL 18**, dan frontend antarmuka **React + TypeScript** yang di-serve oleh **Nginx**, serta diorkestrasi secara menyeluruh dengan **Docker Compose**.

---

## 1. Gambaran Singkat & Fitur

Sistem ini menyediakan layanan autentikasi berbasis sesi sisi server (*server-side stateful session*) dengan token acak berkekuatan kriptografis tinggi. Penyimpanan kata sandi telah memenuhi standar keamanan industri menggunakan algoritma Argon2id.

### Daftar Fitur Utama (Kebutuhan Minimum)
- **Pendaftaran Akun Publik (`/register`)**: Pendaftaran akun pengguna dengan validasi format email, username (3-30 karakter), dan kata sandi minimal 8 karakter. Peran yang diberikan selalu `user`.
- **Masuk Akun Fleksibel (`/login`)**: Dukungan masuk menggunakan satu kolom pengenal (dapat berupa alamat email atau username) beserta kata sandi.
- **Profil Pengguna (`/profile` & `GET /api/me`)**: Menampilkan detail akun terautentikasi (ID, username, email, dan label peran).
- **Pembatasan 2 Sesi Aktif Simultan (FIFO)**: Setiap akun dibatasi maksimal memiliki 2 sesi aktif secara bersamaan. Login ketiga akan otomatis mencabut (*evict*) sesi yang paling lama dibuat (`created_at` tertua) secara aman dan bebas *race condition*.
- **Manajemen Pengguna oleh Admin (`/admin/users`)**: Antarmuka khusus pengguna dengan peran `admin` untuk melihat seluruh akun pengguna, peran, status keaktifan, dan jumlah sesi login aktif secara *real-time*.
- **Force Logout oleh Admin**: Kemampuan admin untuk mencabut seluruh sesi aktif milik akun tertentu secara instan, dilengkapi dialog konfirmasi.
- **Proteksi Akses (Route Guard & Middleware)**: Pembatasan akses berbasis peran (`user` vs `admin`) di tingkat API backend dan navigasi frontend.
- **Deteksi Sesi Berakhir Otomatis**: Pengguna yang sesinya telah dicabut atau kedaluwarsa otomatis dialihkan ke halaman login dengan pesan ramah pengguna.
- **Inisialisasi Admin Otomatis (Idempoten)**: Akun admin pertama dibuat secara otomatis saat kontainer backend start tanpa risiko membuat akun ganda.

### Fitur Tambahan (Di Luar Kebutuhan Minimum)
- **Mitigasi Anti-Enumerasi Akun dengan Dummy Hash**: Saat ada percobaan login dengan email/username yang tidak terdaftar, backend mengeksekusi komputasi verifikasi Argon2id tiruan agar durasi waktu respons server identik (*constant-time response*), mencegah penyerang menebak keberadaan akun.
- **Perlindungan Login Konkuren dengan Row-Level Locking**: Menggunakan `SELECT ... FOR UPDATE` pada baris pengguna di PostgreSQL untuk mencegah tembusnya batas 2 sesi saat terjadi login paralel serentak (*parallel race condition*).
- **UI Responsif dengan Desain Gelap Modern**: Antarmuka bersih bergaya *glassmorphism* tanpa menggunakan *framework* UI berat eksternal, menjaga ukuran bundle tetap ringan dan cepat.

---

## 2. Cara Menjalankan Aplikasi

Aplikasi dirancang agar dapat dijalankan langsung dari kondisi repositori bersih dengan empat perintah sederhana:

```bash
# 1. Salin konfigurasi environment contoh
cp .env.example .env

# 2. Bangun image dan jalankan seluruh container (background)
docker compose up -d --build

# 3. Tunggu hingga service healthy dan periksa log (opsional)
docker compose ps

# 4. Buka aplikasi di browser
# Frontend: http://localhost:3000
# Backend Health Check: http://localhost:8080/health
```

### Kredensial Masuk Admin Pertama
Akun administrator pertama diinisialisasi secara otomatis berdasarkan konfigurasi `.env.example`:
- **Email**: `admin@fti.com`
- **Username**: `admin`
- **Kata Sandi**: `FtiKki123!`

> [!WARNING]
> **Peringatan Keamanan Penting**: Kredensial di atas merupakan nilai contoh bawaan dari `.env.example` untuk kebutuhan evaluasi lokal. **Sangat disarankan untuk segera mengganti kata sandi atau memperbarui variabel `ADMIN_PASSWORD` pada berkas `.env` sebelum digunakan pada lingkungan publik atau produksi.**

---

## 3. Daftar Endpoint API

Semua request dan response menggunakan format data `application/json`. Cookie sesi `auth_session` dikirimkan secara otomatis oleh browser (`credentials: 'include'`).

| Metode | Endpoint | Hak Akses | Deskripsi | Status Berhasil |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/health` | Publik | Diagnostik kesehatan server dan koneksi database | `200 OK` |
| `POST` | `/api/auth/register` | Publik | Mendaftarkan akun pengguna baru (peran otomatis `user`) | `201 Created` |
| `POST` | `/api/auth/login` | Publik | Autentikasi dengan identifier (email/username) & kata sandi | `200 OK` + `Set-Cookie` |
| `POST` | `/api/auth/logout` | Terautentikasi | Mencabut sesi saat ini dan menghapus cookie | `200 OK` |
| `GET` | `/api/me` | Terautentikasi | Mendapatkan data profil dan peran pengguna yang sedang login | `200 OK` |
| `GET` | `/api/admin/users` | Admin | Mendapatkan daftar seluruh pengguna beserta jumlah sesi aktif | `200 OK` |
| `POST` | `/api/admin/users/{id}/force-logout` | Admin | Mencabut seluruh sesi aktif milik pengguna target | `200 OK` |

---

## 4. Keputusan Teknis Penting

Keputusan arsitektur penting dicatat secara rinci pada [`docs/DECISIONS.md`](file:///c:/Programming/fti/fti-auth-system/docs/DECISIONS.md):

1. **Multi-Stage Docker Build (ADR-001)**: Backend Go dikompilasi ke binary statis murni tanpa CGO menggunakan `golang:1.27-alpine` lalu disalin ke runtime minimal `alpine:3.21`. Frontend dibuild dengan `node:22-alpine` lalu disajikan melalui `nginx:alpine`, menghasilkan image berukuran sangat kecil dan aman.
2. **Penanganan Volume PostgreSQL 18 (ADR-002)**: Mount volume dipetakan ke `/var/lib/postgresql` (bukan `/var/lib/postgresql/data`) agar kompatibel dengan utilitas manajemen klaster PostgreSQL 18 alpine.
3. **Migrasi Ter-embed & Idempoten (ADR-005 & ADR-006)**: Skema SQL di-embed langsung ke binary backend via `embed.FS` dan dieksekusi menggunakan `golang-migrate` saat start, dilanjutkan dengan pembuatan akun admin yang idempoten (tidak menimpa akun yang telah ada).
4. **Sesi Sisi Server dengan Hash SHA-256 (ADR-007)**: Token sesi acak 32-byte dikirimkan ke klien melalui cookie `HttpOnly`, `SameSite=Lax`. Di basis data, hanya hash SHA-256 token yang disimpan, memastikan kebocoran database tidak mengekspos token sesi aktif.
5. **Titik Tunggal Pembuatan Sesi & Row Locking (ADR-008 & ADR-010)**: Penegakan batas maksimal 2 sesi aktif hanya dilakukan di satu fungsi (`auth.Service.CreateSession`) di dalam transaksi database yang mengunci baris pengguna dengan `SELECT ... FOR UPDATE`. Login ketiga mengusir sesi paling lama (`revoked_reason = 'evicted'`).
6. **Mitigasi Anti-Enumerasi (ADR-009)**: Menyamakan waktu respons pada endpoint login saat email/username tidak ditemukan dengan mengeksekusi perbandingan hash tiruan, sehingga mencegah penyerang membedakan status akun terdaftar.

---

## 5. Penggunaan AI

Pengembangan proyek ini memanfaatkan bantuan perangkat AI dengan pembagian peran yang transparan:
- **Claude AI**: Digunakan untuk tahap perencanaan arsitektur, pembedahan kebutuhan fungsional (FSD), perancangan skema database, dan penyusunan strategi pembagian fase kerja.
- **Antigravity (Google DeepMind)**: Digunakan sebagai *pair programming agent* untuk scaffolding struktur repositori, implementasi kode Go (backend), komponen React & TypeScript (frontend), konfigurasi Docker multi-stage, serta pengujian integrasi dan *code review*.

---

## 6. Jawaban Catatan Implementasi

Berikut adalah jawaban atas 5 butir pertanyaan teknis implementasi pada bagian 16 dokumen spesifikasi (FSD):

### 1. Penyimpanan Kata Sandi
Penyimpanan kata sandi diimplementasikan pada fungsi `HashPassword` dan diverifikasi melalui `VerifyPassword` di berkas [`backend/internal/auth/password.go`](file:///c:/Programming/fti/fti-auth-system/backend/internal/auth/password.go) menggunakan algoritma Argon2id (`golang.org/x/crypto/argon2`) dengan format standar PHC string (`$argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>`). Algoritma ini dipilih karena memiliki ketahanan superior terhadap serangan perangkat keras berkecepatan tinggi (GPU/ASIC) melalui kebutuhan memori (*memory-hardness*), serta memungkinkan penyimpanan salt acak 16-byte unik per pengguna dan parameter komputasi secara mandiri dalam satu string hash tanpa memerlukan tabel terpisah.

### 2. Definisi Sesi dan Penegakan Batas Sesi
Setiap sesi login didefinisikan sebagai satu baris unik pada tabel `sessions` yang menyimpan hash SHA-256 token, ID pengguna, agen peramban (*User-Agent*), alamat IP, dan masa berlaku (*expires_at*). Batas maksimal 2 sesi aktif ditegakkan secara eksklusif hanya di satu tempat, yaitu fungsi `CreateSession` di berkas [`backend/internal/auth/service.go`](file:///c:/Programming/fti/fti-auth-system/backend/internal/auth/service.go), di dalam satu transaksi database terisolasi yang mengunci baris pengguna target menggunakan `SELECT id FROM users WHERE id = $1 FOR UPDATE`. Jika jumlah sesi aktif telah mencapai atau melebihi batas, fungsi akan mencabut sesi dengan `created_at` tertua (`revoked_reason = 'evicted'`) hingga tersisa maksimal 1 sesi sebelum sesi baru disisipkan.

### 3. Mitigasi Risiko Token/Sesi Dicuri
Jika raw token dicuri dari peramban, penyerang secara teknis dapat menggunakan token tersebut untuk mengakses API hingga sesi kedaluwarsa atau dicabut, karena sistem belum dilengkapi mekanisme deteksi anomali (seperti *fingerprint mismatch* otomatis atau *IP binding* ketat). Namun demikian, risiko ini diminimalkan karena token disalurkan melalui cookie dengan flag `HttpOnly` sehingga tidak dapat diakses skrip XSS peramban, dan basis data hanya menyimpan representasi hash SHA-256 sehingga pencurian basis data tidak membocorkan token aktif; pengguna juga dapat sewaktu-waktu meminta admin mencabut seluruh sesi melalui fitur force logout.

### 4. Dua Kekurangan Sistem Saat Ini
TODO

### 5. Rencana Penambahan Fitur "Keluar dari Semua Perangkat"
Untuk menambahkan fitur "Keluar dari Semua Perangkat" oleh pengguna mandiri, endpoint baru `POST /api/me/sessions/revoke-all` akan ditambahkan pada routing pengguna di [`backend/cmd/server/main.go`](file:///c:/Programming/fti/fti-auth-system/backend/cmd/server/main.go) dan ditangani oleh handler baru di [`backend/internal/handlers/auth.go`](file:///c:/Programming/fti/fti-auth-system/backend/internal/handlers/auth.go). Handler tersebut akan mengekstrak ID pengguna terautentikasi dari context lalu memanggil fungsi layanan di [`backend/internal/auth/service.go`](file:///c:/Programming/fti/fti-auth-system/backend/internal/auth/service.go) yang mengeksekusi query pencabutan massal yang serupa dengan fungsi `ForceLogout` di [`backend/internal/auth/admin.go`](file:///c:/Programming/fti/fti-auth-system/backend/internal/auth/admin.go), yakni `UPDATE sessions SET revoked_at = now(), revoked_reason = 'user_revoke_all' WHERE user_id = $1 AND revoked_at IS NULL`, dengan opsi mempertahankan sesi saat ini jika menyertakan parameter pengecualian session ID aktif.