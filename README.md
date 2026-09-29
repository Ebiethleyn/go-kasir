# Go Kasir

Go Kasir adalah aplikasi point of sale (POS) berbasis web untuk operasional toko dan UMKM. Aplikasi mencakup katalog dan stok produk, pembayaran, pelanggan, laporan, ekspor, backup, dan pengaturan toko. Backend API dan frontend statis dilayani oleh satu server Go.

## Fitur

### Dashboard

- Ringkasan penjualan dan jumlah transaksi hari ini, produk terlaris, dan stok menipis.
- Grafik tren penjualan dan komposisi kategori untuk tujuh hari terakhir.
- Ringkasan transaksi dan produk untuk hari, minggu, bulan, kuartal, semester, dan tahun berjalan.
- Peringkat produk berdasarkan kuantitas, jumlah transaksi berbeda, omzet, dan harga satuan tertinggi.
- Keuntungan total sepanjang waktu dan per periode dihitung dari `(harga jual - harga beli) x kuantitas` pada item transaksi selesai. Nilai ini adalah margin kotor produk; diskon tingkat transaksi dan biaya operasional tidak dikurangkan.

### Kasir (POS)

- Katalog produk dengan pencarian nama/SKU/barcode dan dukungan lookup barcode.
- Pilih produk satu per satu atau centang beberapa produk lalu tambahkan semuanya ke keranjang.
- Ubah kuantitas, hapus item, kosongkan keranjang, pilih pelanggan, dan terapkan diskon.
- Pajak dapat diatur untuk transaksi; kolom POS mulai dari 0%.
- Metode pembayaran yang dicatat: tunai, kartu, dan QRIS. Pembayaran kartu/QRIS tidak terhubung ke payment gateway.
- Validasi pembayaran tunai dan penghitungan kembalian.
- Setelah pembayaran berhasil, pilih untuk mengunduh struk PDF atau melanjutkan tanpa mencetak.

### Produk dan stok

- Tambah, ubah, cari, dan hapus produk; kelola kategori.
- Simpan SKU, barcode, kategori, harga beli/jual, stok, stok minimum, satuan, dan gambar produk.
- Catat stok masuk, stok keluar, dan penyesuaian stok/opname.
- Impor daftar produk dari Excel dan lihat histori pergerakan stok.

### Pelanggan dan transaksi

- Kelola data pelanggan.
- Daftar pelanggan menampilkan jumlah transaksi selesai dan total nominal transaksi selesai; halaman memperbarui data secara berkala saat terbuka.
- Urutkan pelanggan berdasarkan jumlah transaksi terbanyak atau tersedikit.
- Lihat riwayat transaksi, filter daftar transaksi, batalkan transaksi dengan alasan, dan unduh kembali struk.
- Pembatalan transaksi mengembalikan stok.

### Laporan, ekspor, dan pengaturan

- Laporan penjualan, produk, laba rugi, transaksi, dan histori pembelian pelanggan.
- Unduh laporan dalam PDF atau Excel.
- Backup database ke dump SQL, restore dari file SQL, ekspor data master ke Excel, dan lihat activity log.
- Ubah profil owner, kata sandi, informasi toko, prefix invoice, pajak toko, stok minimum, dan lebar struk.
- Pilih mode terang atau gelap. Preferensi tema disimpan di browser dan berlaku lintas halaman.
- Struk PDF mendukung lebar thermal 58 mm dan 80 mm; ukuran kertas mengikuti panjang isi struk.

## Teknologi

| Bagian | Teknologi |
| --- | --- |
| Backend/API | Go 1.22+, `net/http`, chi |
| Data | MySQL 8, GORM |
| Autentikasi | JWT access/refresh token, bcrypt |
| Frontend | HTML, CSS, JavaScript, Bootstrap 5, Bootstrap Icons |
| Grafik | Chart.js |
| PDF | gofpdf |
| Excel | excelize |

Frontend tidak membutuhkan npm atau proses bundling. Bootstrap, Bootstrap Icons, dan Chart.js dimuat dari CDN, sehingga browser memerlukan akses ke CDN tersebut.

## Struktur Proyek

