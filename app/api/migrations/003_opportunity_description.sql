-- Migration: Add opportunity_description table for fetching and normalizing SAM opportunity descriptions
-- Run with: psql "$DATABASE_URL" -f migrations/003_opportunity_description.sql

-- Create opportunity_description table
CREATE TABLE IF NOT EXISTS opportunity_description (
    notice_id VARCHAR PRIMARY KEY REFERENCES opportunity(notice_id) ON DELETE CASCADE,
    source_type VARCHAR NOT NULL CHECK (source_type IN ('none', 'inline', 'url')),
    source_url TEXT,
    source_inline TEXT,
    fetch_status VARCHAR NOT NULL CHECK (fetch_status IN ('not_requested', 'fetched', 'not_found', 'error')),
    http_status INT,
    fetched_at TIMESTAMPTZ,
    raw_text TEXT,
    raw_text_normalized TEXT,
    text_normalized TEXT,
    content_hash TEXT,
    content_type TEXT,
    last_error TEXT,
    -- Future fields for OpenAI summarization (nullable, ready for future use)
    brief_summary TEXT,
    brief_summary_model TEXT,
    brief_summary_hash TEXT,
    summary_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_opportunity_description_notice_id 
    ON opportunity_description(notice_id);

CREATE INDEX IF NOT EXISTS idx_opportunity_description_fetch_status 
    ON opportunity_description(fetch_status);

CREATE INDEX IF NOT EXISTS idx_opportunity_description_content_hash 
    ON opportunity_description(content_hash) 
    WHERE content_hash IS NOT NULL;

-- Add comment to table
COMMENT ON TABLE opportunity_description IS 'Stores fetched and normalized descriptions for SAM opportunities. Supports lazy fetching, three-tier normalization, and future OpenAI summarization.';

