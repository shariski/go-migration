// Demo 3 — Bikin migration runner sendiri.
//
// Ini bagian "buka kap mesin"-nya. Setelah lihat isi tabel schema_migrations
// tadi, harusnya kelihatan bahwa migration tool itu nggak seajaib kelihatannya.
// Intinya cuma empat langkah:
//
//  1. Baca versi database sekarang berapa.
//  2. Ambil file migration yang nomornya lebih besar dari versi itu.
//  3. Jalankan satu per satu, urut, di dalam transaksi.
//  4. Update nomor versinya di transaksi yang sama.
//
// Sisanya — CLI, dukungan banyak database, down migration, locking — itu
// fitur tambahan. Konsep intinya ya cuma yang di bawah ini.
//
// CATATAN: ini bahan belajar, bukan buat production. Yang sengaja TIDAK
// ada di sini dibahas di bagian bawah file.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

const versionTable = "mini_schema_migrations"

type migration struct {
	version int
	name    string
	sql     string
}

func main() {
	dir := flag.String("dir", defaultDir(), "folder berisi file .sql")
	flag.Parse()

	db, err := sql.Open("postgres", dsn())
	must(err)
	defer db.Close()

	must(ensureVersionTable(db))

	current, err := currentVersion(db)
	must(err)
	fmt.Printf("Versi database sekarang: %d\n", current)

	all, err := loadMigrations(*dir)
	must(err)

	applied := 0
	for _, m := range all {
		if m.version <= current {
			continue // sudah pernah jalan, lewati
		}
		fmt.Printf("  menjalankan %03d_%s...\n", m.version, m.name)
		must(apply(db, m))
		applied++
	}

	if applied == 0 {
		fmt.Println("Nggak ada migration baru.")
		return
	}
	fmt.Printf("Selesai. %d migration dijalankan.\n", applied)
}

// ensureVersionTable bikin tabel pencatat versi kalau belum ada.
// Ini satu-satunya tabel yang "milik" si runner.
func ensureVersionTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + versionTable + ` (
		version    BIGINT      NOT NULL PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)
	return err
}

func currentVersion(db *sql.DB) (int, error) {
	var v sql.NullInt64
	err := db.QueryRow(`SELECT max(version) FROM ` + versionTable).Scan(&v)
	return int(v.Int64), err // NULL -> 0, artinya database masih kosong
}

// loadMigrations baca semua file .sql lalu urutkan berdasarkan nomor di depan
// nama file. Urutan ini yang bikin migration deterministik: siapa pun yang
// menjalankan, di mana pun, hasil akhirnya sama.
func loadMigrations(dir string) ([]migration, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, err
	}

	var out []migration
	for _, p := range paths {
		base := strings.TrimSuffix(filepath.Base(p), ".sql")
		numStr, name, ok := strings.Cut(base, "_")
		if !ok {
			return nil, fmt.Errorf("nama file %q nggak sesuai format <versi>_<nama>.sql", base)
		}
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, fmt.Errorf("versi di %q bukan angka: %w", base, err)
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		out = append(out, migration{version: num, name: name, sql: string(body)})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// apply menjalankan satu migration DI DALAM SATU TRANSAKSI, bareng dengan
// pencatatan versinya.
//
// Ini bagian yang paling penting. Karena keduanya satu transaksi, nggak akan
// pernah ada kondisi "SQL-nya sudah jalan tapi versinya belum kecatat" atau
// sebaliknya. Kalau gagal di tengah, semuanya dibatalkan.
//
// Postgres bisa begini karena DDL-nya transaksional. MySQL nggak bisa —
// di MySQL, CREATE TABLE bikin transaksi ke-commit otomatis. Itulah kenapa
// tool beneran punya konsep "dirty state": buat database yang nggak bisa
// membatalkan perubahan struktur di tengah jalan.
func apply(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op kalau sudah ter-commit

	if _, err := tx.Exec(m.sql); err != nil {
		return fmt.Errorf("migration %d gagal: %w", m.version, err)
	}
	if _, err := tx.Exec(`INSERT INTO `+versionTable+` (version) VALUES ($1)`, m.version); err != nil {
		return err
	}
	return tx.Commit()
}

// defaultDir cari folder migrations relatif terhadap file ini, biar programnya
// tetap jalan mau dipanggil dari root repo atau dari dalam folder ini.
func defaultDir() string {
	if _, err := os.Stat("migrations"); err == nil {
		return "migrations"
	}
	return filepath.Join("03-mini-runner", "migrations")
}

func dsn() string {
	if v := os.Getenv("DSN"); v != "" {
		return v
	}
	return "postgres://demo:demo@localhost:5433/migrasi_demo?sslmode=disable"
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Yang sengaja NGGAK ada di sini, dan kenapa tool beneran butuh itu:
//
//   - Locking. Kalau 3 pod deploy bareng, ketiganya bakal jalanin migration
//     yang sama barengan. golang-migrate pakai advisory lock Postgres
//     buat mastiin cuma satu yang jalan.
//   - Down migration. Butuh file .down.sql dan logika mundur.
//   - Checksum. Buat mendeteksi kalau ada orang yang ngedit file migration
//     yang SUDAH pernah jalan di produksi — sumber bug yang susah dilacak.
//   - Dukungan multi-database, CLI, embed, dan seterusnya.
//
// Tapi konsep intinya tetap yang tadi: tabel versi, file terurut, transaksi.
