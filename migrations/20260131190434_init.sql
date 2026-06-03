-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS licenses (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
    lookup_digest BYTEA NOT NULL,
    key_phc TEXT NOT NULL,
    customer_email TEXT NOT NULL,
    max_activations INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS licenses_lookup_digest_uq
    ON licenses (lookup_digest);

CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id INTEGER NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    hwid_hash BYTEA NOT NULL,
    hostname TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (license_id, hwid_hash)
);

CREATE INDEX IF NOT EXISTS idx_devices_license_id
    ON devices (license_id);

CREATE INDEX IF NOT EXISTS idx_devices_hwid_hash
    ON devices (hwid_hash);

CREATE TABLE IF NOT EXISTS activations (
    id SERIAL PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    license_id INTEGER NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,

    checked_in_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_activations_device_id
    ON activations (device_id);

CREATE INDEX IF NOT EXISTS idx_activations_license_id
    ON activations (license_id);

CREATE INDEX IF NOT EXISTS idx_activations_checked_in_at
    ON activations (checked_in_at);

CREATE TABLE IF NOT EXISTS self_service_tokens (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    purpose TEXT NOT NULL DEFAULT 'selfservice_login',
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_ip INET,
    user_agent TEXT
);

CREATE INDEX IF NOT EXISTS idx_self_service_tokens_email
    ON self_service_tokens (email);

CREATE INDEX IF NOT EXISTS idx_self_service_tokens_expires_at
    ON self_service_tokens (expires_at);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_self_service_tokens_expires_at;
DROP INDEX IF EXISTS idx_self_service_tokens_email;
DROP TABLE IF EXISTS self_service_tokens;

DROP INDEX IF EXISTS idx_activations_checked_in_at;
DROP INDEX IF EXISTS idx_activations_license_id;
DROP INDEX IF EXISTS idx_activations_device_id;
DROP TABLE IF EXISTS activations;

DROP INDEX IF EXISTS idx_devices_hwid_hash;
DROP INDEX IF EXISTS idx_devices_license_id;
DROP TABLE IF EXISTS devices;

DROP INDEX IF EXISTS licenses_lookup_digest_uq;
DROP TABLE IF EXISTS licenses;

DROP TABLE IF EXISTS products;

DROP EXTENSION IF EXISTS pgcrypto;

-- +goose StatementEnd
