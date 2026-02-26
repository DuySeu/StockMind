-- name: GetResearchReports :many
SELECT id, ticker, recommendation, price, created_at FROM research ORDER BY created_at DESC;
-- name: GetResearchReportById :one
SELECT report FROM research WHERE id = $1;
-- name: CreateResearchReport :one
INSERT INTO research (ticker, recommendation, price, report) VALUES ($1, $2, $3, $4) RETURNING *;
-- name: UpdateResearchReport :one
UPDATE research SET recommendation = $2, price = $3, report = $4 WHERE id = $1 RETURNING *;
