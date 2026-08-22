-- +goose Up
-- +goose StatementBegin

CREATE INDEX IF NOT EXISTS idx_self_service_tokens_used
    ON self_service_tokens (used_at)
    WHERE used_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_organization_invites_expiry
    ON organization_invites (expires_at);

CREATE INDEX IF NOT EXISTS idx_organization_invites_accepted
    ON organization_invites (accepted_at)
    WHERE accepted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_email_codes_expiry
    ON admin_email_codes (expires_at);

CREATE INDEX IF NOT EXISTS idx_admin_email_codes_used
    ON admin_email_codes (used_at)
    WHERE used_at IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_admin_email_codes_used;
DROP INDEX IF EXISTS idx_admin_email_codes_expiry;
DROP INDEX IF EXISTS idx_organization_invites_accepted;
DROP INDEX IF EXISTS idx_organization_invites_expiry;
DROP INDEX IF EXISTS idx_self_service_tokens_used;

-- +goose StatementEnd
