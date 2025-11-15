-- name: CreateToken :one
INSERT INTO tokens (user_id, name, scopes, token_type, jti, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, name, scopes, token_type, jti, expires_at, revoked, last_used_at, created_at, updated_at;

-- name: ListTokensByUser :many
SELECT id, user_id, name, scopes, token_type, jti, expires_at, revoked, last_used_at, created_at, updated_at
FROM tokens
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetTokenByID :one
SELECT id, user_id, name, scopes, token_type, jti, expires_at, revoked, last_used_at, created_at, updated_at
FROM tokens
WHERE id = $1
  AND user_id = $2;

-- name: GetTokenByJTI :one
SELECT id, user_id, name, scopes, token_type, jti, expires_at, revoked, last_used_at, created_at, updated_at
FROM tokens
WHERE jti = $1;

-- name: RevokeToken :one
UPDATE tokens
SET revoked = TRUE,
    updated_at = now()
WHERE id = $1
  AND user_id = $2
RETURNING id, user_id, name, scopes, token_type, jti, expires_at, revoked, last_used_at, created_at, updated_at;

-- name: MarkTokenUsed :exec
UPDATE tokens
SET last_used_at = now(),
    updated_at = now()
WHERE id = $1;
