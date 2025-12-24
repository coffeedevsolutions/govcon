package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"govcon/api/internal/models"
	"govcon/api/internal/repositories"
)

type OpportunitiesHandler struct {
	repo *repositories.OpportunityRepository
}

func NewOpportunitiesHandler(repo *repositories.OpportunityRepository) *OpportunitiesHandler {
	return &OpportunitiesHandler{
		repo: repo,
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


