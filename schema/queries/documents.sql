-- name: CreateDocument :one
INSERT INTO documents (id, name, file_type, size_bytes, strategy)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDocumentByID :one
SELECT * FROM documents WHERE id = $1;

-- name: ListDocuments :many
SELECT * FROM documents ORDER BY created_at DESC;

-- name: UpdateDocumentStatus :exec
UPDATE documents
SET status = $2, chunk_count = $3, error_msg = $4
WHERE id = $1;

-- name: DeleteDocument :exec
DELETE FROM documents WHERE id = $1;
