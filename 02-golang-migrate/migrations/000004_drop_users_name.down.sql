-- Inilah kenapa "selalu tulis down migration" itu setengah bohong.
--
-- Kita bisa mengembalikan KOLOM-nya:
ALTER TABLE users ADD COLUMN name TEXT;

-- ...tapi datanya sudah hilang permanen waktu kolomnya di-DROP.
-- Yang di bawah ini cuma rekonstruksi tebak-tebakan, bukan data asli.
-- Kalau first_name/last_name juga sudah dihapus, bahkan ini pun gagal.
UPDATE users SET name = concat_ws(' ', first_name, last_name);
