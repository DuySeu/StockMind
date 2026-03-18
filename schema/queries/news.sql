-- name: GetLatestNews :many
SELECT * FROM news
WHERE DATE(created_at) = DATE($1::timestamptz)
ORDER BY created_at DESC;

-- name: SaveNews :one
INSERT INTO news (
    title, url, description
) VALUES (
    $1, $2, $3
) RETURNING *;
