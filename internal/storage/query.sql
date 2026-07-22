-- name: CreateLink :one
INSERT INTO links (code, original_url)
VALUES ($1, $2)
RETURNING *;

-- name: GetLink :one
SELECT * FROM links WHERE code = $1;
