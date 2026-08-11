// Demo 1 — Apa yang sebenarnya dilakukan (dan TIDAK dilakukan) GORM AutoMigrate.
//
// Di dunia nyata ini satu struct `User` yang kalian edit dari waktu ke waktu.
// Di sini sengaja dipecah jadi dua struct supaya demonya bisa diulang-ulang
// tanpa perlu edit file pas lagi presentasi.
//
//	go run ./01-gorm-automigrate -step 1   # struct minggu lalu + isi data
//	go run ./01-gorm-automigrate -step 2   # struct hari ini
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// UserV1 — struct kita "minggu lalu".
type UserV1 struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Email string
	Phone string
}

func (UserV1) TableName() string { return "users" }

// UserV2 — struct kita "hari ini". Tiga perubahan:
//   - field Phone DIHAPUS
//   - field Name di-rename jadi FullName
//   - field Age ditambahkan
type UserV2 struct {
	ID       uint `gorm:"primaryKey"`
	FullName string
	Email    string
	Age      int
}

func (UserV2) TableName() string { return "users" }

func main() {
	step := flag.Int("step", 1, "1 = struct lama + seed data, 2 = struct baru")
	flag.Parse()

	db := connect()

	switch *step {
	case 1:
		fmt.Println("AutoMigrate pakai UserV1 { ID, Name, Email, Phone }")
		must(db.AutoMigrate(&UserV1{}))

		// Isi data supaya nanti kelihatan datanya ke mana.
		must(db.Create(&[]UserV1{
			{Name: "Budi Santoso", Email: "budi@contoh.id", Phone: "0812-1111"},
			{Name: "Siti Rahayu", Email: "siti@contoh.id", Phone: "0813-2222"},
			{Name: "Agus Wijaya", Email: "agus@contoh.id", Phone: "0814-3333"},
		}).Error)
		fmt.Println("3 baris data dimasukkan.")

	case 2:
		fmt.Println("AutoMigrate pakai UserV2 { ID, FullName, Email, Age }")
		fmt.Println("Harapan kita: phone hilang, name berubah jadi full_name.")
		must(db.AutoMigrate(&UserV2{}))
		fmt.Println("Selesai — tanpa error, tanpa warning. Sekarang cek tabelnya.")

	default:
		log.Fatalf("step tidak dikenal: %d", *step)
	}
}

func connect() *gorm.DB {
	dsn := os.Getenv("DSN")
	if dsn == "" {
		dsn = "postgres://demo:demo@localhost:5433/migrasi_demo?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: ddlOnlyLogger{},
	})
	must(err)
	return db
}

// ddlOnlyLogger cuma menampilkan SQL yang mengubah struktur tabel.
//
// Kalau kalian pakai logger bawaan GORM, AutoMigrate akan membanjiri terminal
// dengan puluhan query ke information_schema. Itu justru bagian yang menarik
// secara konsep (AutoMigrate memang "mengintip" struktur tabel yang ada dulu),
// tapi bikin poin utamanya ketutupan. Jadi di sini disaring.
type ddlOnlyLogger struct{ logger.Interface }

func (l ddlOnlyLogger) LogMode(logger.LogLevel) logger.Interface { return l }
func (ddlOnlyLogger) Info(context.Context, string, ...any)       {}
func (ddlOnlyLogger) Warn(context.Context, string, ...any)       {}
func (ddlOnlyLogger) Error(context.Context, string, ...any)      {}

func (ddlOnlyLogger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	switch {
	case err != nil:
		fmt.Printf("  !! %v\n", err)
	case strings.HasPrefix(sql, "ALTER TABLE"), strings.HasPrefix(sql, "CREATE TABLE"):
		fmt.Printf("  SQL> %s\n", sql)
	}
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
