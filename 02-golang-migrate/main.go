// Demo 2 — golang-migrate dipakai sebagai library, bukan sebagai CLI.
//
// Kenapa sebagai library? Karena migrations-nya ikut ter-embed ke dalam binary
// (lihat //go:embed di bawah). Jadi pas deploy, kalian cuma kirim satu file
// binary — nggak perlu mikirin "folder migrations-nya kebawa nggak ya ke pod?".
// Ini pola yang paling sering dipakai di service Go production.
//
//	go run ./02-golang-migrate -cmd up
//	go run ./02-golang-migrate -cmd down -n 1
//	go run ./02-golang-migrate -cmd version
//	go run ./02-golang-migrate -cmd force -n 2
package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"

	"database/sql"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

//go:embed migrations-broken/*.sql
var brokenFiles embed.FS

func main() {
	cmd := flag.String("cmd", "up", "up | down | version | force")
	n := flag.Int("n", 1, "jumlah langkah untuk down, atau nomor versi untuk force")
	dir := flag.String("dir", "ok", "ok = migrations/, broken = migrations-broken/")
	flag.Parse()

	m := newMigrator(*dir)
	defer m.Close()

	switch *cmd {
	case "up":
		run(m.Up())
	case "down":
		run(m.Steps(-*n))
	case "force":
		// force dipakai kalau state-nya dirty: kita bilang ke tool
		// "percaya deh, database sekarang beneran ada di versi ini".
		run(m.Force(*n))
	case "version":
		printVersion(m)
		return
	default:
		log.Fatalf("perintah tidak dikenal: %s", *cmd)
	}
	printVersion(m)
}

func newMigrator(dir string) *migrate.Migrate {
	root, files := "migrations", fs.FS(migrationFiles)
	if dir == "broken" {
		root, files = "migrations-broken", fs.FS(brokenFiles)
	}

	src, err := iofs.New(files, root)
	must(err)

	dsn := os.Getenv("DSN")
	if dsn == "" {
		dsn = "postgres://demo:demo@localhost:5433/migrasi_demo?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	must(err)

	// Driver ini yang bikin dan mengelola tabel schema_migrations.
	// Dia juga ambil advisory lock di Postgres, jadi kalau 3 pod
	// deploy bareng, cuma satu yang jalanin migration-nya.
	drv, err := postgres.WithInstance(db, &postgres.Config{})
	must(err)

	m, err := migrate.NewWithInstance("iofs", src, "postgres", drv)
	must(err)
	return m
}

func run(err error) {
	switch {
	case errors.Is(err, migrate.ErrNoChange):
		fmt.Println("Nggak ada migration baru. Database sudah paling update.")
	case err != nil:
		fmt.Printf("MIGRATION GAGAL: %v\n", err)
		os.Exit(1)
	default:
		fmt.Println("Migration berhasil.")
	}
}

func printVersion(m *migrate.Migrate) {
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("Versi sekarang: (kosong, belum ada migration yang jalan)")
		return
	}
	must(err)
	fmt.Printf("Versi sekarang: %d | dirty: %t\n", v, dirty)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
