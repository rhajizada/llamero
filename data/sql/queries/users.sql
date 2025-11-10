-- name: UpsertUser :one
INSERT INTO users (sub, provider, email, display_name, role, scopes, groups, last_login_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())
ON CONFLICT (provider, sub)
DO UPDATE SET
    email = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    role = EXCLUDED.role,
    scopes = EXCLUDED.scopes,
    groups = EXCLUDED.groups,
    last_login_at = EXCLUDED.last_login_at,
    updated_at = now()
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByProviderSub :one
SELECT * FROM users WHERE provider = $1 AND sub = $2;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
