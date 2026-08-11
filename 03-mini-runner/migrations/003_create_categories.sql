CREATE TABLE categories (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

ALTER TABLE products ADD COLUMN category_id BIGINT REFERENCES categories(id);
