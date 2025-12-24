-- Migration: Add search columns and indexes for fast filtering and full-text search
-- Run with: psql "$DATABASE_URL" -f migrations/002_search_indexes.sql

-- Enable pg_trgm extension if not already enabled (for fuzzy matching)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Add solicitation_number column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'opportunity' AND column_name = 'solicitation_number'
    ) THEN
        ALTER TABLE opportunity ADD COLUMN solicitation_number VARCHAR;
    END IF;
END $$;

-- Add agency_path_name column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'opportunity' AND column_name = 'agency_path_name'
    ) THEN
        ALTER TABLE opportunity ADD COLUMN agency_path_name VARCHAR;
    END IF;
END $$;

-- Backfill solicitation_number from opportunity_raw.raw_data
UPDATE opportunity o
SET solicitation_number = (
    SELECT raw_data->>'solicitationNumber'
    FROM opportunity_raw r
    WHERE r.notice_id = o.notice_id
    AND raw_data->>'solicitationNumber' IS NOT NULL
    AND raw_data->>'solicitationNumber' != ''
)
WHERE solicitation_number IS NULL;

-- Backfill agency_path_name from opportunity_raw.raw_data
UPDATE opportunity o
SET agency_path_name = (
    SELECT raw_data->>'fullParentPathName'
    FROM opportunity_raw r
    WHERE r.notice_id = o.notice_id
    AND raw_data->>'fullParentPathName' IS NOT NULL
    AND raw_data->>'fullParentPathName' != ''
)
WHERE agency_path_name IS NULL;

-- Create generated stored tsvector column for full-text search
-- This includes title, solicitation_number, agency_path_name, and description
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'opportunity' AND column_name = 'search_tsv'
    ) THEN
        ALTER TABLE opportunity ADD COLUMN search_tsv tsvector 
            GENERATED ALWAYS AS (
                to_tsvector('english',
                    COALESCE(title, '') || ' ' ||
                    COALESCE(solicitation_number, '') || ' ' ||
                    COALESCE(agency_path_name, '') || ' ' ||
                    COALESCE(description, '')
                )
            ) STORED;
    END IF;
END $$;

-- Create GIN index on search_tsv for fast full-text search
CREATE INDEX IF NOT EXISTS idx_opportunity_search_tsv 
    ON opportunity USING GIN (search_tsv);

-- Create B-tree indexes for filter columns
CREATE INDEX IF NOT EXISTS idx_opportunity_type_of_set_aside 
    ON opportunity(type_of_set_aside) 
    WHERE type_of_set_aside IS NOT NULL;

-- Create GIN index on naics JSONB for array queries
CREATE INDEX IF NOT EXISTS idx_opportunity_naics_gin 
    ON opportunity USING GIN (naics);

-- Create index on agency_path_name for prefix/ILIKE matching
CREATE INDEX IF NOT EXISTS idx_opportunity_agency_path_name 
    ON opportunity(agency_path_name) 
    WHERE agency_path_name IS NOT NULL;

-- Create index on state extracted from place_of_performance JSONB
-- This uses a functional index for JSONB queries
CREATE INDEX IF NOT EXISTS idx_opportunity_pop_state 
    ON opportunity((place_of_performance->>'state')) 
    WHERE place_of_performance->>'state' IS NOT NULL;

-- Create trigram GIN indexes for fuzzy matching on title and solicitation_number
CREATE INDEX IF NOT EXISTS idx_opportunity_title_trgm_v2 
    ON opportunity USING GIN (title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_opportunity_solicitation_number_trgm 
    ON opportunity USING GIN (solicitation_number gin_trgm_ops)
    WHERE solicitation_number IS NOT NULL;

-- Composite index for common filter combinations (naics + state + set_aside)
-- This can help with multi-filter queries
CREATE INDEX IF NOT EXISTS idx_opportunity_filters_composite 
    ON opportunity(type_of_set_aside, (place_of_performance->>'state'))
    WHERE type_of_set_aside IS NOT NULL 
    AND place_of_performance->>'state' IS NOT NULL;

-- Note: posted_date and response_deadline indexes already exist from setup-db
-- Verify they exist, create if missing
CREATE INDEX IF NOT EXISTS idx_opportunity_posted_date 
    ON opportunity(posted_date);

CREATE INDEX IF NOT EXISTS idx_opportunity_response_deadline 
    ON opportunity(response_deadline);

