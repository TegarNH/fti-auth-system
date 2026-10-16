# FSD — Sistem Authentication Internal Tim FTI

*Functional Specification Document ·  15 Oktober 2026 · Untuk Asesmen Fullstack Developer, PT Kuliner Kreasindo Inovasi*

## 1. Ringkasan

Aplikasi web internal untuk tim kecil dengan sistem authentication sebagai fondasinya: pendaftaran, login (email atau username), logout, profil, dua peran (user dan admin), batas 2 sesi aktif per akun, dan force logout oleh admin. Backend Go, basis data PostgreSQL, frontend bebas, seluruhnya berjalan lewat Docker Compose.

**Prinsip desain:** sesi disimpan di sisi server (bukan JWT stateless) karena dua ketentuan wajib, yaitu batas 2 sesi dan force logout, mengharuskan sesi bisa dicabut seketika.

## 2. Ruang Lingkup

**Wajib (Must):** FR-01 sampai FR-08, aturan sesi, deployment Docker Compose, README.

**Sebaiknya ada (Should):** halaman daftar sesi milik sendiri, pencabutan sesi sendiri, audit log, pembatasan percobaan login (rate limit), ubah password.

**Opsional (Nice):** modul domain (opsi B atau C), nonaktifkan akun, ubah peran oleh admin, keluar dari semua perangkat.

**Di luar lingkup:** verifikasi email, reset password lewat email, 2FA, OAuth.

## 3. Aktor dan Peran

| Aktor | Hak akses |
| --- | --- |
| Tamu (belum login) | Daftar, login |
| User | Melihat profil sendiri, logout, melihat dan mencabut sesi sendiri |
| Admin | Semua hak user, ditambah melihat daftar seluruh pengguna dan force logout pengguna lain |

Peran selalu ditentukan server. Pendaftaran secara otomatis menghasilkan peran `user`.

## 4. Kebutuhan Fungsional

| ID | Kebutuhan | Prioritas |
| --- | --- | --- |
| FR-01 | Pendaftaran dengan email, username, dan kata sandi | Must |
| FR-02 | Login dengan email **atau** username beserta kata sandi | Must |
| FR-03 | Logout mencabut sesi yang sedang dipakai | Must |
| FR-04 | Profil hanya bisa diakses setelah login dan hanya menampilkan data pemilik sesi | Must |
| FR-05 | Dua peran (user, admin); daftar seluruh pengguna hanya untuk admin | Must |
| FR-06 | Akun admin pertama tersedia otomatis setelah `docker compose up` | Must |
| FR-07 | Maksimal 2 sesi aktif per akun; login ketiga mengakhiri sesi paling lama | Must |
| FR-08 | Admin dapat force logout pengguna lain; seluruh sesinya langsung tidak berlaku | Must |
| FR-09 | User melihat daftar sesi miliknya (perangkat, waktu login, aktivitas terakhir) dan dapat mencabutnya | Should |
| FR-10 | Audit log: login berhasil/gagal, logout, sesi tergusur, force logout | Should |
| FR-11 | Rate limit percobaan login gagal | Should |
| FR-12 | Ubah kata sandi (mencabut sesi lain) | Should |
| FR-13 | Admin dapat menonaktifkan akun dan mengubah peran | Nice |

### Aturan validasi

- **Email:** format valid, disimpan huruf kecil, unik.
- **Username:** 3 sampai 30 karakter, hanya huruf, angka, titik, garis bawah; unik tanpa membedakan huruf besar/kecil.
- **Kata sandi:** minimal 8 karakter (disarankan 10), maksimal 72 byte atau 128 karakter sesuai algoritma yang dipilih.
- **Pesan error login** selalu generik ("email/username atau kata sandi salah") agar tidak membocorkan akun mana yang ada.

## 5. Aturan Bisnis Sesi

**Definisi sesi:** satu baris di tabel `sessions`, dibuat setiap login berhasil, mewakili satu perangkat/browser. Sesi dianggap **aktif** jika`revoked_at IS NULL` dan `expires_at > now()`.

**Alur login (satu transaksi):**

1. Cari user berdasarkan email atau username, verifikasi kata sandi.
2. Kunci baris user dengan `SELECT ... FOR UPDATE` agar dua login bersamaan tidak melewati batas.
3. Hitung sesi aktif. Jika sudah 2, cabut sesi dengan `created_at` paling lama hingga tersisa 1.
4. Buat sesi baru dan kembalikan token.

**Token:** 32 byte acak dari `crypto/rand`, dikirim ke klien sekali dalam cookie `HttpOnly`. Di database hanya disimpan **hash SHA-256**-nya, sehingga kebocoran database tidak membocorkan token yang bisa dipakai.

**Masa berlaku:** 7 hari sejak dibuat (dapat diatur lewat env). Kolom `last_seen_at` diperbarui saat sesi dipakai.

