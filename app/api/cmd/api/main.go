package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"govcon/api/internal/handlers"
	"govcon/api/internal/repositories"
)

func main() {
	// Database connection
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

	// Initialize repository
	opportunityRepo := repositories.NewOpportunityRepository(pool)

	// Initialize handlers
	opportunitiesHandler := handlers.NewOpportunitiesHandler(opportunityRepo)

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		handlers.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	// DB test endpoint
	mux.HandleFunc("/db-test", func(w http.ResponseWriter, r *http.Request) {
		var id int
		var msg string
		err := pool.QueryRow(r.Context(),
			`SELECT id, message FROM ping ORDER BY id DESC LIMIT 1`,
		).Scan(&id, &msg)
		if err != nil {
			handlers.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		handlers.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "message": msg})
	})

	// Opportunities endpoints
	// Note: More specific routes must be registered before less specific ones
	// /opportunities/search must come before /opportunities/ to avoid route conflicts
	mux.HandleFunc("/opportunities/search", opportunitiesHandler.HandleSearchV2)
	mux.HandleFunc("/opportunities", opportunitiesHandler.HandleSearch) // Keep old endpoint for backward compatibility
	// Handle individual opportunity by noticeId (must be last to catch /opportunities/:id)
	mux.HandleFunc("/opportunities/", func(w http.ResponseWriter, r *http.Request) {
		opportunitiesHandler.HandleGetOpportunity(w, r)
	})

	// CORS middleware for development
	handler := corsMiddleware(mux)

	log.Println("Go API listening on :4000")
	log.Fatal(http.ListenAndServe(":4000", handler))
}

// corsMiddleware adds CORS headers for development
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
