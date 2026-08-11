# Schema Migration di Go

Materi pendukung sesi **"Schema Migration System"** — HiColleagues Sharing
Session Vol. 07: *Building a Custom ORM in Go*, 11 Agustus 2026.

Repo ini isinya demo yang bisa kalian jalanin sendiri. Kalau kalian nonton
sesinya, ini tempat buat ngulang pelan-pelan. Kalau nggak nonton, README ini
dibuat supaya tetap nyambung dibaca sendiri.

## Yang dibahas

1. Kenapa `AutoMigrate` bawaan GORM **bukan** migration system
2. Cara kerja `golang-migrate`, sampai isi tabel `schema_migrations`-nya
3. Kenapa `down` migration nggak nyelametin data kalian
4. Bikin migration runner sendiri, ~90 baris

## Butuh apa

- Go 1.22+
- Docker (buat Postgres-nya)

## Mulai

```bash
make up          # nyalain Postgres di localhost:5433
make help        # lihat semua demo yang ada
```

Kalau sudah selesai:

```bash
make down        # matiin container + hapus datanya
```

---

## Demo 1 — Kolomnya kok nggak ilang?

```bash
make demo-gorm
```

Ada dua struct yang dua-duanya nunjuk ke tabel `users`:

```go
type UserV1 struct {                type UserV2 struct {
    ID    uint                          ID       uint
    Name  string                        FullName string
    Email string                        Email    string
    Phone string                        Age      int
}                                   }
```

Bayangin `UserV1` itu struct kalian minggu lalu, `UserV2` yang hari ini.
Kalian hapus `Phone`, ganti `Name` jadi `FullName`, tambah `Age`.

Di dunia nyata ini satu struct yang kalian edit. Di sini sengaja dipisah dua
biar demonya bisa diulang-ulang tanpa perlu edit file.

**Yang GORM jalankan:**

```sql
ALTER TABLE "users" ADD "full_name" text
ALTER TABLE "users" ADD "age" bigint
```

Dua statement. Nggak ada error, nggak ada warning.

**Tabelnya jadi:**

```
 id | name         | email          | phone     | full_name | age
----+--------------+----------------+-----------+-----------+-----
  1 | Budi Santoso | budi@contoh.id | 0812-1111 |           |
```

`phone` masih ada. `full_name` kosong. Data aslinya nyangkut di `name` —
kolom yang menurut kode kalian sudah nggak ada.

### Kenapa bisa gitu

AutoMigrate baca struct kalian, terus nanya ke `information_schema.columns`
tabelnya sekarang punya kolom apa. Terus dia bandingin — **tapi cuma satu arah**:

> *"Field mana di struct yang belum ada di tabel?"*

Dia nggak pernah nanya kebalikannya, *"kolom mana di tabel yang sudah nggak ada
di struct?"*. Makanya dia nggak akan pernah `DROP COLUMN`.

Ini bukan bug. Ini keputusan desain yang sengaja — biar dia nggak menghapus data
kalian tanpa permisi. Masalahnya muncul waktu kita ngira dia migration system.

### Yang AutoMigrate nggak bisa

| | |
|---|---|
| Drop kolom | Nggak akan pernah |
| Rename kolom | Bikin kolom baru, data lama ditinggal |
| Ubah tipe data | Kadang bisa, kadang diem-diem gagal |
| Mindahin/backfill data | Bukan urusan dia |
| Rollback | Nggak ada |
| **Nyatet versi** | **Nggak ada sama sekali** |

Yang terakhir itu yang paling fatal. Coba jawab *"production sekarang di schema
versi berapa?"* — nggak akan bisa, karena nggak ada yang nyatet.

### Terus AutoMigrate jelek dong?

Nggak. Dia melakukan persis yang dia janjikan.

**AutoMigrate itu schema sync-er, bukan migration system.** Buat prototyping
lokal, bikin test fixture, atau side project yang datanya boleh hilang, dia
enak banget. Masalah muncul waktu kita pakai buat hal yang memang bukan tugasnya.

---

## Demo 2 — Bedah golang-migrate

```bash
make demo-migrate
```

Satu perubahan = dua file, dengan nomor urut di depan:

```
000001_create_users.up.sql / .down.sql
000002_add_phone_to_users.up.sql / .down.sql
000003_split_user_name.up.sql / .down.sql
000004_drop_users_name.up.sql / .down.sql
```