```text
go-kasir/
├── backend/
│   ├── cmd/api/main.go          # startup server, route, migrasi, dan seed
│   ├── internal/
│   │   ├── config/              # konfigurasi environment
│   │   ├── handler/             # HTTP handlers
│   │   ├── middleware/          # autentikasi dan middleware
│   │   ├── model/               # model database
│   │   ├── repository/          # query database
│   │   ├── service/             # aturan bisnis
│   │   └── pkg/                 # JWT, PDF, Excel, backup, response
│   ├── migrations/              # skema dan seed SQL opsional
│   ├── uploads/                 # file upload lokal
│   ├── .env.example
│   └── go.mod
├── frontend/
│   ├── index.html               # login
│   ├── dashboard.html
│   ├── pos.html
│   ├── products.html
│   ├── customers.html
│   ├── transactions.html
│   ├── reports.html
│   ├── backup.html
│   ├── settings.html
│   └── assets/                  # CSS dan JavaScript bersama
└── README.md
```

## Prasyarat

- Go 1.22 atau lebih baru.
- MySQL 8 yang berjalan dan dapat diakses oleh aplikasi.
- Hak akses MySQL untuk membuat database pada startup pertama.
- `mysqldump` dan `mysql` CLI hanya diperlukan untuk fitur backup/restore. Di Windows aplikasi mencoba lokasi umum Laragon, XAMPP, dan MySQL Server.
- Browser modern. Aplikasi menggunakan Bootstrap dan Chart.js dari CDN.

## Menjalankan Aplikasi

### 1. Siapkan konfigurasi

Jalankan dari direktori proyek:

```powershell
cd C:\golang\go-kasir\backend
Copy-Item .env.example .env
```

Di macOS/Linux:

```bash
cd go-kasir/backend
cp .env.example .env
```

Sesuaikan `.env` dengan konfigurasi MySQL. Variabel yang tersedia:

| Variabel | Default | Keterangan |
| --- | --- | --- |
| `APP_PORT` | `8080` | Port HTTP server |
| `APP_ENV` | `development` | Label environment aplikasi |
| `DB_HOST` | `127.0.0.1` | Host MySQL |
| `DB_PORT` | `3306` | Port MySQL |
| `DB_USER` | `root` | User MySQL |
| `DB_PASSWORD` | kosong | Kata sandi MySQL |
| `DB_NAME` | `kasir_db` | Nama database |
| `JWT_SECRET` | nilai contoh | Secret penandatangan token; ganti sebelum penggunaan nyata |
| `JWT_EXPIRE_HOURS` | `24` | Masa berlaku access token dalam jam |
| `JWT_REFRESH_EXPIRE_HOURS` | `168` | Masa berlaku refresh token dalam jam |
| `UPLOAD_DIR` | `./uploads` | Direktori file upload, relatif terhadap working directory |
| `MAX_UPLOAD_MB` | `5` | Batas ukuran upload dalam MB |

Jangan gunakan secret contoh atau kata sandi bawaan pada lingkungan produksi. File `.env` berisi konfigurasi privat dan tidak boleh dibagikan.

### 2. Unduh dependensi dan jalankan

Pastikan MySQL aktif, lalu dari direktori `backend`:

```bash
go mod tidy
go run ./cmd/api
```

Saat startup, server akan membuat database bila belum ada, menjalankan GORM AutoMigrate, mengisi data awal untuk tabel yang masih kosong, membuat direktori upload, lalu melayani API dan frontend.

Buka aplikasi:

- Login: <http://localhost:8080/> atau <http://localhost:8080/index.html>
- Dashboard: <http://localhost:8080/dashboard.html>
- Kasir: <http://localhost:8080/pos.html>
- API health check: <http://localhost:8080/api/health>

Frontend dilayani langsung oleh Go; tidak perlu menjalankan server frontend terpisah. Hentikan server dengan `Ctrl+C`. Setelah mengubah kode backend, jalankan ulang proses Go agar perubahan backend aktif. Perubahan HTML/CSS/JS dapat dilihat setelah browser dimuat ulang.

### 3. Login pertama

Jika tabel owner masih kosong, data awal yang dibuat adalah:

| Username | Password |
| --- | --- |
| `admin` | `admin123` |

Ganti password dan secret JWT sebelum aplikasi digunakan pada jaringan bersama atau untuk data nyata.

### Data awal

Seed hanya mengisi data ketika tabel terkait masih kosong. Seed bawaan mencakup owner, pengaturan toko, kategori, delapan produk contoh, dan dua pelanggan contoh. Seed tidak menghapus atau menimpa data yang sudah ada, dan tidak membuat transaksi contoh.

Migrasi SQL di `backend/migrations/` tersedia untuk kebutuhan manual. Untuk penggunaan normal, startup aplikasi menjalankan AutoMigrate.

## Alur Data dan Aturan Bisnis

