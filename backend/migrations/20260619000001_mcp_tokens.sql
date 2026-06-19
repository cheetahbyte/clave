-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS mcp_tokens (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    token_id TEXT NOT NULL UNIQUE,
    token_hash BYTEA NOT NULL,
    token_prefix TEXT NOT NULL,
    created_by UUID REFERENCES admin_users(id) ON DELETE SET NULL,
    regenerated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_mcp_tokens_token_id ON mcp_tokens (token_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_mcp_tokens_token_id;
DROP TABLE IF EXISTS mcp_tokens;

-- +goose StatementEnd