### Isi "otak"-nya

Ini bagian yang paling bikin lega waktu pertama kali lihat:

```sql
SELECT * FROM schema_migrations;
```

```
 version | dirty
---------+-------
       4 | f
```

Dua kolom. Itu doang. `version` = migration terakhir yang sukses jalan,
`dirty` = lagi ada yang gagal di tengah atau nggak.

Semua "keajaiban" migration tool ujungnya cuma ngurus satu baris ini.

### Kolom `dirty`

Alurnya golang-migrate tiap jalanin satu migration:

1. Tulis `version = N, dirty = true`
2. Jalanin SQL-nya
3. Kalau sukses, `dirty = false`

Kalau langkah 2 gagal, `dirty` nyangkut di `true`. Coba sendiri:

```bash
make demo-dirty
```

```
MIGRATION GAGAL: Dirty database version 2. Fix and force version.
```

Tool-nya nolak jalan lagi. Ini disengaja — dia nggak tahu SQL kalian sudah
jalan sampai baris ke berapa, jadi dia nggak berani nebak.

Cara benerinnya manual:

1. Lihat langsung ke database, cek migration itu sebenernya nyampe mana
2. Beresin sisanya pakai tangan
3. `migrate force <versi>` buat bilang "udah, sekarang beneran di versi ini"

Migration yang bikin dirty di demo ini kesalahan klasik:

```sql
ALTER TABLE orders ADD COLUMN status TEXT NOT NULL;
```

Perhatiin: nggak ada `DEFAULT`.

Waktu nambah kolom, Postgres harus ngasih nilai ke **semua baris yang sudah ada
sebelumnya**. Nggak ada `DEFAULT` berarti satu-satunya yang bisa dia isi itu
`NULL`. Tapi `NOT NULL` justru melarang `NULL`. Jadi perintahnya bertabrakan
sama dirinya sendiri, dan Postgres nolak:

```
ERROR: column "status" of relation "orders" contains null values
```

Kuncinya ada di **jumlah baris yang sudah ada**:

| Isi tabel | Hasil |
|---|---|
| 0 baris | Berhasil — nggak ada yang perlu diisi, jadi nggak ada tabrakan |
| ada isinya | Gagal — baris lama mau diisi apa? |

Nah, ini yang bikin dia jadi kesalahan klasik. Database lokal dan CI biasanya
kosong atau baru di-seed, jadi migration-nya lolos, review lolos, CI hijau.
Production punya jutaan baris — di situlah tabrakannya baru kejadian, dan cuma
di situ.

Di demo ini `000001` sengaja masukin 2 baris data dulu, biar kegagalannya
kejadian juga di laptop kalian. Kalau tabelnya dibiarkan kosong, migration-nya
malah sukses dan nggak ada yang bisa didemokan.

Perbaikannya: kasih `DEFAULT`, biar Postgres punya nilai buat diisi ke
baris-baris lama.

```sql
ALTER TABLE orders ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
```

### Advisory lock

Deploy 3 pod barengan, ketiganya nyoba migrate barengan. golang-migrate ambil
advisory lock di Postgres, jadi cuma satu yang jalan, sisanya nunggu.

### `embed.FS`

Migration ikut masuk ke dalam binary:

```go
//go:embed migrations/*.sql
var migrationFiles embed.FS
```

Deploy cukup kirim satu file binary. Nggak perlu mikir *"folder migrations-nya
kebawa nggak ya ke container?"*. Lihat `02-golang-migrate/main.go`.

---

## Demo 3 — Kenapa `down` nggak nyelametin data

```bash
make demo-rollback
```

Hampir semua tutorial bilang *"selalu tulis down migration!"*. Ini nggak
sepenuhnya benar, dan bedanya penting.

Alurnya di demo:

1. Migration jalan sampai versi 4
2. User isi nomor HP lewat aplikasi — data beneran
3. Ada bug, panik, `migrate down 3`
4. Bug dibenerin, `migrate up` lagi

Hasilnya:

```
 id | first_name | phone
----+------------+-------
  1 | Budi       |
  2 | Siti       |
```

Kolom `phone` balik. Isinya kosong. `DROP COLUMN` nggak ada tombol undo-nya.

### Posisi yang saya ambil

**`down` itu buat lokal.** Lagi ngembangin, salah bikin migration, mundur,
betulin. Enak dan aman.

