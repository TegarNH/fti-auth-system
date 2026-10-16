# fti-auth-system

Sistem autentikasi internal berbasis web untuk PT Kuliner Kreasindo Inovasi. Dibangun dengan **Go** di backend, **PostgreSQL 18** sebagai database, **React + TypeScript** di frontend yang di-serve lewat Nginx, dan semuanya diorkestrasi pakai **Docker Compose**.

Proyek ini merupakan bagian dari asesmen Fullstack Developer. Saya mengutamakan kebenaran implementasi dan kesederhanaan kode di atas segalanya, terutama pada bagian yang berkaitan langsung dengan keamanan sesi.

---

## Fitur

### Yang Diminta (Kebutuhan Minimum)

- **Register & Login** — daftar dengan email, username, dan password; masuk bisa pakai email *atau* username lewat satu kolom identifier.
- **Profil Terautentikasi** — `GET /api/me` hanya bisa diakses dengan sesi aktif dan selalu mengembalikan data pemilik sesi itu sendiri.
- **Batas 2 Sesi Aktif (FIFO)** — setiap akun maksimal 2 sesi aktif bersamaan. Login ketiga otomatis mencabut sesi paling lama. Implementasinya ada di satu fungsi saja (`CreateSession`) di dalam transaksi dengan row-level lock, jadi tidak bisa di-bypass lewat login paralel sekalipun.
- **Manajemen Pengguna untuk Admin** — halaman `/admin/users` menampilkan daftar semua user, peran, dan jumlah sesi aktif saat ini.
- **Force Logout oleh Admin** — admin bisa langsung cabut semua sesi aktif milik user lain dari halaman admin.
- **Route Guard & Middleware** — proteksi berbasis peran di level API (backend) dan navigasi (frontend). Kalau sesi dicabut di tengah jalan, frontend otomatis redirect ke `/login` dengan pesan yang jelas.
- **Auto-seed Admin Pertama** — akun admin langsung tersedia begitu `docker compose up` selesai. Prosesnya idempoten, jadi restart tidak akan membuat duplikat atau menimpa password yang sudah diubah.

### Fitur Tambahan (Di Luar Minimum)

- **Daftar & Cabut Sesi Sendiri** — user bisa lihat semua perangkat yang sedang login ke akunnya (User-Agent, IP, waktu login, aktivitas terakhir, dan penanda "sesi ini") dan mencabut salah satunya langsung dari halaman profil, tanpa perlu minta bantuan admin. Endpoint: `GET /api/me/sessions` dan `DELETE /api/me/sessions/{id}`.
- **Anti-Enumerasi dengan Dummy Hash** — saat login dengan email/username yang tidak terdaftar, backend tetap menjalankan komputasi Argon2id tiruan supaya waktu respons tidak berbeda jauh dengan login yang gagal karena password salah. Ini mencegah penyerang menebak akun mana yang terdaftar.
- **Row-Level Locking untuk Login Paralel** — `SELECT ... FOR UPDATE` pada baris user memastikan batas 2 sesi tidak pernah tembus walau ada dua login yang masuk dalam waktu bersamaan.
- **UI Dark Mode Glassmorphism** — antarmuka responsif bergaya modern tanpa framework UI berat. Bundle tetap ringan.

---

## Cara Menjalankan

Cukup empat perintah dari kondisi repo bersih:

```bash
cp .env.example .env
docker compose up -d --build
# Tunggu sebentar sampai semua container healthy
# Frontend: http://localhost:3000
# Backend:  http://localhost:8080/health
```

### Login sebagai Admin

Kredensial default ada di `.env.example`:

| Field | Nilai |
|---|---|
| Email | `admin@fti.com` |
| Username | `admin` |
| Password | `FtiKki123!` |


---

## Endpoint API

Semua request dan response menggunakan `application/json`. Cookie sesi `auth_session` dikirim otomatis oleh browser (`credentials: 'include'`).

| Metode | Path | Akses | Deskripsi |
|:---|:---|:---|:---|
| `GET` | `/health` | Publik | Cek kesehatan server dan koneksi DB |
| `POST` | `/api/auth/register` | Publik | Daftar akun baru (peran selalu `user`) |
| `POST` | `/api/auth/login` | Publik | Login dengan `identifier` (email/username) + `password` |
| `POST` | `/api/auth/logout` | Login | Cabut sesi saat ini, hapus cookie |
| `GET` | `/api/me` | Login | Data profil pemilik sesi |
| `GET` | `/api/me/sessions` | Login | Daftar sesi aktif milik sendiri |
| `DELETE` | `/api/me/sessions/{id}` | Login | Cabut salah satu sesi milik sendiri |
| `GET` | `/api/admin/users` | Admin | Daftar semua user + jumlah sesi aktif |
| `POST` | `/api/admin/users/{id}/force-logout` | Admin | Cabut semua sesi aktif milik user target |

---

## Keputusan Teknis

Beberapa keputusan teknis antara lain yaitu:

