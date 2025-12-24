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

	// Test the exact query the repository would use
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM opportunity 
		WHERE posted_date >= $1 AND posted_date <= $2
	`, "2025-12-01", "2025-12-23").Scan(&count)
	if err != nil {
		log.Fatal("Failed to query:", err)
	}

	fmt.Printf("✅ Query result: %d opportunities\n", count)

	// Also test with a sample to see what dates we're comparing
	rows, err := pool.Query(ctx, `
		SELECT posted_date FROM opportunity 
		WHERE posted_date >= $1 AND posted_date <= $2
		LIMIT 5
	`, "2025-12-01", "2025-12-23")
	if err != nil {
		log.Fatal("Failed to query:", err)
	}
	defer rows.Close()

	fmt.Println("\n📅 Sample dates in range:")
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			continue
		}
		fmt.Printf("   %s\n", date)
	}
}

