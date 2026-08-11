DSN ?= postgres://demo:demo@localhost:5433/migrasi_demo?sslmode=disable
PSQL := docker compose exec -T db psql -U demo -d migrasi_demo

.DEFAULT_GOAL := help

help: ## Tampilkan daftar perintah
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

up: ## Nyalakan Postgres dan tunggu sampai siap
	docker compose up -d
	@echo "Menunggu Postgres siap..."
	@until docker compose exec -T db pg_isready -U demo -d migrasi_demo >/dev/null 2>&1; do sleep 1; done
	@echo "Postgres siap di localhost:5433"

down: ## Matikan Postgres dan hapus datanya
	docker compose down -v

psql: ## Masuk ke psql interaktif
	docker compose exec db psql -U demo -d migrasi_demo

reset: ## Kosongkan database (schema public dibuat ulang)
	@$(PSQL) -q -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@echo "Database dikosongkan."

# ---------------------------------------------------------------- demo 1
demo-gorm: reset ## Demo: AutoMigrate nggak pernah drop kolom
	@echo "\n=== LANGKAH 1: struct versi lama, lalu isi data ===\n"
	@go run ./01-gorm-automigrate -step 1
	@$(PSQL) -c "\d users"
	@$(PSQL) -c "SELECT * FROM users;"
	@echo "\n=== LANGKAH 2: struct diubah, AutoMigrate dijalankan lagi ===\n"
	@go run ./01-gorm-automigrate -step 2
	@$(PSQL) -c "\d users"
	@echo "\n>>> Perhatikan: kolom phone MASIH ADA, full_name kosong, data lama nyangkut di name.\n"
	@$(PSQL) -c "SELECT id, name, full_name, phone FROM users;"

# ---------------------------------------------------------------- demo 2
demo-migrate: reset ## Demo: golang-migrate up/down + isi tabel schema_migrations
	@echo "\n=== migrate up (000001 s/d 000004) ===\n"
	@go run ./02-golang-migrate -cmd up
	@echo "\n>>> Backfill di 000003 jalan: first_name/last_name keisi dari name.\n"
	@$(PSQL) -c "SELECT id, first_name, last_name, email FROM users;"
	@echo "\n>>> Ini seluruh 'otak' dari golang-migrate. Dua kolom. Itu aja.\n"
	@$(PSQL) -c "SELECT * FROM schema_migrations;"
	@echo "\n=== turun 1 versi: batalin 000004_drop_users_name ===\n"
	@go run ./02-golang-migrate -cmd down -n 1
	@echo "\n>>> Kolom 'name' balik, dan isinya kelihatan utuh.\n"
	@$(PSQL) -c "SELECT id, name, first_name, last_name FROM users;"
	@echo ">>> Tapi ini BUKAN data asli — ini hasil nyusun ulang dari first_name +"
	@echo ">>> last_name yang kebetulan masih ada. Lihat 'make demo-rollback'"
	@echo ">>> buat kasus yang beneran nggak bisa diselamatkan.\n"

demo-rollback: reset ## Demo: kenapa 'down migration' nggak nyelametin data
	@go run ./02-golang-migrate -cmd up
	@# Ini MEWAKILI data yang diisi user lewat aplikasi, bukan bagian dari
	@# migration. Bedanya penting: data aplikasi nggak ikut ke-generate ulang
	@# waktu migration dijalankan lagi.
	@$(PSQL) -q -c "UPDATE users SET phone = '0812-' || lpad(id::text, 4, '0');"
	@echo "\n>>> Kondisi awal: user sudah isi nomor HP lewat aplikasi.\n"
	@$(PSQL) -c "SELECT id, first_name, phone FROM users;"
	@echo "\n=== Ada bug di produksi. Panik. Rollback 3 versi. ===\n"
	@go run ./02-golang-migrate -cmd down -n 3
	@echo "\n=== Bug sudah dibenerin, deploy maju lagi. ===\n"
	@go run ./02-golang-migrate -cmd up
	@echo "\n>>> Kolom phone balik. Datanya? Hilang. Selamanya.\n"
	@$(PSQL) -c "SELECT id, first_name, phone FROM users;"
	@echo ">>> DROP COLUMN itu nggak ada tombol undo-nya. Yang nyelametin kalian"
	@echo ">>> cuma backup — bukan file .down.sql.\n"

demo-dirty: reset ## Demo: migration gagal di tengah jalan -> dirty state
	@go run ./02-golang-migrate -cmd up -dir broken || true
	@echo "\n>>> dirty = true. Tool nolak jalan lagi sampai kalian benerin manual.\n"
	@$(PSQL) -c "SELECT * FROM schema_migrations;"
	@echo "\n=== coba jalankan lagi ===\n"
	@go run ./02-golang-migrate -cmd up -dir broken || true

# ---------------------------------------------------------------- demo 3
demo-mini: reset ## Demo: migration runner bikinan sendiri (~90 baris)
	@go run ./03-mini-runner
	@$(PSQL) -c "SELECT * FROM mini_schema_migrations;"
	@$(PSQL) -c "\d products"

# ---------------------------------------------------------------- slide
slides: ## Generate slides/presentasi.pptx dari presentasi.md
	@python3 -c "import pptx" 2>/dev/null || { echo "Butuh python-pptx:  pip install python-pptx"; exit 1; }
	@python3 slides/build_pptx.py

.PHONY: help up down psql reset demo-gorm demo-migrate demo-rollback demo-dirty demo-mini slides
