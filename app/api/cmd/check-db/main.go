package main

import (
	"context"
	"fmt"
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

	var total int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM opportunity").Scan(&total)
	if err != nil {
		log.Fatal("Failed to count:", err)
	}

	var rawTotal int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM opportunity_raw").Scan(&rawTotal)
	if err != nil {
		log.Fatal("Failed to count raw:", err)
	}

	var versionTotal int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM opportunity_version").Scan(&versionTotal)
	if err != nil {
		log.Fatal("Failed to count versions:", err)
	}

	fmt.Printf("📊 Database Statistics:\n")
	fmt.Printf("   Opportunities: %d\n", total)
	fmt.Printf("   Raw snapshots: %d\n", rawTotal)
	fmt.Printf("   Versions: %d\n", versionTotal)

	// Show a few sample notice IDs
	rows, err := pool.Query(ctx, "SELECT notice_id, title, posted_date FROM opportunity ORDER BY last_updated DESC LIMIT 5")
	if err != nil {
		log.Fatal("Failed to query:", err)
	}
	defer rows.Close()

	fmt.Printf("\n📋 Recent opportunities:\n")
	for rows.Next() {
		var noticeID, title, postedDate string
		if err := rows.Scan(&noticeID, &title, &postedDate); err != nil {
			continue
		}
		fmt.Printf("   %s: %s (posted: %s)\n", noticeID[:8]+"...", title, postedDate)
	}
}

