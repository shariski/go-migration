# Schema Migration di Go

**Falahudin Halim Shariski** — Technical Lead, AIG Nusa Hub

HiColleagues Sharing Session Vol. 07 · 11 Agustus 2026

---

## Repo

Semua kode dan demo ada di:

### `github.com/shariski/go-migration`

```bash
make up
make demo-gorm
```

---

## Masalah yang mau diselesaikan

Struktur database berbeda-beda antar environment.

Di lokal kolomnya ada. Di staging ada. Di production nggak ada.

```
ERROR: column "phone_number" of relation "users" does not exist
```

Penyebabnya bukan SQL-nya salah, tapi nggak ada catatan
SQL mana yang sudah dijalankan di server mana.

---

## Apa itu schema migration

Version control untuk struktur database.

Setiap perubahan struktur jadi satu file, punya nomor urut, dan tercatat
mana yang sudah dijalankan di mana.

---

## Yang diselesaikan schema migration

**Reproducible** — database baru bisa dibangun ulang dari nol, hasilnya sama.

**Kolaborasi** — tarik kode, jalankan migration, struktur langsung sinkron.

**Otomatis** — CI/CD bisa menjalankan sendiri.

**Jejak perubahan** — bisa dilacak kolom ini kapan dan kenapa ditambahkan.

---

## Cara konvensional

File SQL dikumpulkan di satu folder, dijalankan manual.

```
migrations/
  create_users.sql
  add_phone.sql
  fix_email_index.sql
  add_phone_v2.sql
```

Koordinasinya lewat chat atau dokumen.

---

## Kenapa cara konvensional gagal

Yang menyimpan state-nya ingatan manusia.

- Nggak ada catatan mana yang sudah dijalankan
- Urutan eksekusi ambigu
- Dijalankan dua kali bisa error
- Setup lokal untuk anggota tim baru jadi manual

File SQL-nya sendiri sudah benar. Yang nggak ada itu pencatatnya.

---

## GORM AutoMigrate

GORM adalah ORM yang paling banyak dipakai di Go.

```go
db.AutoMigrate(&User{})
```

Struktur tabel disesuaikan dengan struct.

---

## Cara kerja AutoMigrate

1. Baca struct pakai reflection
2. Query ke `information_schema` untuk tahu kolom yang ada sekarang
3. Bandingkan keduanya
4. Generate `ALTER TABLE` untuk yang belum ada

Perbandingan di langkah 3 cuma satu arah:

> *Field mana di struct yang belum ada di tabel?*

Pertanyaan kebalikannya nggak pernah ditanyakan.

---

## Demo: perubahan struct

```go
// Sebelum
type User struct {
    ID    uint
    Name  string
    Email string
    Phone string
}
```

```go
// Sesudah — Phone dihapus, Name diganti FullName, Age ditambah
type User struct {
    ID       uint
    FullName string
    Email    string
    Age      int
}
```

---

## SQL yang dijalankan GORM

```sql
ALTER TABLE "users" ADD "full_name" text
ALTER TABLE "users" ADD "age" bigint
```

Hanya dua statement. Nggak ada error dan nggak ada warning.

---

## Hasil di database

```
   Column   |  Type
------------+--------
 id         | bigint
 name       | text     <- masih ada
 email      | text
 phone      | text     <- masih ada, padahal sudah dihapus dari struct
 full_name  | text     <- kolom baru, kosong
 age        | bigint
```

```
 id |     name     | full_name |   phone
----+--------------+-----------+-----------
  1 | Budi Santoso |           | 0812-1111
  2 | Siti Rahayu  |           | 0813-2222
```

Data lama tetap di kolom `name`, yang di kode sudah nggak ada.

---

## Batasan AutoMigrate