**Di production, kita maju, bukan mundur.** Ada masalah? Bikin migration baru
yang benerin. Yang nyelametin data kalian pas darurat itu **backup**, bukan
file `.down.sql`.

Lihat `000004_drop_users_name.down.sql` — di situ ditulis apa adanya:
kolomnya bisa dibalikin, datanya nggak.

### Terus gimana dong? Expand–contract

Pecah jadi beberapa deploy yang terpisah:

**1. Expand** — tambah yang baru, jangan sentuh yang lama
([`000003`](02-golang-migrate/migrations/000003_split_user_name.up.sql))

```sql
ALTER TABLE users ADD COLUMN first_name TEXT;
ALTER TABLE users ADD COLUMN last_name  TEXT;

UPDATE users SET first_name = split_part(name, ' ', 1), ...;
```

**2. Migrate** — update semua service supaya baca `first_name`/`last_name`.
Nggak ada perubahan database di tahap ini.

**3. Contract** — baru hapus yang lama
([`000004`](02-golang-migrate/migrations/000004_drop_users_name.up.sql))

```sql
ALTER TABLE users DROP COLUMN name;
```

Kenapa harus dipisah? Karena pas migration jalan, **kode versi lama masih hidup**
dan masih baca kolom `name`. Kalau langsung dihapus, aplikasi mati sebelum
deploy selesai.

---

## Demo 4 — Bikin runner sendiri

```bash
make demo-mini
```

Setelah lihat isi `schema_migrations` cuma dua kolom, wajar kalau kepikiran:
*"berarti bisa bikin sendiri dong?"* Bisa. Intinya empat langkah:

1. Baca versi database sekarang
2. Ambil file yang nomornya lebih besar dari versi itu
3. Jalanin urut, **di dalam transaksi**
4. Update versinya di transaksi yang sama

Langkah 3 dan 4 harus satu transaksi. Kalau dipisah, bisa kejadian
"SQL-nya jalan tapi versinya nggak kecatat" — dan pas dijalanin lagi,
migration-nya diulang.

Implementasinya di [`03-mini-runner/main.go`](03-mini-runner/main.go), ~90 baris.

### Yang sengaja nggak ada di situ

Ini bahan belajar, bukan buat production. Yang belum ada:

- **Locking** — 3 pod deploy bareng, ketiganya jalan barengan
- **Down migration** — butuh file `.down.sql` dan logika mundur
- **Checksum** — buat deteksi kalau ada yang ngedit file migration yang
  sudah pernah jalan di production
- **Multi-database, CLI, embed**, dan seterusnya

Satu catatan menarik: mini runner ini bisa pakai transaksi karena DDL di
Postgres itu transaksional. Di MySQL nggak bisa — `CREATE TABLE` di sana bikin
transaksi ke-commit otomatis. Itulah kenapa tool beneran perlu konsep
"dirty state".

---

## Kebiasaan yang layak dibawa

- Jangan pernah edit file migration yang **sudah** jalan di production.
  Bikin file baru.
- Kolom baru: nullable dulu, atau kasih `DEFAULT`. Jangan `NOT NULL` polos.
- Jalanin migration **sebelum** deploy aplikasinya, bukan barengan.
- Bikin index pakai `CREATE INDEX CONCURRENTLY` di tabel yang sudah besar.
- Migration itu bukan cuma soal struktur — backfill data juga bagian dari
  tugasnya.
- Tim kalian harusnya bisa jawab *"prod di versi berapa?"* dalam 10 detik.

## Tool lain

| Tool | Kapan cocok |
|---|---|
| [golang-migrate](https://github.com/golang-migrate/migrate) | Default yang aman, paling banyak dipakai |
| [goose](https://github.com/pressly/goose) | Mirip, tapi migration bisa ditulis pakai Go |
| [atlas](https://atlasgo.io) | Deklaratif — tulis mau jadi apa, SQL-nya dia yang bikin |
| GORM AutoMigrate | Lokal dan prototyping aja |

## Isi repo

```
.
├── 01-gorm-automigrate/    demo "kolomnya kok nggak ilang?"
├── 02-golang-migrate/
│   ├── migrations/         alur expand-contract yang lengkap
│   └── migrations-broken/  migration yang sengaja gagal (demo dirty)
├── 03-mini-runner/         migration runner ~90 baris
└── slides/presentasi.md    materi presentasi
```
