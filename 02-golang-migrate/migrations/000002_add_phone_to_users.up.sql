-- Migration yang aman: nambah kolom nullable.
-- Nggak ngunci tabel lama, nggak bikin data lama invalid.
ALTER TABLE users ADD COLUMN phone TEXT;
