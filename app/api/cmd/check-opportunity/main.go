package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run ./cmd/check-opportunity <notice-id>")
	}

	noticeID := os.Args[1]

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

	var title, postedDate string
	err = pool.QueryRow(ctx, 
		"SELECT title, posted_date FROM opportunity WHERE notice_id = $1",
		noticeID,
	).Scan(&title, &postedDate)

	if err != nil {
		fmt.Printf("❌ Opportunity %s not found in database\n", noticeID)
		os.Exit(1)
	}

	fmt.Printf("✅ Found opportunity:\n")
	fmt.Printf("   Notice ID: %s\n", noticeID)
	fmt.Printf("   Title: %s\n", title)
	fmt.Printf("   Posted: %s\n", postedDate)
}