| Operasi | Status |
|---|---|
| Tambah kolom | Bisa |
| Drop kolom | Nggak pernah dilakukan |
| Rename kolom | Bikin kolom baru, data lama ditinggal |
| Ubah tipe data | Terbatas |
| Backfill data | Nggak ditangani |
| Rollback | Nggak ada |
| Catatan versi | Nggak ada |

---

## Konsekuensi paling besar

AutoMigrate nggak menyimpan versi.

Pertanyaan *"production sekarang di schema versi berapa?"*
nggak bisa dijawab, karena nggak ada yang mencatat.

---

## Posisi AutoMigrate

AutoMigrate bukan migration system, tapi schema sync-er.

Cocok untuk prototyping lokal, test fixture, dan project yang datanya
boleh hilang.

Masalah muncul ketika dipakai untuk hal yang di luar cakupannya.

---

## golang-migrate: struktur file

Satu perubahan terdiri dari dua file.

```
migrations/
  000001_create_users.up.sql
  000001_create_users.down.sql
  000002_add_phone_to_users.up.sql
  000002_add_phone_to_users.down.sql
  000003_split_user_name.up.sql
  000003_split_user_name.down.sql
```

Nomor urut di depan menentukan urutan eksekusi.

---

## Perintah dasar

```bash
migrate up              # jalankan semua yang belum
migrate down 1          # mundur satu langkah
migrate version         # cek versi sekarang
migrate force 3         # set versi secara manual
```

---

## Tabel schema_migrations

```sql
SELECT * FROM schema_migrations;
```

```
 version | dirty
---------+-------
       4 | f
```

Dua kolom. `version` adalah migration terakhir yang berhasil,
`dirty` menandakan ada migration yang gagal di tengah.

Ini seluruh state yang disimpan golang-migrate.

---

## Kolom dirty

Urutan yang dilakukan setiap menjalankan satu migration:

1. Tulis `version = N`, `dirty = true`
2. Jalankan SQL-nya
3. Kalau berhasil, set `dirty = false`

Kalau langkah 2 gagal, `dirty` tetap `true`.

```
MIGRATION GAGAL: Dirty database version 2. Fix and force version.
```

golang-migrate menolak jalan lagi sampai diperbaiki manual.

---

## Contoh migration yang bikin dirty

```sql
ALTER TABLE orders ADD COLUMN status TEXT NOT NULL;
```

Postgres harus mengisi nilai untuk semua baris yang sudah ada.
Tanpa `DEFAULT`, yang bisa diisi cuma `NULL`, padahal `NOT NULL`
melarang `NULL`.

| Isi tabel | Hasil |
|---|---|
| 0 baris | Berhasil |
| Ada isinya | Gagal |

Database lokal dan CI biasanya kosong, production nggak.
Karena itu kesalahan ini sering baru ketahuan di production.

Perbaikannya: `... NOT NULL DEFAULT 'pending'`

---

## Advisory lock

Kalau beberapa pod deploy bersamaan, semuanya akan mencoba menjalankan
migration.

golang-migrate mengambil advisory lock di Postgres, sehingga hanya satu
proses yang jalan dan sisanya menunggu.

---

## Embed migration ke binary

```go
//go:embed migrations/*.sql
var migrationFiles embed.FS
```

Migration ikut masuk ke dalam binary, jadi deploy cukup mengirim satu file
dan nggak perlu memastikan folder `migrations/` ikut terbawa.

---

## Batasan down migration

```sql
-- 000004_drop_users_name.up.sql
ALTER TABLE users DROP COLUMN name;
```

```sql
-- 000004_drop_users_name.down.sql
ALTER TABLE users ADD COLUMN name TEXT;
```

Kolomnya kembali, datanya nggak.

`DROP COLUMN` menghapus data secara permanen.

---

## Kapan down migration berguna

Down migration cuma bisa mengembalikan informasi yang masih tersimpan
di tempat lain di database. Dia menghitung ulang, bukan mengembalikan.

**Di lokal**, down berguna: salah bikin migration, mundur, perbaiki.

