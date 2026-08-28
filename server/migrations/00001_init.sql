-- +goose Up

CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    api_key_hash TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE urls (
    id BIGINT PRIMARY KEY,
    short_code TEXT NOT NULL,
    long_url TEXT NOT NULL,
    long_url_hash CHAR(64) NOT NULL,
    user_id BIGINT REFERENCES users(id),
    is_custom_alias BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX idx_urls_short_code
ON urls(short_code);

CREATE UNIQUE INDEX idx_urls_user_long_url_hash
ON urls(user_id, long_url_hash)
WHERE user_id IS NOT NULL;

CREATE INDEX idx_urls_expires_at
ON urls(expires_at)
WHERE expires_at IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_urls_expires_at;
DROP INDEX IF EXISTS idx_urls_user_long_url_hash;
DROP INDEX IF EXISTS idx_urls_short_code;

DROP TABLE urls;
DROP TABLE users;
