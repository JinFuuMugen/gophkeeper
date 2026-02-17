-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE users
ADD COLUMN IF NOT EXISTS kdf_salt bytea NOT NULL DEFAULT gen_random_bytes(16);

-- +goose Down
ALTER TABLE users
DROP COLUMN IF EXISTS kdf_salt;