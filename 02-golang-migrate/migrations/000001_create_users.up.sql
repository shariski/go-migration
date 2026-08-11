CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Data awal, khusus buat demo. Di repo beneran, seed data biasanya
-- dipisah dari migration struktur.
INSERT INTO users (name, email) VALUES
    ('Budi Santoso', 'budi@contoh.id'),
    ('Siti Rahayu',  'siti@contoh.id'),
    ('Agus Wijaya',  'agus@contoh.id');
