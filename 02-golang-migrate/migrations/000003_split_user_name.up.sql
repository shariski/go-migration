-- Tahap EXPAND dari pola expand-contract.
--
-- Kita mau memecah `name` jadi `first_name` + `last_name`.
-- Yang TIDAK kita lakukan di sini: menghapus kolom `name`.
-- Alasannya: pas migration ini jalan, kode versi lama masih hidup
-- dan masih baca kolom `name`. Kalau langsung dihapus, aplikasi mati.
ALTER TABLE users ADD COLUMN first_name TEXT;
ALTER TABLE users ADD COLUMN last_name  TEXT;

-- Backfill data lama. Migration itu bukan cuma soal struktur,
-- tapi juga soal mindahin data.
UPDATE users
SET first_name = split_part(name, ' ', 1),
    last_name  = NULLIF(substr(name, strpos(name, ' ') + 1), name);
