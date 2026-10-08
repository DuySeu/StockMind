-- name: GetWatchlist :many
SELECT * FROM watchlist ORDER BY position, created_at DESC LIMIT NULLIF($1::int, 0);
-- name: CreateWatchlistData :one
INSERT INTO watchlist (ticker, position)
VALUES ($1, (SELECT COALESCE(MAX(position), 0) + 1 FROM watchlist))
RETURNING *;
-- name: SetWatchlistPosition :exec
UPDATE watchlist SET position = $2 WHERE id = $1;
-- name: DeleteWatchlistData :one
DELETE FROM watchlist WHERE id = $1 RETURNING *;
