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
    id VARCHAR(50) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    provider VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Threads table
CREATE TABLE IF NOT EXISTS threads (
    id VARCHAR(50) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(50) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    content JSONB NOT NULL,
    thread_id VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);