-- Начальная схема GophKeeper: пользователи и приватные записи.

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS secrets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id          UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type              SMALLINT    NOT NULL,
    name              TEXT        NOT NULL,
    metadata          TEXT        NOT NULL DEFAULT '',
    encrypted_payload BYTEA       NOT NULL,
    version           BIGINT      NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_secrets_owner ON secrets (owner_id);
