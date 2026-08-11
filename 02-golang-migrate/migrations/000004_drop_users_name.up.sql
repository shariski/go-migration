-- Tahap CONTRACT.
--
-- PENTING: ini TIDAK boleh ikut deploy yang sama dengan 000003.
-- Urutannya: deploy 000003 -> update semua service supaya pakai
-- first_name/last_name -> pastikan nggak ada lagi yang baca `name`
-- -> baru deploy migration ini.
ALTER TABLE users DROP COLUMN name;
