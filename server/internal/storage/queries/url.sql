-- name: CreateURL :one
INSERT INTO urls (
    id,
    short_code,
    long_url,
    long_url_hash,
    user_id,
    is_custom_alias,
    expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: GetURLByShortCode :one
SELECT *
FROM urls
WHERE short_code = $1;

-- name: GetURLByUserAndHash :one
SELECT *
FROM urls
WHERE user_id = $1
AND long_url_hash = $2;
