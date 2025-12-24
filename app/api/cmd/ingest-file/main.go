package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"govcon/api/internal/models"
	"govcon/api/internal/services"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run ./cmd/ingest-file <json-file-path>")
	}

	jsonFilePath := os.Args[1]

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
	err = pool.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", 1).Scan(&lockAcquired)
	if err != nil {
		log.Fatal("Failed to check advisory lock:", err)
	}

	if !lockAcquired {
		log.Println("Another ingestion job is already running. Exiting gracefully.")
		os.Exit(0)
	}

	defer func() {
		_, unlockErr := pool.Exec(ctx, "SELECT pg_advisory_unlock($1)", 1)
		if unlockErr != nil {
			log.Printf("Warning: Failed to release advisory lock: %v", unlockErr)
		}
	}()

	log.Println("✅ Acquired advisory lock, starting file ingestion...")

	// Read JSON file
	jsonData, err := os.ReadFile(jsonFilePath)
	if err != nil {
		log.Fatalf("Failed to read JSON file: %v", err)
	}

	// Parse JSON response
	var samResponse struct {
		TotalRecords     int                      `json:"totalRecords"`
		OpportunitiesData []models.Opportunity     `json:"opportunitiesData"`
	}

	if err := json.Unmarshal(jsonData, &samResponse); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	log.Printf("📄 Loaded %d opportunities from file", len(samResponse.OpportunitiesData))

	// Initialize ingestion service
	// We don't need SAM service for file ingestion, but the service requires it
	// Create a dummy one or modify the service to accept nil
	samService := services.NewSAMService()
	ingestionService := services.NewIngestionService(pool, samService)

	// Process each opportunity
	stats := &services.IngestionStats{}
	for _, opp := range samResponse.OpportunitiesData {
		stats.Total++
		result, err := ingestionService.ProcessOpportunity(ctx, opp)
		if err != nil {
			stats.Errors++
			log.Printf("Error processing opportunity %s: %v", opp.NoticeID, err)
			continue
		}
		switch result {
		case "new":
			stats.New++
		case "updated":
			stats.Updated++
		case "skipped":
			stats.Skipped++
		}
	}

	// Log results
	log.Println("✅ File ingestion completed successfully")
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

