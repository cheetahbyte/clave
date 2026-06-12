-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS organization_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'admin',
    token_hash TEXT NOT NULL UNIQUE,
    invited_by UUID REFERENCES admin_users(id) ON DELETE SET NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_organization_invites_org_id
    ON organization_invites (organization_id);

-- One pending invite per email per org.
CREATE UNIQUE INDEX IF NOT EXISTS uq_organization_invites_pending
    ON organization_invites (organization_id, lower(email))
    WHERE accepted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS uq_organization_invites_pending;
DROP INDEX IF EXISTS idx_organization_invites_org_id;
DROP TABLE IF EXISTS organization_invites;

-- +goose StatementEnd
