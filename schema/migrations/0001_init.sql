-- Database initialization script for PostgreSQL Docker Compose
-- This file will be executed automatically when the PostgreSQL container starts
-- Set client encoding and timezone
-- +goose Up
-- =============================================
-- SCHEMA CREATION AND UTILITY FUNCTIONS
-- =============================================
-- Function to automatically update the updated_at timestamp
-- +goose statementbegin
CREATE OR REPLACE FUNCTION trigger_set_timestamp () RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW(); -- Sets updated_at to the current transaction timestamp
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose statementend

-- =============================================
-- TABLE CREATION
-- =============================================

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    provider VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Agent flow table
CREATE TABLE IF NOT EXISTS agent_flows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    config JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Conversations table
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    metadata JSONB DEFAULT '{}', -- Store summary of the conversation
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}', -- Store addtional information about the message such as tool calls, tool results, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Research table
CREATE TABLE IF NOT EXISTS research (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(255) NOT NULL,
    recommendation VARCHAR(255) NOT NULL,
    reference_price VARCHAR(255) NOT NULL,
    report JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Watchlist table
CREATE TABLE IF NOT EXISTS watchlist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS news (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    url VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS documents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(500) NOT NULL,
    file_type   VARCHAR(10) NOT NULL CHECK (file_type IN ('pdf', 'docx', 'md', 'txt')),
    size_bytes  BIGINT NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'processing', 'ready', 'failed')),
    chunk_count INT4 NOT NULL DEFAULT 0,
    strategy    VARCHAR(20) NOT NULL DEFAULT 'semantic'
                    CHECK (strategy IN ('recursive', 'fixed', 'paragraph', 'semantic')),
    error_msg   TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TRIGGER set_documents_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW EXECUTE PROCEDURE trigger_set_timestamp();

DROP TRIGGER IF EXISTS set_documents_updated_at ON documents;

INSERT INTO watchlist (ticker) VALUES ('FPT'), ('DGC'), ('VRE'), ('VCG'), ('VPB');

-- Default Agent Flow
INSERT INTO agent_flows (id, name, config) VALUES
('01993ca8-a62e-79e3-995c-a46e25a4a2a2', 'OpenAI Flow', 
'{
    "agents": {
        "NormalChat": {
            "provider": "openai",
            "description": "Handles general chat interactions.",
            "modelId": "nvidia/nemotron-3-super-120b-a12b:free",
            "systemPrompt": "You are a helpful assistant.",
            "maxTokens": 8192,
            "thinkingToken": 1024,
            "temperature": 0.7,
            "topP": 0.9,
            "topK": 40,
            "tools": [],
            "mcpServers": [
                {
                    "name": "stocks-mcp",
                    "protocol": "streamablehttp",
                    "url": "http://localhost:8081/mcp"
                }
            ]
        }
    },
    "nodes": [
        {
            "id": "start",
            "type": "start",
            "next": "NormalChat"
        },
        {
            "id": "NormalChat",
            "type": "agent",
            "agentName": "NormalChat",
            "next": "end",
            "output": {
                "type": "text",
                "contentFormat": "{{.message}}",
                "contentRole": "assistant"
            }
        }
    ]
}'),
('01993ca8-a62e-79e3-995c-a46e25a4a2a4', 'Anthropic Flow', 
'{
    "agents": {
        "NormalChat": {
            "provider": "anthropic",
            "description": "Handles general chat interactions.",
            "modelId": "nvidia/nemotron-3-super-120b-a12b:free",
            "systemPrompt": "You are a helpful assistant.",
            "maxTokens": 8192,
            "thinkingToken": 1024,
            "temperature": 0.7,
            "topP": 0.9,
            "topK": 40,
            "tools": [],
            "mcpServers": [
                {
                    "name": "stocks-mcp",
                    "protocol": "streamablehttp",
                    "url": "http://localhost:8081/mcp"
                }
            ]
        }
    },
    "nodes": [
        {
            "id": "start",
            "type": "start",
            "next": "NormalChat"
        },
        {
            "id": "NormalChat",
            "type": "agent",
            "agentName": "NormalChat",
            "next": "end",
            "output": {
                "type": "text",
                "contentFormat": "{{.message}}",
                "contentRole": "assistant"
            }
        }
    ]
}'),
('01993ca8-a62e-79e3-995c-a46e25a4a2a3', 'Multiple Agents Flow',
'{
    "agents": {
        "StockPrice": {
            "provider": "openai",
            "description": "Handles general chat interactions.",
            "modelId": "nvidia/nemotron-3-nano-30b-a3b:free",
            "systemPrompt": "You are a helpful assistant. Your task is to get the stock price based on the given stock symbol.",
            "maxTokens": 8192,
            "thinkingToken": 1024,
            "temperature": 0.7,
            "topP": 0.9,
            "topK": 40,
            "tools": [],
            "mcpServers": [
                {
                    "name": "stocks-mcp",
                    "protocol": "streamablehttp",
                    "url": "http://localhost:8081/mcp"
                }
            ]
        },
        "StockReport": {
            "provider": "openai",
            "description": "Handles general chat interactions.",
            "modelId": "nvidia/nemotron-3-nano-30b-a3b:free",
            "systemPrompt": "You are a helpful assistant. Your task is to generate a stock report based on the given stock price.",
            "maxTokens": 8192,
            "thinkingToken": 1024,
            "temperature": 0.7,
            "topP": 0.9,
            "topK": 40,
            "tools": [],
            "mcpServers": []
        }
    },
    "nodes": [
        {
            "id": "start",
            "type": "start",
            "next": "StockPrice"
        },
        {
            "id": "StockPrice",
            "type": "agent",
            "agentName": "StockPrice",
            "next": "StockReport",
            "output": {
                "type": "text",
                "contentFormat": "{{.message}}",
                "contentRole": "assistant"
            }
        },
        {
            "id": "StockReport",
            "type": "agent",
            "agentName": "StockReport",
            "next": "end",
            "output": {
                "type": "text",
                "contentFormat": "{{.message}}",
                "contentRole": "assistant"
            }
        }
    ]
}');