- Harga beli dan harga jual disalin ke `transaction_items` pada saat transaksi dibuat. Laporan keuntungan memakai snapshot harga tersebut, bukan harga produk saat ini.
- Transaksi berstatus `completed` dihitung dalam ringkasan penjualan dan keuntungan. Transaksi berstatus `voided` dikecualikan.
- Keuntungan dashboard dihitung sebagai jumlah `(sell_price - buy_price) x quantity` dari item transaksi selesai. Rumus ini tidak mengurangi diskon transaksi secara keseluruhan, pajak, atau biaya operasional.
- Transaksi pelanggan hanya terhubung ke pelanggan jika pelanggan dipilih saat checkout. Checkout sebagai pelanggan umum menyimpan `customer_id` kosong.
- Jumlah dan nominal transaksi pada daftar pelanggan hanya menghitung transaksi selesai dan diperbarui berkala oleh halaman selama tetap terbuka.
- Void transaksi mengembalikan stok produk dan menyimpan alasan pembatalan.
- Pengaturan pajak toko tersimpan di halaman Pengaturan. Kolom pajak di POS dapat diedit per transaksi dan saat ini dimulai dari 0%.
- Metode pembayaran hanya dicatat oleh aplikasi. Tidak ada pemrosesan pembayaran eksternal.

## API

Semua route API menggunakan prefix `/api`. Respons JSON standar berbentuk:

```json
{
	"success": true,
	"message": "ok",
	"data": {}
}
```

Route privat memerlukan header `Authorization: Bearer <access_token>`. Login dan refresh token adalah route publik.

| Method | Route | Akses | Kegunaan |
| --- | --- | --- | --- |
| `GET` | `/api/health` | Publik | Pemeriksaan server |
| `POST` | `/api/auth/login` | Publik | Login dan memperoleh token |
| `POST` | `/api/auth/refresh` | Publik | Memperbarui token |
| `GET` | `/api/auth/profile` | Token | Membaca profil owner |
| `PUT` | `/api/auth/profile` | Token | Memperbarui profil owner |
| `POST` | `/api/auth/change-password` | Token | Mengganti kata sandi |
| `GET` | `/api/dashboard` | Token | Data dashboard dan ringkasan periode |
| `GET`, `POST` | `/api/categories` | Token | Daftar dan buat kategori |
| `PUT`, `DELETE` | `/api/categories/{id}` | Token | Ubah atau hapus kategori |
| `GET` | `/api/products` | Token | Daftar produk |
| `GET` | `/api/products/search` | Token | Cari produk |
| `GET` | `/api/products/lookup` | Token | Cari produk melalui barcode/SKU |
| `GET` | `/api/products/low-stock` | Token | Daftar stok menipis |
| `POST` | `/api/products` | Token | Buat produk |
| `POST` | `/api/products/import` | Token | Impor Excel |
| `GET`, `PUT`, `DELETE` | `/api/products/{id}` | Token | Baca, ubah, atau hapus produk |
| `POST` | `/api/products/{id}/stock` | Token | Catat perubahan stok |
| `POST` | `/api/products/{id}/image` | Token | Upload gambar produk |
| `GET` | `/api/stock-movements` | Token | Histori pergerakan stok |
| `GET`, `POST` | `/api/customers` | Token | Daftar dan buat pelanggan |
| `GET`, `PUT`, `DELETE` | `/api/customers/{id}` | Token | Baca, ubah, atau hapus pelanggan |
| `GET`, `POST` | `/api/transactions` | Token | Daftar transaksi dan checkout |
| `GET` | `/api/transactions/{id}` | Token | Detail transaksi |
| `POST` | `/api/transactions/{id}/void` | Token | Batalkan transaksi |
| `GET` | `/api/transactions/{id}/receipt` | Token | Unduh struk PDF |
| `GET` | `/api/reports/sales` | Token | Laporan penjualan |
| `GET` | `/api/reports/sales/pdf` | Token | Ekspor laporan penjualan PDF |
| `GET` | `/api/reports/sales/excel` | Token | Ekspor laporan penjualan Excel |
| `GET` | `/api/reports/products` | Token | Laporan produk |
| `GET` | `/api/reports/products/pdf` | Token | Ekspor laporan produk PDF |
| `GET` | `/api/reports/products/excel` | Token | Ekspor laporan produk Excel |
| `GET` | `/api/reports/profit-loss` | Token | Laporan laba rugi |
| `GET` | `/api/reports/profit-loss/pdf` | Token | Ekspor laba rugi PDF |
| `GET` | `/api/reports/profit-loss/excel` | Token | Ekspor laba rugi Excel |
| `GET` | `/api/reports/transactions` | Token | Laporan transaksi |
| `GET` | `/api/reports/transactions/pdf` | Token | Ekspor transaksi PDF |
| `GET` | `/api/reports/transactions/excel` | Token | Ekspor transaksi Excel |
| `GET` | `/api/reports/customers/{id}` | Token | Histori pembelian pelanggan |
| `GET` | `/api/reports/customers/{id}/pdf` | Token | Ekspor histori pelanggan PDF |
| `GET` | `/api/reports/customers/{id}/excel` | Token | Ekspor histori pelanggan Excel |
| `GET`, `PUT` | `/api/settings` | Token | Baca dan ubah pengaturan toko |
| `GET` | `/api/backup` | Token | Unduh dump SQL |
| `POST` | `/api/backup/restore` | Token | Restore dump SQL |
| `GET` | `/api/export/master` | Token | Ekspor data master Excel |
| `GET` | `/api/activity-logs` | Token | Membaca activity log |

