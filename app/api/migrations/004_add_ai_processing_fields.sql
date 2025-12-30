-- Migration: Add AI processing fields to opportunity_description table
-- Run with: psql "$DATABASE_URL" -f migrations/004_add_ai_processing_fields.sql

-- Add AI processing columns
ALTER TABLE opportunity_description
    ADD COLUMN IF NOT EXISTS ai_input_text TEXT NULL,
    ADD COLUMN IF NOT EXISTS ai_input_hash TEXT NULL,
    ADD COLUMN IF NOT EXISTS ai_input_version INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS ai_generated_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS ai_meta JSONB NULL,
    ADD COLUMN IF NOT EXISTS excerpt_text TEXT NULL,
    ADD COLUMN IF NOT EXISTS poc_email_primary TEXT NULL;

-- Create partial index for backfill queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_desc_needs_ai
    ON opportunity_description (notice_id)
    WHERE ai_input_text IS NULL AND raw_text_normalized IS NOT NULL;

-- Add comment
COMMENT ON COLUMN opportunity_description.ai_input_text IS 'AI-ready cleaned text optimized for contractor-focused analysis';
COMMENT ON COLUMN opportunity_description.ai_input_hash IS 'SHA256 hash of ai_input_text for change detection';
COMMENT ON COLUMN opportunity_description.ai_input_version IS 'Version number for safe backfills when extraction logic improves';
COMMENT ON COLUMN opportunity_description.ai_generated_at IS 'Timestamp when AI text was generated';
COMMENT ON COLUMN opportunity_description.ai_meta IS 'Structured metadata (POCs, URLs, set-aside, certs, etc.) in JSONB format';
COMMENT ON COLUMN opportunity_description.excerpt_text IS 'Short excerpt (800-1200 chars) for list views';
COMMENT ON COLUMN opportunity_description.poc_email_primary IS 'Primary POC email for display and filtering';

