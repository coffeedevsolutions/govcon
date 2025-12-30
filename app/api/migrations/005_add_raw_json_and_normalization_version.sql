-- Migration: Add raw_json_response and normalization_version columns to opportunity_description table
-- Run with: psql "$DATABASE_URL" -f migrations/005_add_raw_json_and_normalization_version.sql

-- Add raw_json_response column to store the complete JSON response from SAM API
ALTER TABLE opportunity_description 
ADD COLUMN IF NOT EXISTS raw_json_response TEXT;

-- Add normalization_version column to track which version of normalization logic was used
ALTER TABLE opportunity_description 
ADD COLUMN IF NOT EXISTS normalization_version INT DEFAULT 1;

-- Update existing records to have normalization_version = 1
UPDATE opportunity_description 
SET normalization_version = 1 
WHERE normalization_version IS NULL;

-- Add comment to columns
COMMENT ON COLUMN opportunity_description.raw_json_response IS 'Stores the complete raw JSON response body from SAM API description request, before any processing';
COMMENT ON COLUMN opportunity_description.normalization_version IS 'Tracks which version of normalization logic was used to process this description. Incremented when normalization code changes.';

