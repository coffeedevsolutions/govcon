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

	fmt.Println("📋 Distinct 'type' values in database:")
	rows, err := pool.Query(ctx, "SELECT DISTINCT type FROM opportunity WHERE type IS NOT NULL ORDER BY type LIMIT 20")
	if err != nil {
		log.Fatal("Failed to query:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var typ string
		if err := rows.Scan(&typ); err != nil {
			continue
		}
		fmt.Printf("   '%s'\n", typ)
	}

	// Check count with ptype=o
	var countO int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM opportunity WHERE type = $1", "o").Scan(&countO)
	if err != nil {
		log.Fatal("Failed to count:", err)
	}
	fmt.Printf("\n🔍 Count where type = 'o': %d\n", countO)

	// Check what ptype=o should actually match
	var countSolicitation int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM opportunity WHERE type = $1", "Solicitation").Scan(&countSolicitation)
	if err != nil {
		log.Fatal("Failed to count:", err)
	}
	fmt.Printf("🔍 Count where type = 'Solicitation': %d\n", countSolicitation)
}

