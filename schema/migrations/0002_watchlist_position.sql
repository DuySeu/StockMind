-- Manual ordering for the watchlist board
-- +goose Up
ALTER TABLE watchlist ADD COLUMN IF NOT EXISTS position INTEGER NOT NULL DEFAULT 0;

-- Seed the order the board already shows, newest first, so nothing moves on deploy
-- +goose StatementBegin
UPDATE watchlist AS w
SET position = ordered.rank
FROM (
    SELECT id, ROW_NUMBER() OVER (ORDER BY created_at DESC) AS rank
    FROM watchlist
) AS ordered
WHERE w.id = ordered.id;
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS watchlist_position_idx ON watchlist (position);
