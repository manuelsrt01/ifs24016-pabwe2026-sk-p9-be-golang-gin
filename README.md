# Delcom Blog — Backend REST API (Golang Gin, GORM, PostgreSQL 17 & AI Search Engine)

Repositori Backend Studi Kasus Praktikum PABWE 2026 Modul P9: **Delcom Blog — Platform Publikasi Artikel Multi-Tier dengan AI Content Generator dan Integrasi Search Engine**.

Dibangun menggunakan **Golang 1.27.4**, **Gin Web Framework**, **GORM ORM**, **PostgreSQL 17**, **JWT Authentication**, **Swaggo OpenAPI/Swagger UI**, serta arsitektur **Clean Architecture & Feature-First** dengan **100% Statement Coverage (TDD)**.

---

## 1. Persyaratan Perangkat Lunak & Versi Teknologi

- **Golang**: versi `1.27.4`
- **Gin Web Framework**: `v1.12.0`
- **GORM ORM**: `v1.31.2` + `gorm.io/driver/postgres v1.6.3` + `github.com/glebarez/sqlite v1.11.0`
- **PostgreSQL**: versi `17`
- **JWT**: `github.com/golang-jwt/jwt/v5 v5.3.1`
- **Bcrypt Crypto**: `golang.org/x/crypto v0.59.0`
- **Environment Loader**: `github.com/joho/godotenv v1.5.1`
- **Swagger / OpenAPI**: `github.com/swaggo/gin-swagger v1.6.1`, `files v1.0.1`, `swag v1.16.6`
- **Unit Testing**: `go test` + `github.com/stretchr/testify v1.12.1`

---

## 2. Struktur Direktori Backend

```text
ifs24016-pabwe2026-sk-p9-be-golang-gin/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point server Gin (Port 8081)
├── docs/                           # Spesifikasi Swagger / OpenAPI 2.0
│   ├── api.json
│   ├── docs.go
│   ├── docs_test.go
│   ├── swagger.json
│   └── swagger.yaml
├── internal/
│   ├── config/                     # Loader konfigurasi .env
│   │   ├── config.go
│   │   └── config_test.go
│   ├── database/                   # Koneksi PostgreSQL/SQLite, AutoMigrate, Seeding
│   │   ├── database.go
│   │   └── database_test.go
│   ├── middleware/                 # CORS, JWT Auth, RequireRole, RateLimiter
│   │   ├── middleware.go
│   │   └── middleware_test.go
│   ├── response/                   # Format JSON respons standar {status, message, data}
│   │   ├── response.go
│   │   └── response_test.go
│   ├── features/
│   │   ├── auth/                   # Registrasi, Login JWT, Logout, Profil
│   │   │   ├── dto.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   ├── handler.go
│   │   │   └── auth_test.go
│   │   ├── post/                   # CRUD Artikel, Unique Slug, Reading Time, Stats, Admin
│   │   │   ├── dto.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   ├── handler.go
│   │   │   └── post_test.go
│   │   └── ai/                     # Integrasi Search Engine & OpenAI Content Generator
│   │       ├── client.go
│   │       ├── search.go
│   │       ├── dto.go
│   │       ├── service.go
│   │       ├── handler.go
│   │       └── ai_test.go
│   └── server/                     # Pendaftaran rute Gin & Swagger UI
│       ├── router.go
│       └── router_test.go
├── .env.example
├── .gitignore
├── Jenkinsfile
├── go.mod
├── go.sum
└── README.md
```

---

## 3. Setup Database PostgreSQL & Konfigurasi `.env`

1. Buat database PostgreSQL bernama `pabwe_p9_blog`:
   ```bash
   psql -U postgres -c "CREATE DATABASE pabwe_p9_blog;"
   ```

2. Salin berkas `.env.example` menjadi `.env` dan sesuaikan nilainya:
   ```dotenv
   APP_ENV=development
   APP_PORT=8081
   APP_KEY=delcom_blog_secret_key_super_secure_2026_random_string_32chars

   ADMIN_USERNAME=admin
   ADMIN_PASSWORD=admin

   DB_CONNECTION=pgsql
   DB_HOST=127.0.0.1
   DB_PORT=5432
   DB_DATABASE=pabwe_p9_blog
   DB_USERNAME=postgres
   DB_PASSWORD=postgres
   DB_SSLMODE=disable

   SEARCH_ENGINE_ENDPOINT=https://delcom.org/api/v1/search
   OPENAI_ENDPOINT=https://delcom.org/api/v1
   OPENAI_KEY=isi_api_key_anda
   OPENAI_MODEL=qwen3.8-flash
   ```

