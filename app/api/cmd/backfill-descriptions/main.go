package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"govcon/api/internal/models"
	"govcon/api/internal/repositories"
	"govcon/api/internal/services"
)

const (
	// Advisory lock key for backfill job
	backfillLockKey = 2
	// Default worker pool size
	defaultWorkers = 3
	// Default rate limit: tokens per second
	defaultRateLimit = 2.0
	// Max retries for failed operations
	maxRetries = 3
	// Initial backoff duration
	initialBackoff = 1 * time.Second
)

type backfillStats struct {
	Total      int
	Processed  int
	Updated    int
	Skipped    int
	Errors     int
	mu         sync.Mutex
}

func (s *backfillStats) IncrementProcessed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Processed++
}

func (s *backfillStats) IncrementUpdated() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Updated++
}

func (s *backfillStats) IncrementSkipped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Skipped++
}

func (s *backfillStats) IncrementErrors() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Errors++
}

// TokenBucket implements a simple token bucket rate limiter
type TokenBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucket(capacity, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) Take() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = min(tb.capacity, tb.tokens+elapsed*tb.refillRate)
	tb.lastRefill = now
	
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}
	return false
}

func (tb *TokenBucket) Wait() {
	for !tb.Take() {
		time.Sleep(100 * time.Millisecond)
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func main() {
	limit := flag.Int("limit", 0, "Maximum number of records to process (0 = no limit)")
	whereClause := flag.String("where", "", "SQL WHERE clause condition (e.g., 'ai_input_text IS NULL AND raw_text_normalized IS NOT NULL')")
	dryRun := flag.Bool("dry-run", false, "Dry run mode: log what would be updated without making changes")
	workers := flag.Int("workers", defaultWorkers, "Number of worker goroutines")
	flag.Parse()

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
	err = pool.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", backfillLockKey).Scan(&lockAcquired)
	if err != nil {
		log.Fatal("Failed to check advisory lock:", err)
	}

	if !lockAcquired {
		log.Println("Another backfill job is already running. Exiting gracefully.")
		os.Exit(0)
	}

	// Ensure lock is released on exit
	defer func() {
		_, unlockErr := pool.Exec(ctx, "SELECT pg_advisory_unlock($1)", backfillLockKey)
		if unlockErr != nil {
			log.Printf("Warning: Failed to release advisory lock: %v", unlockErr)
		}
	}()

	log.Println("✅ Acquired advisory lock, starting backfill...")
	if *dryRun {
		log.Println("🔍 DRY RUN MODE: No changes will be made")
	}

	// Build WHERE clause
	whereSQL := "WHERE raw_text_normalized IS NOT NULL"
	if *whereClause != "" {
		whereSQL += " AND " + *whereClause
	} else {
		// Default: only process records without AI input
		whereSQL += " AND ai_input_text IS NULL"
	}

	// Count total records
	var totalCount int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM opportunity_description %s", whereSQL)
	err = pool.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		log.Fatalf("Failed to count records: %v", err)
	}

	if totalCount == 0 {
		log.Println("No records found matching criteria")
		os.Exit(0)
	}

	log.Printf("📊 Found %d records to process", totalCount)
	if *limit > 0 && *limit < totalCount {
		log.Printf("⚠️  Limiting to %d records", *limit)
		totalCount = *limit
	}

	// Initialize repositories and services
	descRepo := repositories.NewDescriptionRepository(pool)
	descService := services.NewDescriptionService()

	// Create rate limiter (for SAM API calls if needed)
	rateLimit := defaultRateLimit
	if rateStr := os.Getenv("BACKFILL_RATE_LIMIT"); rateStr != "" {
		if r, err := strconv.ParseFloat(rateStr, 64); err == nil && r > 0 {
			rateLimit = r
		}
	}
	tokenBucket := NewTokenBucket(rateLimit, rateLimit)

	// Adjust workers if needed
	if *workers < 1 {
		*workers = 1
	}
	if *workers > 10 {
		log.Printf("⚠️  Limiting workers to 10 (requested: %d)", *workers)
		*workers = 10
	}

	stats := &backfillStats{Total: totalCount}

	// Query records
	query := fmt.Sprintf(`
		SELECT notice_id, raw_text_normalized, fetch_status, source_type
		FROM opportunity_description
		%s
		ORDER BY notice_id
	`, whereSQL)
	if *limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", *limit)
	}

	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Fatalf("Failed to query records: %v", err)
	}
	defer rows.Close()

	// Create channels for work distribution
	workChan := make(chan record, *workers*2)
	doneChan := make(chan bool, *workers)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for rec := range workChan {
				processRecord(ctx, rec, descRepo, descService, tokenBucket, stats, *dryRun, workerID)
			}
			doneChan <- true
		}(i)
	}

	// Read records and send to workers
	go func() {
		defer close(workChan)
		for rows.Next() {
			var rec record
			err := rows.Scan(&rec.NoticeID, &rec.RawTextNormalized, &rec.FetchStatus, &rec.SourceType)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				stats.IncrementErrors()
				continue
			}
			workChan <- rec
		}
		if err := rows.Err(); err != nil {
			log.Printf("Error iterating rows: %v", err)
		}
	}()

	// Wait for all workers to finish
	wg.Wait()

	// Log results
	log.Println("✅ Backfill completed")
	log.Printf("📊 Statistics:")
	log.Printf("   Total: %d", stats.Total)
	log.Printf("   Processed: %d", stats.Processed)
	log.Printf("   Updated: %d", stats.Updated)
	log.Printf("   Skipped: %d", stats.Skipped)
	log.Printf("   Errors: %d", stats.Errors)

	if stats.Errors > 0 {
		log.Printf("⚠️  Warning: %d errors occurred during backfill", stats.Errors)
		os.Exit(1)
	}

	os.Exit(0)
}