Parameter pencarian, filter, tanggal, dan pagination bergantung pada route dan dapat dilihat pada halaman laporan terkait serta handler di `backend/internal/handler/`.

## Backup dan Restore

Backup dan restore memanggil program CLI MySQL dari server aplikasi:

- Backup menggunakan `mysqldump` dan mengunduh file `.sql`.
- Restore mengirim file `.sql` ke program `mysql` untuk dijalankan pada database yang dikonfigurasi.
- Pastikan binary tersebut tersedia di `PATH` atau salah satu lokasi standar yang dideteksi aplikasi di Windows.
- Restore dapat mengubah atau menimpa data. Buat backup sebelum menjalankannya dan pastikan file sumber tepercaya.

## Pengembangan dan Pemeriksaan

Dari direktori `backend`:

```bash
# Jalankan server
go run ./cmd/api

# Build binary
go build -o kasir-api ./cmd/api

# Jalankan seluruh tes
go test ./...
```

Frontend terdiri dari file statis di `frontend/`; tidak ada langkah build JavaScript. Perubahan backend memerlukan server dimulai ulang. File SQL `backend/migrations/001_schema.sql` dan `002_seed.sql` disediakan untuk referensi atau pemakaian manual.

## Keamanan dan Batasan

- Aplikasi saat ini memakai satu role owner/admin; belum ada role kasir dengan hak akses berbeda.
- JWT default berlaku 24 jam untuk access token dan 168 jam untuk refresh token. Frontend mencoba refresh saat menerima HTTP 401.
- Konfigurasi CORS backend saat ini mengizinkan semua origin. Batasi origin sebelum server dapat diakses dari jaringan publik.
- Ganti password `admin` dan `JWT_SECRET`, gunakan password database yang kuat, dan lindungi `.env`, backup SQL, serta direktori upload.
- Aplikasi belum menyediakan payment gateway atau biaya operasional untuk perhitungan laba bersih.
- Gunakan HTTPS dan kontrol akses jaringan pada deployment di luar komputer lokal. Jangan mengekspos database MySQL langsung ke internet.

## Troubleshooting

| Gejala | Pemeriksaan |
| --- | --- |
| Server gagal terhubung ke database | Pastikan MySQL berjalan dan `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, serta `DB_NAME` sesuai. User database perlu hak membuat database saat startup pertama. |
| Port `8080` sudah digunakan | Ubah `APP_PORT` di `.env`, lalu jalankan ulang server. |
| Halaman menampilkan sesi berakhir | Login kembali; token disimpan di browser dan refresh otomatis hanya berlaku selama refresh token sah. |
| Perubahan backend belum terlihat | Hentikan proses Go dengan `Ctrl+C`, jalankan kembali `go run ./cmd/api`, lalu muat ulang halaman. |
| Backup atau restore gagal | Pastikan `mysqldump` atau `mysql` tersedia dan dapat mengakses database dengan konfigurasi yang sama. |
| Grafik tidak muncul | Periksa koneksi browser ke CDN Chart.js dan lihat console developer tools. |
| Produk tidak ditemukan lewat lookup | Pastikan barcode atau SKU cocok; pencarian nama dapat dilakukan dari daftar produk POS. |

## Struktur Data Utama

- `owners`: akun pemilik aplikasi.
- `categories`: kategori produk.
- `products`: katalog, harga, satuan, dan stok terkini.
- `stock_movements`: histori perubahan stok.
- `customers`: data pelanggan.
- `transactions`: invoice, pembayaran, pajak, status, dan pelanggan opsional.
- `transaction_items`: snapshot produk, kuantitas, harga beli/jual, diskon item, dan subtotal saat transaksi.
- `settings`: identitas toko, pajak, stok minimum, prefix invoice, dan lebar struk.
- `activity_logs`: aktivitas penting aplikasi.
