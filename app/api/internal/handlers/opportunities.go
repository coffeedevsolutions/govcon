package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"govcon/api/internal/models"
	"govcon/api/internal/repositories"
	"govcon/api/internal/services"
)

type OpportunitiesHandler struct {
	repo            *repositories.OpportunityRepository
	descRepo        *repositories.DescriptionRepository
	descService     *services.DescriptionService
	samService      *services.SAMService
	db              *pgxpool.Pool
}

func NewOpportunitiesHandler(repo *repositories.OpportunityRepository, descRepo *repositories.DescriptionRepository, descService *services.DescriptionService, samService *services.SAMService, db *pgxpool.Pool) *OpportunitiesHandler {
	return &OpportunitiesHandler{
		repo:        repo,
		descRepo:    descRepo,
		descService: descService,
		samService:  samService,
		db:          db,
	}
}

func (h *OpportunitiesHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse query parameters
	postedFrom := r.URL.Query().Get("postedFrom")
	postedTo := r.URL.Query().Get("postedTo")
	searchText := r.URL.Query().Get("search")

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	ptype := r.URL.Query().Get("ptype")
	
	// Parse active filter (optional)
	var active *bool
	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		if parsed, err := strconv.ParseBool(activeStr); err == nil {
			active = &parsed
		}
	}

	// Build search params
	params := repositories.SearchParams{
		PostedFrom: postedFrom,
		PostedTo:   postedTo,
		Active:     active,
		PType:      ptype,
		SearchText: searchText,
		Limit:      limit,
		Offset:     offset,
	}

	// Query repository
	result, err := h.repo.SearchOpportunities(r.Context(), params)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Ensure items is always an array, never null
	items := result.Items
	if items == nil {
		items = []models.Opportunity{}
	}

	// Return response with pagination metadata
	response := map[string]interface{}{
		"items":        items,
		"totalRecords": result.TotalRecords,
		"limit":        result.Limit,
		"offset":       result.Offset,
		"hasMore":      result.HasMore,
	}

	WriteJSON(w, http.StatusOK, response)
}

// HandleSearchV2 handles the new search endpoint with keyset pagination
func (h *OpportunitiesHandler) HandleSearchV2(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse query parameters
	params := repositories.SearchParamsV2{
		Q:          r.URL.Query().Get("q"),
		NAICS:      r.URL.Query().Get("naics"),
		SetAside:   r.URL.Query().Get("setAside"),
		State:      r.URL.Query().Get("state"),
		Agency:     r.URL.Query().Get("agency"),
		PostedFrom: r.URL.Query().Get("postedFrom"),
		PostedTo:     r.URL.Query().Get("postedTo"),
		DueFrom:    r.URL.Query().Get("dueFrom"),
		DueTo:      r.URL.Query().Get("dueTo"),
		Sort:       r.URL.Query().Get("sort"),
		Cursor:     r.URL.Query().Get("cursor"),
	}

	// Parse limit with defaults
	limit := 25
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	params.Limit = limit

	// Query repository
	result, err := h.repo.SearchOpportunitiesV2(r.Context(), params)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := err.Error()
		
		// If it's a migration error, return 503 (Service Unavailable) with helpful message
		if strings.Contains(errorMsg, "database migration required") {
			statusCode = http.StatusServiceUnavailable
		}
		
		WriteJSON(w, statusCode, map[string]string{
			"error": errorMsg,
		})
		return
	}

	// Ensure items is always an array, never null
	items := result.Items
	if items == nil {
		items = []models.Opportunity{}
	}

	// Build response
	response := map[string]interface{}{
		"items":      items,
		"nextCursor": result.NextCursor,
	}

	// Include debug info in dev (check if we're in dev mode - for now always include)
	response["debug"] = result.Debug

	WriteJSON(w, http.StatusOK, response)
}

