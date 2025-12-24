package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"govcon/api/internal/services"
)

const (
	// Advisory lock key for ingestion job
	ingestionLockKey = 1
	// Default rolling window days
	defaultRollingWindowDays = 30
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

	// Try to acquire advisory lock
	var lockAcquired bool
	err = pool.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", ingestionLockKey).Scan(&lockAcquired)
	if err != nil {
		log.Fatal("Failed to check advisory lock:", err)
	}

	if !lockAcquired {
		log.Println("Another ingestion job is already running. Exiting gracefully.")
		os.Exit(0)
	}

	// Ensure lock is released on exit
	defer func() {
		_, unlockErr := pool.Exec(ctx, "SELECT pg_advisory_unlock($1)", ingestionLockKey)
		if unlockErr != nil {
			log.Printf("Warning: Failed to release advisory lock: %v", unlockErr)
		}
	}()

	log.Println("✅ Acquired advisory lock, starting ingestion...")

	// Get rolling window days from environment variable or use default
	rollingWindowDays := defaultRollingWindowDays
	if daysStr := os.Getenv("INGESTION_WINDOW_DAYS"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 {
			rollingWindowDays = days
		}
	}

	// Calculate rolling window
	now := time.Now()
	postedTo := now.Format("01/02/2006")
	postedFrom := now.AddDate(0, 0, -rollingWindowDays).Format("01/02/2006")

	log.Printf("📅 Pulling opportunities from %s to %s (%d day window)", postedFrom, postedTo, rollingWindowDays)

	// Initialize services
	samService := services.NewSAMService()
	ingestionService := services.NewIngestionService(pool, samService)

	// Run ingestion
	stats, err := ingestionService.IngestOpportunities(ctx, postedFrom, postedTo)
	if err != nil {
		log.Fatalf("❌ Ingestion failed: %v", err)
	}

	// Log results
	log.Println("✅ Ingestion completed successfully")
	log.Printf("📊 Statistics:")
	log.Printf("   Total processed: %d", stats.Total)
	log.Printf("   New: %d", stats.New)
	log.Printf("   Updated: %d", stats.Updated)
	log.Printf("   Skipped: %d", stats.Skipped)
	log.Printf("   Errors: %d", stats.Errors)

	if stats.Errors > 0 {
		log.Printf("⚠️  Warning: %d errors occurred during ingestion", stats.Errors)
		os.Exit(1)
	}

	os.Exit(0)
}

