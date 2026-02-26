-- name: GetWatchlist :many
SELECT * FROM watchlist ORDER BY created_at DESC;
-- name: CreateWatchlistData :one
INSERT INTO watchlist (ticker) VALUES ($1) RETURNING *;
-- name: DeleteWatchlistData :one
DELETE FROM watchlist WHERE id = $1 RETURNING *;
