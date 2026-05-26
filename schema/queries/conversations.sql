-- name: CreateConversation :one
INSERT INTO conversations (id, user_id, title, description, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- name: GetConversationsByUserID :many
SELECT * FROM conversations WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetConversationByID :one
SELECT * FROM conversations WHERE id = $1;

-- name: GetConversationWithMessages :one
SELECT
  c.metadata AS conv_metadata,
  COALESCE(
    (SELECT jsonb_agg(sub ORDER BY sub.created_at ASC)
     FROM (
       SELECT id, conversation_id, role, content, metadata, created_at
       FROM messages
       WHERE conversation_id = c.id
       ORDER BY created_at DESC
       LIMIT $2 OFFSET $3
     ) sub
    ), '[]'::jsonb
  )::jsonb AS messages
FROM conversations c
WHERE c.id = $1;

-- name: CreateMessage :exec
INSERT INTO messages (id, conversation_id, content, role, metadata) VALUES ($1, $2, $3, $4, $5);

-- name: GetMessagesByConversationID :many
SELECT * FROM (
  SELECT * FROM messages
  WHERE conversation_id = $1
  ORDER BY created_at DESC
  LIMIT $2 OFFSET $3
) sub ORDER BY created_at ASC;

-- name: GetMessageCountByConversationID :one
SELECT COUNT(*) FROM messages WHERE conversation_id = $1;

-- name: UpdateConversationName :exec
UPDATE conversations SET title = $2 WHERE id = $1;

-- name: UpdateConversationMetadata :exec
UPDATE conversations SET metadata = $2 WHERE id = $1;

-- name: DeleteConversationByID :exec
DELETE FROM conversations WHERE id = $1;
