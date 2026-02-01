-- name: CreateSelfServiceLink :one
insert into self_service_tokens (
    id,
    email,
    token_hash,
    purpose,
    expires_at,
    created_ip,
    user_agent
)
values (
    gen_random_uuid(),
    sqlc.arg(email),
    sqlc.arg(token_hash),
    coalesce(sqlc.arg(purpose), 'selfservice_login'),
    sqlc.arg(expires_at),
    sqlc.arg(created_ip),
    sqlc.arg(user_agent)
)
returning
    id,
    email,
    token_hash,
    purpose,
    expires_at,
    used_at,
    created_at,
    created_ip,
    user_agent;
-- name: ConsumeSelfServiceToken :one
update self_service_tokens
set used_at = now()
where token_hash = sqlc.arg(token_hash)
    and expires_at > now()
    and used_at is null
returning email;
