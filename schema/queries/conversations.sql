-- name: CreateConversation :one
INSERT INTO conversations (id, user_id, title, description, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- name: GetConversationsByUserID :many
SELECT * FROM conversations WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CreateMessage :exec
INSERT INTO messages (id, conversation_id, content, role, metadata) VALUES ($1, $2, $3, $4, $5);

-- name: GetMessagesByConversationID :many
SELECT * FROM messages WHERE conversation_id = $1 ORDER BY created_at ASC;

-- name: UpdateConversationName :exec
UPDATE conversations SET title = $2 WHERE id = $1;

-- name: DeleteConversationByID :exec
DELETE FROM conversations WHERE id = $1;