**Force logout:** `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`.&#32;

## 6. Arsitektur dan Teknologi

| Lapisan | Pilihan yang disarankan | Alasan |
| --- | --- | --- |
| Backend | Go + `chi` (router) + `pgx` | Ringan |
| Basis data | PostgreSQL 18 | Sesuai requirement, last version |
| Migrasi | `golang-migrate` dengan `embed`, dijalankan saat backend start | Otomatis, tanpa langkah manual |
| Hash kata sandi | `argon2id` (parameter bisa diatur lewat env) | Standar modern |
| Frontend | React + Vite (atau Vue/Svelte), di-serve Nginx pada port 3000 | Build statis, container kecil |
| Orkestrasi | Docker Compose: `db`, `backend`, `frontend` | Wajib |

**Struktur backend yang disarankan:**

```
backend/
  cmd/server/main.go
  internal/
    config/        # baca env
    db/            # koneksi + migrations/*.sql
    auth/          # hash, token, service (register, login, logout, batas sesi)
    middleware/    # RequireAuth, RequireAdmin, rate limit, CORS
    handlers/      # HTTP handler
    repository/    # query SQL
```

Batas 2 sesi ditempatkan di **satu fungsi** (mis. `auth.Service.CreateSession`).

**Catatan CORS dan cookie:** frontend (`localhost:3000`) dan backend (`localhost:8080`) berbeda origin tetapi sama site. Gunakan `SameSite=Lax`, `HttpOnly`, CORS dengan `Access-Control-Allow-Credentials: true` dan origin spesifik (bukan `*`), serta `credentials: 'include'` di klien. Atur flag `Secure` lewat env (aktif saat memakai HTTPS).

## 7. Model Data

**users**

| Kolom | Tipe | Catatan |
| --- | --- | --- |
| id | UUID, PK | `gen_random_uuid()` |
| email | text | unik (huruf kecil) |
| username | text | unik tanpa membedakan huruf besar/kecil |
| password\_hash | text | argon2id, termasuk salt dan parameter |
| role | text | `user` atau `admin`, dengan CHECK constraint |
| is\_active | boolean | default true |
| created\_at, updated\_at | timestamptz |  |

**sessions**

| Kolom | Tipe | Catatan |
| --- | --- | --- |
| id | UUID, PK |  |
| user\_id | UUID, FK ke users | ON DELETE CASCADE, diindeks |
| token\_hash | bytea | unik, diindeks |
| user\_agent | text | untuk tampilan "perangkat" |
| ip\_address | inet |  |
| created\_at | timestamptz | dasar penentuan sesi paling lama |
| last\_seen\_at | timestamptz |  |
| expires\_at | timestamptz |  |
| revoked\_at | timestamptz null | null berarti belum dicabut |
| revoked\_reason | text null | `logout`, `evicted`, `force_logout`, `password_changed` |

**audit\_logs** (Should): `id`, `user_id` (nullable), `action`, `ip_address`, `user_agent`, `metadata` (jsonb), `created_at`.

## 8. Daftar Endpoint

| Metode | Path | Akses | Fungsi |
| --- | --- | --- | --- |
| GET | `/health` | Publik | Cek kesehatan layanan |
| POST | `/api/auth/register` | Publik | Daftar (email, username, password) |
| POST | `/api/auth/login` | Publik | Login (`identifier`, `password`), mengatur cookie sesi |
| POST | `/api/auth/logout` | Login | Cabut sesi saat ini |
| GET | `/api/me` | Login | Data profil pemilik sesi |
| GET | `/api/me/sessions` | Login | Daftar sesi aktif milik sendiri |
| DELETE | `/api/me/sessions/{id}` | Login | Cabut salah satu sesi milik sendiri |
| POST | `/api/me/password` | Login | Ubah kata sandi |
| GET | `/api/admin/users` | Admin | Daftar seluruh pengguna (dengan jumlah sesi aktif) |
| POST | `/api/admin/users/{id}/force-logout` | Admin | Cabut semua sesi pengguna |
| GET | `/api/admin/audit-logs` | Admin | Riwayat audit (Should) |

**Kode status:** 400 validasi gagal, 401 belum login atau sesi tidak valid, 403 peran tidak cukup, 409 email/username sudah dipakai, 429 terlalu banyak percobaan.

## 9. Halaman Frontend

| Halaman | Isi |
| --- | --- |
| `/register` | Form daftar dengan validasi sisi klien dan pesan error dari server |
| `/login` | Satu kolom "email atau username" + kata sandi |
| `/profile` | Data pengguna, daftar perangkat/sesi (Should), tombol logout |
| `/admin/users` | Tabel pengguna, peran, jumlah sesi, tombol **Force logout** (hanya admin) |

