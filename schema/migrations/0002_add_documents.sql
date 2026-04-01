-- +goose Up

CREATE TABLE IF NOT EXISTS documents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(500) NOT NULL,
    file_type   VARCHAR(10) NOT NULL CHECK (file_type IN ('pdf', 'docx', 'md', 'txt')),
    size_bytes  BIGINT NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'processing', 'ready', 'failed')),
    chunk_count INT4 NOT NULL DEFAULT 0,
    strategy    VARCHAR(20) NOT NULL DEFAULT 'recursive'
                    CHECK (strategy IN ('recursive', 'fixed', 'paragraph', 'semantic')),
    error_msg   TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TRIGGER set_documents_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW EXECUTE PROCEDURE trigger_set_timestamp();

-- +goose Down

DROP TRIGGER IF EXISTS set_documents_updated_at ON documents;
DROP TABLE IF EXISTS documents;
