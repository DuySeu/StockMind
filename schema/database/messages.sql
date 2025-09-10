CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(50) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    content JSONB NOT NULL,
    thread_id VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);