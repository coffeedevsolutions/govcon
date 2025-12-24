package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer pool.Close()

	// Create the ping table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS ping (
			id SERIAL PRIMARY KEY,
			message TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	log.Println("✅ Created ping table")

	// Insert initial message if table is empty
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ping").Scan(&count)
	if err != nil {
		log.Fatal("Failed to count rows:", err)
	}

	if count == 0 {
		_, err = pool.Exec(ctx, "INSERT INTO ping (message) VALUES ($1)", "hello from postgres")
		if err != nil {
			log.Fatal("Failed to insert initial message:", err)
		}
		log.Println("✅ Inserted initial message")
	} else {
		log.Printf("✅ Table already has %d row(s), skipping insert", count)
	}

	// Enable pg_trgm extension for fuzzy text matching
	_, err = pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_trgm;`)
	if err != nil {
		log.Fatal("Failed to enable pg_trgm extension:", err)
	}
	log.Println("✅ Enabled pg_trgm extension")

	// Create opportunity_raw table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS opportunity_raw (
			id SERIAL PRIMARY KEY,
			notice_id VARCHAR NOT NULL UNIQUE,
			raw_data JSONB NOT NULL,
			fetched_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	if err != nil {
		log.Fatal("Failed to create opportunity_raw table:", err)
	}
	log.Println("✅ Created opportunity_raw table")

	// Create indexes on opportunity_raw
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunity_raw_notice_id ON opportunity_raw(notice_id);`)
	if err != nil {
		log.Fatal("Failed to create index on opportunity_raw:", err)
	}
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunity_raw_fetched_at ON opportunity_raw(fetched_at);`)
	if err != nil {
		log.Fatal("Failed to create index on opportunity_raw:", err)
	}

	// Create opportunity table with all normalized columns
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS opportunity (
			notice_id VARCHAR PRIMARY KEY,
			title TEXT NOT NULL,
			organization_type VARCHAR,
			posted_date VARCHAR,
			type VARCHAR,
			base_type VARCHAR,
			archive_type VARCHAR,
			archive_date VARCHAR,
			type_of_set_aside VARCHAR,
			type_of_set_aside_desc VARCHAR,
			response_deadline VARCHAR,
			naics JSONB,
			classification_code VARCHAR,
			active BOOLEAN NOT NULL DEFAULT false,
			point_of_contact JSONB,
			place_of_performance JSONB,
			description TEXT,
			department VARCHAR,
			sub_tier VARCHAR,
			office VARCHAR,
			links JSONB,
			content_hash VARCHAR NOT NULL,
			last_updated TIMESTAMPTZ NOT NULL DEFAULT now(),
			first_seen TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	if err != nil {
		log.Fatal("Failed to create opportunity table:", err)
	}
	log.Println("✅ Created opportunity table")

	// Create indexes on opportunity table
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunity_posted_date ON opportunity(posted_date);`)
	if err != nil {
		log.Fatal("Failed to create index on opportunity:", err)
	}
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunity_response_deadline ON opportunity(response_deadline);`)
	if err != nil {
		log.Fatal("Failed to create index on opportunity:", err)
	}
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunity_active ON opportunity(active);`)
	if err != nil {
		log.Fatal("Failed to create index on opportunity:", err)
	}
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunity_content_hash ON opportunity(content_hash);`)
	if err != nil {
		log.Fatal("Failed to create index on opportunity:", err)
	}

	// Create GIN full-text search index on concatenated search document
	_, err = pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_opportunity_search_gin 
		ON opportunity USING GIN (
			to_tsvector('english', 
				COALESCE(title, '') || ' ' || 
				COALESCE(department, '') || ' ' || 
				COALESCE(description, '')
			)
		);
	`)
	if err != nil {
		log.Fatal("Failed to create GIN search index:", err)
	}
	log.Println("✅ Created GIN full-text search index")

	// Create pg_trgm indexes for fuzzy matching
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunity_title_trgm ON opportunity USING GIN (title gin_trgm_ops);`)
	if err != nil {
		log.Fatal("Failed to create pg_trgm index on title:", err)
	}
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunity_department_trgm ON opportunity USING GIN (department gin_trgm_ops);`)
	if err != nil {
		log.Fatal("Failed to create pg_trgm index on department:", err)
	}
	log.Println("✅ Created pg_trgm indexes for fuzzy matching")

	// Create opportunity_version table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS opportunity_version (
			id SERIAL PRIMARY KEY,
			notice_id VARCHAR NOT NULL REFERENCES opportunity(notice_id) ON DELETE CASCADE,
			content_hash VARCHAR NOT NULL,
			raw_snapshot JSONB NOT NULL,
			fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			changed_fields JSONB
		);
	`)
	if err != nil {
		log.Fatal("Failed to create opportunity_version table:", err)
	}
	log.Println("✅ Created opportunity_version table")

	// Create indexes on opportunity_version
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunity_version_notice_id ON opportunity_version(notice_id);`)
	if err != nil {
		log.Fatal("Failed to create index on opportunity_version:", err)
	}
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunity_version_fetched_at ON opportunity_version(fetched_at);`)
	if err != nil {
		log.Fatal("Failed to create index on opportunity_version:", err)
	}

	log.Println("✅ Database setup complete!")
}

