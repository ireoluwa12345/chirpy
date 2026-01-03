-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, password)
VALUES (
    $1, NOW(), NOW(), $2, $3
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, email, password
FROM users
WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET updated_at = NOW(), email = $1, password = $2
WHERE id = $3
RETURNING *;