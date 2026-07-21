-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS admin_email_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_user_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admin_email_codes_admin_user_id
    ON admin_email_codes (admin_user_id, created_at DESC);

DROP INDEX IF EXISTS idx_admin_recovery_codes_admin_user_id;
DROP TABLE IF EXISTS admin_recovery_codes;

ALTER TABLE admin_users
    DROP COLUMN IF EXISTS totp_secret_nonce,
    DROP COLUMN IF EXISTS totp_secret_enc,
    DROP COLUMN IF EXISTS totp_enabled;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE admin_users
    ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS totp_secret_enc BYTEA,
    ADD COLUMN IF NOT EXISTS totp_secret_nonce BYTEA;

CREATE TABLE IF NOT EXISTS admin_recovery_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_user_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admin_recovery_codes_admin_user_id
    ON admin_recovery_codes (admin_user_id);

DROP INDEX IF EXISTS idx_admin_email_codes_admin_user_id;
DROP TABLE IF EXISTS admin_email_codes;

-- +goose StatementEnd
