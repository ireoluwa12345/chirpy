-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, user_id, body)
VALUES (
    $1, NOW(), NOW(), $2, $3
)
RETURNING *;

-- name: GetChirps :many
SELECT id, created_at, updated_at, user_id, body
FROM chirps ORDER BY created_at DESC;

-- name: GetChirpByID :one
SELECT id, created_at, updated_at, user_id, body
FROM chirps
WHERE id = $1;

-- name: DeleteChirp :exec
DELETE FROM chirps
WHERE id = $1;