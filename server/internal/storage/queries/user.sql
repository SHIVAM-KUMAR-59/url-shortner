-- name: CreateUser :one
INSERT INTO users (id, email, api_key_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByAPIKeyHash :one
SELECT * FROM users WHERE api_key_hash = $1;