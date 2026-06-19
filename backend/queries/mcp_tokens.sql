-- name: GetMCPTokenByOrganization :one
SELECT * FROM mcp_tokens WHERE organization_id = $1;

-- name: GetMCPTokenByTokenID :one
SELECT * FROM mcp_tokens WHERE token_id = $1;

-- name: UpsertMCPToken :one
INSERT INTO mcp_tokens (organization_id, token_id, token_hash, token_prefix, created_by, regenerated_at, last_used_at)
VALUES ($1, $2, $3, $4, $5, now(), NULL)
ON CONFLICT (organization_id) DO UPDATE
SET token_id = EXCLUDED.token_id,
    token_hash = EXCLUDED.token_hash,
    token_prefix = EXCLUDED.token_prefix,
    created_by = EXCLUDED.created_by,
    regenerated_at = now(),
    last_used_at = NULL
RETURNING *;

-- name: TouchMCPTokenLastUsed :exec
UPDATE mcp_tokens SET last_used_at = now() WHERE organization_id = $1;
