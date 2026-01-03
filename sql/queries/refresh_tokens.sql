-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token, expires_at, created_at, updated_at, revoked)
VALUES (
    $1, $2, $3, NOW(), NOW(), NULL
)
RETURNING *;

-- name: CheckRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1 AND revoked IS NULL AND expires_at > NOW();

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked = NOW(), updated_at = NOW()
WHERE token = $1;