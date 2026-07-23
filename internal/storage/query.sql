-- name: CreateLink :one
INSERT INTO links (code, original_url)
VALUES ($1, $2)
RETURNING *;

-- name: GetLink :one
SELECT * FROM links WHERE code = $1;

-- name: IncrClickStat :exec
INSERT INTO click_stats (code, dimension, value, count)
VALUES ($1, $2, $3, 1)
ON CONFLICT (code, dimension, value)
DO UPDATE SET count = click_stats.count + 1;

-- name: ListClickStats :many
SELECT dimension, value, count
FROM click_stats
WHERE code = $1
ORDER BY dimension, count DESC;