// HandleGetOpportunity handles GET /opportunities/:noticeId
func (h *OpportunitiesHandler) HandleGetOpportunity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Extract noticeId from path
	// For now, we'll use a simple approach - in production you'd use a router like chi
	path := r.URL.Path
	noticeID := strings.TrimPrefix(path, "/opportunities/")
	if noticeID == "" || noticeID == path {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "noticeId is required"})
		return
	}

	// Query repository
	opportunity, err := h.repo.GetOpportunityByNoticeID(r.Context(), noticeID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{
			"error": "opportunity not found",
		})
		return
	}

	WriteJSON(w, http.StatusOK, opportunity)
}

// HandleGetDescription handles GET /opportunities/:noticeId/description?refresh=false
func (h *OpportunitiesHandler) HandleGetDescription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Extract noticeId from path
	// Path format: /opportunities/{noticeId}/description
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/opportunities/")
	path = strings.TrimSuffix(path, "/description")
	noticeID := strings.Trim(path, "/")
	
	if noticeID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "noticeId is required"})
		return
	}

	ctx := r.Context()
	refresh := r.URL.Query().Get("refresh") == "true"

	// Get opportunity to check description source
	opportunity, err := h.repo.GetOpportunityByNoticeID(ctx, noticeID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{
			"error": "opportunity not found",
		})
		return
	}

	// Detect source type
	sourceType, sourceURL, sourceInline := services.DetectSource(*opportunity)

	// Get existing description if any
	existingDesc, err := h.descRepo.GetDescription(ctx, noticeID)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to get description: %v", err),
		})
		return
	}

	// If we have a cached description and not refreshing, check and self-heal if needed
	if existingDesc != nil && existingDesc.FetchStatus == models.FetchStatusFetched && !refresh {
		currentNormalizationVersion := services.NORMALIZATION_VERSION
		needsReprocessing := false
		var sourceText string
		
		// Check normalization version - if mismatch, re-process from raw JSON or raw text
		if existingDesc.NormalizationVersion == nil || *existingDesc.NormalizationVersion != currentNormalizationVersion {
			needsReprocessing = true
			log.Printf("Description version mismatch: noticeId=%s, stored version=%v, current version=%d, re-processing", 
				noticeID, existingDesc.NormalizationVersion, currentNormalizationVersion)
			
			// Prefer raw_json_response if available, fall back to raw_text
			if existingDesc.RawJsonResponse != nil && *existingDesc.RawJsonResponse != "" {
				// Parse JSON to extract description
				var jsonResponse map[string]interface{}
				if err := json.Unmarshal([]byte(*existingDesc.RawJsonResponse), &jsonResponse); err == nil {
					if descValue, ok := jsonResponse["description"]; ok {
						if desc, ok := descValue.(string); ok && desc != "" {
							sourceText = desc
						}
					}
				}
				// If JSON parsing failed or no description field, use raw JSON as-is
				if sourceText == "" {
					sourceText = *existingDesc.RawJsonResponse
				}
			} else if existingDesc.RawText != nil {
				sourceText = *existingDesc.RawText
			}
		} else if existingDesc.RawText != nil {
			// Self-heal: unwrap JSON wrappers and strip HTML tags in cached descriptions
			rawTextBefore := *existingDesc.RawText
			
			// Unwrap any JSON wrapper
			fixedRaw := services.UnwrapDescriptionText(rawTextBefore)
			
			// Check if text contains HTML tags (need to re-normalize)
			hasHTMLTags := strings.Contains(fixedRaw, "<") && strings.Contains(fixedRaw, ">")
			
			// Also check if normalized fields contain HTML tags (indicates old cached data)
			hasHTMLInNormalized := false
			if existingDesc.RawTextNormalized != nil {
				hasHTMLInNormalized = strings.Contains(*existingDesc.RawTextNormalized, "<") && strings.Contains(*existingDesc.RawTextNormalized, ">")
			}
			if !hasHTMLInNormalized && existingDesc.TextNormalized != nil {
				hasHTMLInNormalized = strings.Contains(*existingDesc.TextNormalized, "<") && strings.Contains(*existingDesc.TextNormalized, ">")
			}
			
			// If unwrapping changed the text OR HTML tags are present, re-process all normalized fields
			if fixedRaw != rawTextBefore || hasHTMLTags || hasHTMLInNormalized {
				needsReprocessing = true
				sourceText = fixedRaw
				// Log when re-processing
				if hasHTMLTags || hasHTMLInNormalized {
					log.Printf("Description self-heal: HTML tags detected for noticeId=%s, re-processing normalized fields", noticeID)
				} else {
					log.Printf("Description self-heal: unwrapping changed text for noticeId=%s, re-processing normalized fields", noticeID)
				}
				log.Printf("  BEFORE: %q", previewText(&rawTextBefore, 120))
				log.Printf("  AFTER unwrap:  %q", previewText(&fixedRaw, 120))
			}
		}
		
		// Re-process if needed
		if needsReprocessing && sourceText != "" {
			// Unwrap description text
			unwrappedText := services.UnwrapDescriptionText(sourceText)
			
			// Re-process normalized fields
			rawTextNormalized := services.NormalizeRaw(unwrappedText)
			textNormalized := services.Normalize(rawTextNormalized)
			contentHash := services.ComputeContentHash(textNormalized)
			
			// Re-process AI-optimized fields
			aiInputText, excerptText, aiMeta, pocEmailPrimary, err := services.OptimizeForAI(rawTextNormalized)
			
			// Update fetchedAt to indicate it was fixed
			now := time.Now()
			existingDesc.FetchedAt = &now
			
			// Update the description with fixed values
			existingDesc.RawText = &unwrappedText
			existingDesc.RawTextNormalized = &rawTextNormalized
			existingDesc.TextNormalized = &textNormalized
			existingDesc.ContentHash = &contentHash
			existingDesc.NormalizationVersion = &currentNormalizationVersion
			
			// Set AI-optimized fields if optimization succeeded
			if err == nil {
				aiInputHash := services.ComputeContentHash(aiInputText)
				aiInputVersion := 1
				existingDesc.AIInputText = &aiInputText
				existingDesc.AIInputHash = &aiInputHash
				existingDesc.AIInputVersion = &aiInputVersion
				existingDesc.AIGeneratedAt = &now
				existingDesc.AIMeta = &aiMeta
				existingDesc.ExcerptText = &excerptText
				existingDesc.POCEmailPrimary = pocEmailPrimary
			} else {
				log.Printf("Description self-heal: failed to optimize for AI for noticeId=%s: %v", noticeID, err)
				// If AI optimization fails, preserve existing AI fields or set defaults
				// Other AI fields can remain as-is (they may be nil, which is fine)
			}
			
			// Safety check: ensure ai_input_version is never nil before persisting (required NOT NULL constraint)
			if existingDesc.AIInputVersion == nil {
				aiInputVersion := 1
				existingDesc.AIInputVersion = &aiInputVersion
				log.Printf("Description self-heal: set default ai_input_version=1 for noticeId=%s", noticeID)
			}
			
			// Persist the fix so it's corrected next time
			if err := h.descRepo.UpsertDescription(ctx, existingDesc); err != nil {
				log.Printf("Description self-heal: failed to persist fix for noticeId=%s: %v", noticeID, err)
				// Continue anyway - we'll return the fixed version even if persistence fails
			} else {
				log.Printf("Description self-heal: successfully persisted fix for noticeId=%s", noticeID)
			}
		}
		
		response := buildDescriptionResponse(existingDesc)
		WriteJSON(w, http.StatusOK, response)
		return
	}

	// Handle different source types
	var desc *models.OpportunityDescription

	switch sourceType {
	case models.SourceTypeNone:
		// No description available
		desc = &models.OpportunityDescription{
			NoticeID:    noticeID,
			SourceType:   models.SourceTypeNone,
			FetchStatus:  models.FetchStatusNotFound,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		h.descRepo.UpsertDescription(ctx, desc)
		response := buildDescriptionResponse(desc)
		WriteJSON(w, http.StatusOK, response)
		return

	case models.SourceTypeInline:
		// Inline text - normalize and store immediately
		rawText := sourceInline
		rawText = services.UnwrapDescriptionText(rawText)
		rawTextNormalized := services.NormalizeRaw(rawText)
		textNormalized := services.Normalize(rawTextNormalized)
		contentHash := services.ComputeContentHash(textNormalized)
		currentNormalizationVersion := services.NORMALIZATION_VERSION

		now := time.Now()
		desc = &models.OpportunityDescription{
			NoticeID:          noticeID,
			SourceType:        models.SourceTypeInline,
			SourceInline:      &sourceInline,
			FetchStatus:       models.FetchStatusFetched,
			FetchedAt:         &now,
			RawText:           &rawText,
			RawTextNormalized: &rawTextNormalized,
			TextNormalized:    &textNormalized,
			ContentHash:       &contentHash,
			NormalizationVersion: &currentNormalizationVersion,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		
		// Generate AI-optimized text (inline text is always fetched)
		aiInputText, excerptText, aiMeta, pocEmailPrimary, err := services.OptimizeForAI(rawTextNormalized)
		if err == nil {
			aiInputHash := services.ComputeContentHash(aiInputText)
			aiInputVersion := 1
			desc.AIInputText = &aiInputText
			desc.AIInputHash = &aiInputHash
			desc.AIInputVersion = &aiInputVersion
			desc.AIGeneratedAt = &now
			desc.AIMeta = &aiMeta
			desc.ExcerptText = &excerptText
			desc.POCEmailPrimary = pocEmailPrimary
		}
		
		h.descRepo.UpsertDescription(ctx, desc)
		response := buildDescriptionResponse(desc)
		WriteJSON(w, http.StatusOK, response)
		return

	case models.SourceTypeURL:
		// URL source - need to fetch
		// Initialize description record if it doesn't exist
		if existingDesc == nil {
			initialDesc := &models.OpportunityDescription{
				NoticeID:    noticeID,
				SourceType:  models.SourceTypeURL,
				SourceURL:   &sourceURL,
				FetchStatus: models.FetchStatusNotRequested,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			h.descRepo.UpsertDescription(ctx, initialDesc)
		}

		// Use advisory lock to prevent concurrent fetches
		lockKey := computeAdvisoryLockKey(noticeID)
		
		var lockAcquired bool
		err := h.db.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&lockAcquired)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("failed to acquire lock: %v", err),
			})
			return
		}

		if !lockAcquired {
			// Another request is fetching, wait a bit and check again
			time.Sleep(500 * time.Millisecond)
			existingDesc, err := h.descRepo.GetDescription(ctx, noticeID)
			if err == nil && existingDesc.FetchStatus == models.FetchStatusFetched {
				response := buildDescriptionResponse(existingDesc)
				WriteJSON(w, http.StatusOK, response)
				return
			}
			WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "description is being fetched by another request",
			})
			return
		}

		// Ensure lock is released
		defer func() {
			h.db.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey)
		}()

		// Check again after acquiring lock (another request might have finished)
		if !refresh {
			existingDesc, err := h.descRepo.GetDescription(ctx, noticeID)
			if err == nil && existingDesc.FetchStatus == models.FetchStatusFetched {
				response := buildDescriptionResponse(existingDesc)
				WriteJSON(w, http.StatusOK, response)
				return
			}
		}

		// Fetch from SAM API
		rawText, rawJsonResponse, httpStatus, contentType, err := h.descService.FetchDescriptionWithKey(sourceURL)

		now := time.Now()
		currentNormalizationVersion := services.NORMALIZATION_VERSION
		desc = &models.OpportunityDescription{
			NoticeID:    noticeID,
			SourceType:   models.SourceTypeURL,
			SourceURL:    &sourceURL,
			HTTPStatus:   &httpStatus,
			FetchedAt:    &now,
			ContentType:  &contentType,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err != nil {
			// Fetch error
			errorMsg := err.Error()
			desc.FetchStatus = models.FetchStatusError
			desc.LastError = &errorMsg
		} else if httpStatus == http.StatusNotFound || strings.Contains(strings.ToLower(rawText), "description not found") {
			// Not found
			desc.FetchStatus = models.FetchStatusNotFound
			desc.RawText = &rawText
			if rawJsonResponse != "" {
				desc.RawJsonResponse = &rawJsonResponse
			}
		} else {
			// Success - store raw JSON response, then unwrap, normalize and store
			if rawJsonResponse != "" {
				desc.RawJsonResponse = &rawJsonResponse
			}
			
			// Unwrap, normalize and store
			rawText = services.UnwrapDescriptionText(rawText)
			rawTextNormalized := services.NormalizeRaw(rawText)
			textNormalized := services.Normalize(rawTextNormalized)
			contentHash := services.ComputeContentHash(textNormalized)

			desc.FetchStatus = models.FetchStatusFetched
			desc.RawText = &rawText
			desc.RawTextNormalized = &rawTextNormalized
			desc.TextNormalized = &textNormalized
			desc.ContentHash = &contentHash
			desc.NormalizationVersion = &currentNormalizationVersion
			
			// Generate AI-optimized text (only for successfully fetched descriptions)
			aiInputText, excerptText, aiMeta, pocEmailPrimary, err := services.OptimizeForAI(rawTextNormalized)
			if err == nil {
				aiInputHash := services.ComputeContentHash(aiInputText)
				aiInputVersion := 1
				desc.AIInputText = &aiInputText
				desc.AIInputHash = &aiInputHash
				desc.AIInputVersion = &aiInputVersion
				desc.AIGeneratedAt = &now
				desc.AIMeta = &aiMeta
				desc.ExcerptText = &excerptText
				desc.POCEmailPrimary = pocEmailPrimary
			}
		}

		// Store in database
		err = h.descRepo.UpsertDescription(ctx, desc)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("failed to store description: %v", err),
			})
			return
		}

		response := buildDescriptionResponse(desc)
		WriteJSON(w, http.StatusOK, response)
		return
	}
}