- **Multi-stage Docker build** — backend Go dikompilasi ke binary statis (`CGO_ENABLED=0`) lalu disalin ke `alpine` minimal. Frontend di-build dengan Node lalu disajikan Nginx. Image hasil akhirnya kecil dan tidak ada toolchain di runtime.
- **Volume PostgreSQL 18** — mount ke `/var/lib/postgresql` (bukan `/data`) karena PostgreSQL 18 alpine menaruh cluster di path tersebut. Ini salah satu hal yang tidak obvious dan sempat bikin container gagal start.
- **Migrasi ter-embed & idempoten** — file SQL di-embed langsung ke binary via `embed.FS` dan dijalankan lewat `golang-migrate` saat start. Tidak ada langkah migrasi manual.
- **Hash SHA-256 di database** — raw token tidak pernah masuk database. Yang disimpan hanya hash SHA-256-nya. Kalau database bocor, token aktif tetap aman.
- **Satu titik pembuatan sesi** — semua pembuatan sesi, termasuk pengecekan batas dan eviction, ada di satu fungsi `auth.Service.CreateSession` dalam satu transaksi. Tidak ada celah untuk membuat sesi dari tempat lain.

---

## Penggunaan AI

Proyek ini dikerjakan dengan bantuan AI secara terbuka:

- **Claude AI** — dipakai di awal untuk memahami Soal, merancang arsitektur, dan merencanakan task.
- **Antigravity IDE (Agent Gemini 3.8 Flash & Claude Sonnet 4.6)** — dipakai sebagai *pair programming agent* sepanjang pengerjaan: scaffolding struktur repo, implementasi kode Go dan React/TypeScript, konfigurasi Docker, penulisan integration test, dan review kode.

---

## Catatan Implementasi

Jawaban atas 5 butir pertanyaan di bagian 16 FSD:

### 1. Penyimpanan Kata Sandi

Kata sandi di-hash menggunakan fungsi `HashPassword` dan diverifikasi lewat `VerifyPassword` di [`backend/internal/auth/password.go`](backend/internal/auth/password.go). Algoritmanya Argon2id dari `golang.org/x/crypto/argon2` dengan format PHC string (`$argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>`). Saya pilih Argon2id karena ini rekomendasi OWASP saat ini untuk hashing password — memory-hard sehingga tahan terhadap serangan GPU/ASIC, dan salt 16-byte unik per user sudah ter-embed di string hash itu sendiri tanpa perlu kolom terpisah di database.

### 2. Definisi Sesi dan Penegakan Batas

Setiap sesi adalah satu baris di tabel `sessions` yang menyimpan hash token, `user_id`, User-Agent, IP address, dan timestamp. Sesi dianggap aktif jika `revoked_at IS NULL` dan `expires_at > now()`. Batas 2 sesi aktif ditegakkan **hanya** di fungsi `CreateSession` di [`backend/internal/auth/service.go`](backend/internal/auth/service.go), di dalam satu transaksi yang membuka row lock dengan `SELECT id FROM users WHERE id = $1 FOR UPDATE`. Kalau sudah ada 2 sesi aktif, yang paling lama (`created_at` terkecil) langsung dicabut dengan `revoked_reason = 'evicted'` sebelum sesi baru diinsert.

### 3. Risiko Pencurian Token

Kalau raw token berhasil dicuri dari browser (misalnya lewat network sniffing atau kebocoran log), penyerang bisa menggunakannya sampai sesi kedaluwarsa atau dicabut, karena sistem ini belum punya mekanisme deteksi anomali otomatis seperti IP binding ketat atau device fingerprinting. Risikonya sudah diminimalisir dengan beberapa lapisan: cookie `HttpOnly` mencegah akses lewat JavaScript (XSS), database hanya menyimpan hash SHA-256 jadi kebocoran DB tidak otomatis mengekspos token, masa berlaku sesi 7 hari (bisa diatur via env), dan pengguna bisa mencabut sesi secara mandiri atau minta admin force logout.

### 4. Dua Kekurangan Sistem Saat Ini

1. Tidak ada rate limiting pada endpoint login, sistem rentan terhadap
   brute-force attack tanpa hambatan apapun.

2. Tidak ada sliding expiry pada sesi, last_seen_at diperbarui tapi tidak
   mempengaruhi expires_at, sehingga user aktif pun bisa tiba-tiba ter-logout
   tanpa peringatan setelah 7 hari sejak login pertama.


### 5. Rencana Penambahan "Keluar dari Semua Perangkat"

Untuk menambah fitur ini, endpoint `POST /api/me/sessions/revoke-all` perlu ditambahkan ke routing di [`backend/cmd/server/main.go`](backend/cmd/server/main.go) dan ditangani oleh handler baru di [`backend/internal/handlers/auth.go`](backend/internal/handlers/auth.go). Handler tersebut akan mengambil `userID` dari context (yang sudah diisi middleware `RequireAuth`), lalu memanggil method service baru di [`backend/internal/auth/service.go`](backend/internal/auth/service.go) yang menjalankan query serupa dengan `ForceLogout` di [`backend/internal/auth/admin.go`](backend/internal/auth/admin.go): `UPDATE sessions SET revoked_at = now(), revoked_reason = 'user_revoke_all' WHERE user_id = $1 AND revoked_at IS NULL`. Kalau ingin mempertahankan sesi yang sedang aktif (biar user tidak langsung ter-logout dari perangkat yang sedang dipakai), bisa tambahkan `AND id != $2` dengan ID sesi saat ini sebagai parameter kedua.