3. Saat server pertama kali dijalankan, GORM `AutoMigrate()` secara otomatis membuat tabel `users` dan `posts`, serta menyuntikkan akun bawaan (seeder):
   - **Admin**: username `admin` / password `admin` (`role: admin`)
   - **Penulis (Member)**: username `member` / password `member` (`role: member`)

---

## 4. Daftar Lengkap Endpoint REST API (21 Endpoint)

| Method | Endpoint | Hak Akses | Keterangan & Fungsi |
| :--- | :--- | :--- | :--- |
| `GET` | `/swagger/*any` | Publik | Dokumentasi Interaktif Swagger UI (`http://localhost:8081/swagger/index.html`) |
| `GET` | `/api.json` | Publik | Spesifikasi OpenAPI / Swagger JSON dinamis |
| `GET` | `/api/public/categories` | Publik | Mengambil daftar seluruh kategori artikel blog (`Teknologi`, `Pendidikan`, `Gaya Hidup`, `Bisnis`, `Opini`, `Umum`) |
| `GET` | `/api/public/posts` | Publik | Mengambil daftar artikel terbit (`is_published = true`) dengan pencarian (`?q=...`) dan filter kategori (`?category=...`) |
| `GET` | `/api/public/posts/:slug` | Publik | Mengambil detail artikel publik terbit berdasarkan slug beserta penulis dan `reading_time` |
| `POST` | `/api/auth/register` | Publik | Pendaftaran akun penulis (`author`) baru |
| `POST` | `/api/auth/login` | Publik | Autentikasi kredensial pengguna dan penerbitan token JWT |
| `POST` | `/api/auth/logout` | Authenticated | Logout sesi pengguna |
| `GET` | `/api/auth/profile` | Authenticated | Mengambil data profil penulis yang sedang login |
| `PUT` | `/api/auth/profile` | Authenticated | Memperbarui informasi profil penulis yang sedang login |
| `GET` | `/api/posts` | Penulis | Mengambil seluruh daftar artikel milik penulis yang sedang login |
| `POST` | `/api/posts` | Penulis | Membuat artikel baru (slug unik & kalkulasi waktu baca otomatis) |
| `GET` | `/api/posts/stats` | Penulis | Mengambil ringkasan 3 kartu statistik penulis (`total_artikel`, `total_terbit`, `belum_selesai`) |
| `GET` | `/api/posts/:id` | Penulis | Mengambil detail satu artikel milik penulis berdasarkan ID |
| `PUT` | `/api/posts/:id` | Penulis | Memperbarui artikel milik penulis |
| `PATCH` | `/api/posts/:id/toggle-publish` | Penulis | Mengubah status publikasi artikel secara instan (Draf ↔ Terbit) beserta `published_at` |
| `DELETE` | `/api/posts/:id` | Penulis | Menghapus artikel milik penulis |
| `POST` | `/api/posts/ai-generate` | Penulis | Menghasilkan draf artikel utuh menggunakan AI yang terintegrasi dengan Search Engine |
| `GET` | `/api/admin/users` | Admin Only | Mengambil data seluruh pengguna sistem beserta total artikel & total terbit |
| `GET` | `/api/admin/posts` | Admin Only | Monitoring arsip seluruh artikel dari semua penulis dalam sistem |
| `GET` | `/api/admin/users/:id/posts` | Admin Only | Mengambil seluruh artikel yang ditulis oleh pengguna tertentu |

---

## 5. Cara Menjalankan, Menguji & Kompilasi Binary

### Menjalankan Server Development
```bash
cd ifs24016-pabwe2026-sk-p9-be-golang-gin
go run cmd/server/main.go
```
- Server REST API berjalan di: `http://localhost:8081`
- Dokumentasi Swagger UI: `http://localhost:8081/swagger/index.html`

### Menjalankan Unit Testing & Verifikasi 100% Statement Coverage
```bash
go test -v -cover ./internal/...
```

### Kompilasi Standalone Binary Production
```bash
# Windows
go build -o server.exe cmd/server/main.go
.\server.exe

# Linux / macOS
go build -o server cmd/server/main.go
./server
```