// buildDescriptionResponse converts OpportunityDescription to DescriptionResponse
func buildDescriptionResponse(desc *models.OpportunityDescription) models.DescriptionResponse {
	response := models.DescriptionResponse{
		NoticeID:   desc.NoticeID,
		SourceType: string(desc.SourceType),
		SourceURL:  desc.SourceURL,
	}

	// Determine status
	switch desc.FetchStatus {
	case models.FetchStatusFetched:
		response.Status = "fetched"
	case models.FetchStatusNotFound:
		response.Status = "not_found"
	case models.FetchStatusError:
		response.Status = "error"
	default:
		if desc.SourceType == models.SourceTypeNone {
			response.Status = "none"
		} else {
			response.Status = "available_unfetched"
		}
	}

	// Set text fields
	response.RawText = desc.RawText
	response.RawPostParseText = desc.RawTextNormalized
	response.NormalizedText = desc.TextNormalized
	response.RawJsonResponse = desc.RawJsonResponse
	response.NormalizationVersion = desc.NormalizationVersion

	// Set fetchedAt
	if desc.FetchedAt != nil {
		response.FetchedAt = new(string)
		*response.FetchedAt = desc.FetchedAt.Format(time.RFC3339)
	}

	// Set lastError if present
	response.LastError = desc.LastError

	return response
}

// computeAdvisoryLockKey computes a lock key from notice_id
func computeAdvisoryLockKey(noticeID string) int64 {
	hash := sha256.Sum256([]byte(noticeID))
	// Use first 8 bytes as int64 (PostgreSQL advisory locks use int8)
	var key int64
	for i := 0; i < 8; i++ {
		key = (key << 8) | int64(hash[i])
	}
	// Ensure positive (PostgreSQL uses signed int8, but we want positive)
	if key < 0 {
		key = -key
	}
	return key
}

// previewText returns a preview of a string for logging purposes
func previewText(s *string, maxLen int) string {
	if s == nil {
		return "<nil>"
	}
	if len(*s) <= maxLen {
		return *s
	}
	return (*s)[:maxLen] + "..."
}