**Di production**, lebih aman roll forward: bikin migration baru
yang memperbaiki.

Yang mengembalikan data saat darurat adalah backup, bukan file `.down.sql`.

---

## Expand–contract

Perubahan dipecah jadi beberapa deploy terpisah.

1. **Expand** — tambah kolom baru dan isi datanya, kolom lama dibiarkan
2. **Migrate** — update semua service supaya pakai kolom baru
3. **Contract** — hapus kolom lama

Alasannya: saat migration jalan, kode versi lama masih hidup dan masih
membaca kolom lama. Kalau langsung dihapus, request ke kode lama error.

Contohnya ada di repo, migration `000003` dan `000004`.

---

## Inti dari migration runner

1. Baca versi database sekarang
2. Ambil file yang nomornya lebih besar dari versi itu
3. Jalankan berurutan di dalam transaksi
4. Update versinya di transaksi yang sama

Langkah 3 dan 4 harus dalam satu transaksi, supaya nggak terjadi kondisi
SQL-nya sudah jalan tapi versinya belum tercatat.

Implementasi lengkapnya ada di repo: `03-mini-runner/`, sekitar 90 baris.

---

## Ringkasan

**AutoMigrate adalah schema sync-er, bukan migration system.**
Cocok untuk lokal, bukan untuk production.

**Migration tool menyimpan state yang sederhana.**
Satu tabel versi, file terurut, dan transaksi.

**Down migration untuk lokal, production roll forward.**
Pemulihan data mengandalkan backup.

---

## Praktik yang disarankan

- Jangan mengubah file migration yang sudah jalan di production
- Kolom baru dibuat nullable atau diberi `DEFAULT`
- Jalankan migration sebelum deploy aplikasi, bukan bersamaan
- Buat index dengan `CREATE INDEX CONCURRENTLY` di tabel besar
- Backfill data adalah bagian dari migration, bukan langkah terpisah

---

## Tool lain

| Tool | Catatan |
|---|---|
| **golang-migrate** | Paling umum dipakai |
| **goose** | Serupa, migration bisa ditulis dengan Go |
| **atlas** | Deklaratif, SQL-nya digenerate dari deklarasi schema |
| **GORM AutoMigrate** | Lokal dan prototyping |

---

# Terima kasih

### `github.com/shariski/go-migration`

```bash
make up
make demo-gorm        # AutoMigrate nggak drop kolom
make demo-migrate     # isi tabel schema_migrations
make demo-rollback    # batasan down migration
make demo-dirty       # migration gagal di tengah
make demo-mini        # migration runner sederhana
```

**Falahudin Halim Shariski** — Technical Lead, AIG Nusa Hub

---

## Catatan tambahan untuk Q&A

**Tabel dengan jutaan baris**
`ADD COLUMN` nullable atau dengan `DEFAULT` nilai tetap di Postgres 11+
cepat, cuma mengubah metadata. Yang lama: bikin index tanpa `CONCURRENTLY`
dan ubah tipe kolom, karena tabelnya ditulis ulang.
`ADD COLUMN NOT NULL` tanpa `DEFAULT` bukan lambat, tapi langsung gagal
kalau tabelnya ada isinya.

**Migration dijalankan di mana**
Lebih aman sebagai step terpisah sebelum deploy. Kalau dipanggil di `main()`,
semua pod menjalankan bersamaan dan mengandalkan advisory lock.

**Pindah dari AutoMigrate ke golang-migrate**
Dump schema yang sudah ada jadi `000001_init.up.sql`, lanjutkan manual
dari situ.

**Seed data di migration**
Data referensi seperti daftar provinsi atau status order boleh.
Data dummy sebaiknya nggak, karena ikut terbawa ke production.

**Kenapa dirty nggak bisa pulih otomatis**
Tool nggak tahu SQL-nya sudah jalan sampai statement ke berapa.
Di MySQL lebih rumit karena DDL memicu commit otomatis.