Route guard di klien hanya untuk pengalaman pengguna; **penegakan hak akses tetap di backend**. Jika sesi dicabut (401), klien mengarahkan ke `/login` dengan pesan yang jelas.

## 10. Keamanan

- Kata sandi: argon2id dengan salt unik; tidak pernah dicatat di log.
- Login anti-enumerasi: pesan generik, dan tetap menjalankan perbandingan hash tiruan saat user tidak ditemukan agar waktu respons mirip.
- Rate limit login per kombinasi IP dan identifier (contoh: 5 gagal per 15 menit).
- Cookie `HttpOnly`, `SameSite=Lax`; CORS dibatasi ke origin frontend.
- Semua query memakai parameter (tanpa konkatenasi string).
- Pembatasan ukuran body request dan timeout server.
- Admin tidak boleh force logout dirinya sendiri tanpa konfirmasi, dan sebaiknya ada pemeriksaan agar admin terakhir tidak bisa dinonaktifkan.

## 11. Deployment

**Compose:** tiga layanan.

- `db`: `postgres:1``8``-alpine`, volume permanen, `healthcheck` dengan `pg_isready`.
- `backend`: build dari `backend/`, `depends_on` dengan `condition: service_healthy`, port 8080, menjalankan migrasi lalu seed admin saat start.
- `frontend`: build multi-stage (build lalu Nginx), port 3000.

**`.env.example`** (wajib di root, bersama `docker-compose.yml`):

```
POSTGRES_USER=app
POSTGRES_PASSWORD=change_me
POSTGRES_DB=authdb
DATABASE_URL=postgres://app:change_me@db:5432/authdb?sslmode=disable
BACKEND_PORT=8080
FRONTEND_ORIGIN=http://localhost:3000
COOKIE_SECURE=false
SESSION_TTL_HOURS=168
MAX_ACTIVE_SESSIONS=2
ADMIN_EMAIL=admin@fti.com
ADMIN_USERNAME=admin
ADMIN_PASSWORD=FtiKki123!
```

**Seed admin pertama:** saat backend start, jika belum ada user dengan `ADMIN_EMAIL`, buat akun admin dari env (idempoten; tidak menimpa kata sandi akun yang sudah ada). Cantumkan kredensial default ini di README beserta peringatan untuk menggantinya.

**Kriteria berhasil:** setelah empat perintah dari soal, `http://localhost:3000` terbuka, `http://localhost:8080/health` merespons, dan login sebagai admin langsung bisa dilakukan.

## 12. Breakdown Tugas

Estimasi kasar total sekitar 14 sampai 16 jam kerja.

| # | Tugas | Estimasi | Hasil |
| --- | --- | --- | --- |
| 1 | Inisiasi repo, struktur folder, Dockerfile backend dan frontend, `docker-compose.yml`, `.env.example` | 1,5 jam | Tiga container hidup dan saling terhubung |
| 2 | Migrasi SQL (users, sessions, audit\_logs), koneksi DB, migrasi otomatis | 1 jam | Skema terbentuk otomatis saat start |
| 3 | Register, login, logout, hashing, pembuatan token, middleware `RequireAuth` | 3 jam | FR-01 sampai FR-04 |
| 4 | Batas 2 sesi dengan transaksi, force logout, middleware `RequireAdmin` | 2 jam | FR-05, FR-07, FR-08 |
| 5 | Seed admin dari env | 0,5 jam | FR-06 |
| 6 | Frontend: register, login, profil, admin users, penanganan 401 | 3 jam | Alur lengkap bisa dipakai |
| 7 | Fitur tambahan: daftar sesi sendiri, audit log, rate limit, ubah password | 2 jam | FR-09 sampai FR-12 |
| 8 | Pengujian: unit test batas sesi dan hash, uji manual dua browser, uji dari repo bersih | 1,5 jam | Bukti kebutuhan wajib terpenuhi |
| 9 | README lengkap dan 5 jawaban Catatan Implementasi | 1,5 jam | Dokumen siap kumpul |

## 13. Rencana Pengujian

| Skenario | Hasil yang diharapkan |
| --- | --- |
| Daftar dengan email/username yang sudah dipakai | 409 |
| Login dengan email, lalu dengan username | Keduanya berhasil |
| Akses `/api/me` tanpa login | 401 |
| User biasa membuka `/api/admin/users` | 403 |
| Login di 3 browser berbeda dengan akun sama | Browser pertama mendapat 401 setelah login ketiga; browser kedua dan ketiga tetap aktif |
| 2 login bersamaan secara paralel saat sudah ada 1 sesi | Tidak pernah lebih dari 2 sesi aktif |
| Admin force logout seorang user yang sedang login di 2 perangkat | Kedua perangkat langsung mendapat 401 pada request berikutnya |
| `docker compose down -v` lalu `up -d --build` dari clone baru | Aplikasi dan admin pertama siap tanpa langkah tambahan |
