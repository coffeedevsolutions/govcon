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

	// Check date formats in database
	fmt.Println("📅 Date formats in database (sample):")
	rows, err := pool.Query(ctx, "SELECT DISTINCT posted_date FROM opportunity WHERE posted_date IS NOT NULL ORDER BY posted_date DESC LIMIT 10")
	if err != nil {
		log.Fatal("Failed to query:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var postedDate string
		if err := rows.Scan(&postedDate); err != nil {
			continue
		}
		fmt.Printf("   %s\n", postedDate)
	}

	// Check count with date filter
	fmt.Println("\n🔍 Testing date filter (12/01/2025 to 12/23/2025):")
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM opportunity 
		WHERE posted_date >= $1 AND posted_date <= $2
	`, "12/01/2025", "12/23/2025").Scan(&count)
	if err != nil {
		log.Fatal("Failed to count:", err)
	}
	fmt.Printf("   Count: %d\n", count)

	// Check total count
	var total int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM opportunity").Scan(&total)
	if err != nil {
		log.Fatal("Failed to count total:", err)
	}
	fmt.Printf("\n📊 Total opportunities in database: %d\n", total)

	// Check without date filter
	var countNoFilter int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM opportunity WHERE posted_date IS NOT NULL").Scan(&countNoFilter)
	if err != nil {
		log.Fatal("Failed to count:", err)
	}
	fmt.Printf("   Opportunities with posted_date: %d\n", countNoFilter)
}

