-- name: CreateSession :one
INSERT INTO conversations (id, user_id, title, description, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- name: GetSessionsByUserID :many
SELECT * FROM conversations WHERE user_id = $1 ORDER BY created_at DESC;

-- name: SessionAddChatHistory :exec
INSERT INTO messages (id, conversation_id, content, role, metadata) VALUES ($1, $2, $3, $4, $5);

-- name: GetSessionHistoryBySessionID :many
SELECT * FROM messages WHERE conversation_id = $1 ORDER BY created_at ASC;

-- name: UpdateSessionName :exec
UPDATE conversations SET title = $2 WHERE id = $1;

-- name: DeleteSessionByID :exec
DELETE FROM conversations WHERE id = $1;