type record struct {
	NoticeID          string
	RawTextNormalized *string
	FetchStatus       string
	SourceType        string
}

func processRecord(ctx context.Context, rec record, descRepo *repositories.DescriptionRepository, descService *services.DescriptionService, tokenBucket *TokenBucket, stats *backfillStats, dryRun bool, workerID int) {
	stats.IncrementProcessed()

	// Check if we should process this record
	if rec.RawTextNormalized == nil || *rec.RawTextNormalized == "" {
		stats.IncrementSkipped()
		return
	}

	// Only process if fetch_status is 'fetched' or source_type is 'inline'
	if rec.FetchStatus != "fetched" && rec.SourceType != "inline" {
		stats.IncrementSkipped()
		return
	}

	// Rate limit (for potential SAM API calls)
	tokenBucket.Wait()

	// Process with retry logic
	var err error
	backoff := initialBackoff
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[Worker %d] Retry %d/%d for notice_id %s after %v", workerID, attempt, maxRetries, rec.NoticeID, backoff)
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
		}

		err = processRecordWithRetry(ctx, rec, descRepo, dryRun)
		if err == nil {
			break
		}

		// Check if error is retryable (429, 5xx, etc.)
		if !isRetryableError(err) {
			break
		}
	}

	if err != nil {
		log.Printf("[Worker %d] Failed to process notice_id %s after retries: %v", workerID, rec.NoticeID, err)
		stats.IncrementErrors()
		return
	}

	stats.IncrementUpdated()
	if (stats.Updated % 100) == 0 {
		log.Printf("✅ Processed %d records...", stats.Updated)
	}
}

func processRecordWithRetry(ctx context.Context, rec record, descRepo *repositories.DescriptionRepository, dryRun bool) error {
	// Get full description record
	desc, err := descRepo.GetDescription(ctx, rec.NoticeID)
	if err != nil {
		return fmt.Errorf("failed to get description: %w", err)
	}

	// Generate AI-optimized text
	rawTextNormalized := *rec.RawTextNormalized
	aiInputText, excerptText, aiMeta, pocEmailPrimary, err := services.OptimizeForAI(rawTextNormalized)
	if err != nil {
		return fmt.Errorf("failed to optimize for AI: %w", err)
	}

	if dryRun {
		log.Printf("[DRY RUN] Would update notice_id %s: ai_input_text=%d chars, excerpt_text=%d chars", rec.NoticeID, len(aiInputText), len(excerptText))
		return nil
	}

	// Update description with AI fields
	aiInputHash := services.ComputeContentHash(aiInputText)
	aiInputVersion := 1
	now := time.Now()
	desc.AIInputText = &aiInputText
	desc.AIInputHash = &aiInputHash
	desc.AIInputVersion = &aiInputVersion
	desc.AIGeneratedAt = &now
	desc.AIMeta = &aiMeta
	desc.ExcerptText = &excerptText
	desc.POCEmailPrimary = pocEmailPrimary

	err = descRepo.UpsertDescription(ctx, desc)
	if err != nil {
		return fmt.Errorf("failed to upsert description: %w", err)
	}

	return nil
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for HTTP status codes in error message
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "500") || strings.Contains(errStr, "502") || strings.Contains(errStr, "503") || strings.Contains(errStr, "504") {
		return true
	}
	// Check for network/timeout errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection") || strings.Contains(errStr, "network") {
		return true
	}
	return false
}

