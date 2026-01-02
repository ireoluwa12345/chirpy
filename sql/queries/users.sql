-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, password)
VALUES (
    $1, NOW(), NOW(), $2, $3
)
RETURNING *;