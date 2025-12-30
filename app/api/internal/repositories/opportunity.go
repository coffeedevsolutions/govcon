package repositories

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"govcon/api/internal/models"
)

type OpportunityRepository struct {
	db *pgxpool.Pool
}

func NewOpportunityRepository(db *pgxpool.Pool) *OpportunityRepository {
	return &OpportunityRepository{db: db}
}

type SearchParams struct {
	PostedFrom string
	PostedTo   string
	Active     *bool
	PType      string
	SearchText string
	Limit      int
	Offset     int
}

type SearchResult struct {
	Items        []models.Opportunity
	TotalRecords int
	Limit        int
	Offset       int
	HasMore      bool
}

// SearchOpportunities searches opportunities with filters, pagination, and full-text search.
func (r *OpportunityRepository) SearchOpportunities(ctx context.Context, params SearchParams) (*SearchResult, error) {
	// Build WHERE clause
	conditions := []string{}
	args := []interface{}{}
	argPos := 1

	if params.PostedFrom != "" {
		// Convert MM/DD/YYYY to YYYY-MM-DD format for database comparison
		postedFromDB, err := convertDateFormat(params.PostedFrom)
		if err != nil {
			// Log error but continue - might be already in correct format
			fmt.Printf("Warning: Failed to convert date '%s': %v\n", params.PostedFrom, err)
		} else {
			// Use string comparison since posted_date is stored as VARCHAR
			// YYYY-MM-DD format allows proper lexicographic comparison
			conditions = append(conditions, fmt.Sprintf("posted_date >= $%d", argPos))
			args = append(args, postedFromDB)
			argPos++
		}
	}

	if params.PostedTo != "" {
		// Convert MM/DD/YYYY to YYYY-MM-DD format for database comparison
		postedToDB, err := convertDateFormat(params.PostedTo)
		if err != nil {
			// Log error but continue - might be already in correct format
			fmt.Printf("Warning: Failed to convert date '%s': %v\n", params.PostedTo, err)
		} else {
			// Use string comparison since posted_date is stored as VARCHAR
			// YYYY-MM-DD format allows proper lexicographic comparison
			conditions = append(conditions, fmt.Sprintf("posted_date <= $%d", argPos))
			args = append(args, postedToDB)
			argPos++
		}
	}

	if params.Active != nil {
		conditions = append(conditions, fmt.Sprintf("active = $%d", argPos))
		args = append(args, *params.Active)
		argPos++
	}

	if params.PType != "" {
		// Map SAM API ptype values to database type values
		// ptype=o means "opportunities" which maps to various types in the database
		// For now, if ptype=o, don't filter by type (show all opportunities)
		// Other ptype values can be mapped here if needed
		if params.PType != "o" {
			conditions = append(conditions, fmt.Sprintf("type = $%d", argPos))
			args = append(args, params.PType)
			argPos++
		}
		// If ptype=o, we don't add a type filter (show all opportunity types)
	}

	if params.SearchText != "" {
		// Use full-text search with GIN index
		conditions = append(conditions, fmt.Sprintf(
			"to_tsvector('english', COALESCE(title, '') || ' ' || COALESCE(department, '') || ' ' || COALESCE(description, '')) @@ plainto_tsquery('english', $%d)",
			argPos,
		))
		args = append(args, params.SearchText)
		argPos++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM opportunity %s", whereClause)
	var totalRecords int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalRecords)
	if err != nil {
		return nil, fmt.Errorf("failed to count opportunities: %w", err)
	}

	// Get paginated results
	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT 
			notice_id, title, organization_type, posted_date, type, base_type,
			archive_type, archive_date, type_of_set_aside, type_of_set_aside_desc,
			response_deadline, naics, classification_code, active,
			point_of_contact, place_of_performance, description, department,
			sub_tier, office, links
		FROM opportunity
		%s
		ORDER BY posted_date DESC, notice_id
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query opportunities: %w", err)
	}
	defer rows.Close()

	var opportunities []models.Opportunity
	for rows.Next() {
		var opp models.Opportunity
		var naicsJSON, contactJSON, placeJSON, linksJSON json.RawMessage
		var activeBool bool

		err := rows.Scan(
			&opp.NoticeID, &opp.Title, &opp.OrganizationType, &opp.PostedDate, &opp.Type, &opp.BaseType,
			&opp.ArchiveType, &opp.ArchiveDate, &opp.TypeOfSetAside, &opp.TypeOfSetAsideDesc,
			&opp.ResponseDeadline, &naicsJSON, &opp.ClassificationCode, &activeBool,
			&contactJSON, &placeJSON, &opp.Description, &opp.Department,
			&opp.SubTier, &opp.Office, &linksJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan opportunity: %w", err)
		}

		opp.Active = models.FlexibleBool(activeBool)

		// Unmarshal JSON fields
		if len(naicsJSON) > 0 {
			json.Unmarshal(naicsJSON, &opp.NAICS)
		}
		if len(contactJSON) > 0 {
			json.Unmarshal(contactJSON, &opp.PointOfContact)
		}
		if len(placeJSON) > 0 {
			json.Unmarshal(placeJSON, &opp.PlaceOfPerformance)
		}
		if len(linksJSON) > 0 {
			json.Unmarshal(linksJSON, &opp.Links)
		}

		opportunities = append(opportunities, opp)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating opportunities: %w", err)
	}

	return &SearchResult{
		Items:        opportunities,
		TotalRecords: totalRecords,
		Limit:        limit,
		Offset:       offset,
		HasMore:      offset+limit < totalRecords,
	}, nil
}

// GetOpportunityByNoticeID retrieves a single opportunity by notice ID.
func (r *OpportunityRepository) GetOpportunityByNoticeID(ctx context.Context, noticeID string) (*models.Opportunity, error) {
	var opp models.Opportunity
	var naicsJSON, contactJSON, placeJSON, linksJSON json.RawMessage
	var activeBool bool

	var solicitationNumber, agencyPathName *string
	err := r.db.QueryRow(ctx, `
		SELECT 
			notice_id, title, organization_type, posted_date, type, base_type,
			archive_type, archive_date, type_of_set_aside, type_of_set_aside_desc,
			response_deadline, naics, classification_code, active,
			point_of_contact, place_of_performance, description, department,
			sub_tier, office, links, solicitation_number, agency_path_name
		FROM opportunity
		WHERE notice_id = $1
	`, noticeID).Scan(
		&opp.NoticeID, &opp.Title, &opp.OrganizationType, &opp.PostedDate, &opp.Type, &opp.BaseType,
		&opp.ArchiveType, &opp.ArchiveDate, &opp.TypeOfSetAside, &opp.TypeOfSetAsideDesc,
		&opp.ResponseDeadline, &naicsJSON, &opp.ClassificationCode, &activeBool,
		&contactJSON, &placeJSON, &opp.Description, &opp.Department,
		&opp.SubTier, &opp.Office, &linksJSON, &solicitationNumber, &agencyPathName,
	)
	if err != nil {
		// Check if error is due to missing columns (migration not run)
		errStr := err.Error()
		if strings.Contains(errStr, "solicitation_number") || 
		   strings.Contains(errStr, "agency_path_name") ||
		   (strings.Contains(errStr, "column") && strings.Contains(errStr, "does not exist")) {
			return nil, fmt.Errorf("database migration required: %w. Run: pnpm --filter api db:migrate", err)
		}
		return nil, fmt.Errorf("failed to get opportunity: %w", err)
	}

	opp.Active = models.FlexibleBool(activeBool)

	// Assign optional fields
	if solicitationNumber != nil {
		opp.SolicitationNumber = *solicitationNumber
	}
	if agencyPathName != nil {
		opp.AgencyPathName = *agencyPathName
	}

	// Unmarshal JSON fields
	if len(naicsJSON) > 0 {
		json.Unmarshal(naicsJSON, &opp.NAICS)
	}
	if len(contactJSON) > 0 {
		json.Unmarshal(contactJSON, &opp.PointOfContact)
	}
	if len(placeJSON) > 0 {
		json.Unmarshal(placeJSON, &opp.PlaceOfPerformance)
	}
	if len(linksJSON) > 0 {
		json.Unmarshal(linksJSON, &opp.Links)
	}

	return &opp, nil
}

// SearchParamsV2 represents search parameters for the new search endpoint
type SearchParamsV2 struct {
	Q          string // keyword search
	NAICS      string // exact match in JSONB array
	SetAside   string // exact match
	State      string // extract from place_of_performance JSONB
	Agency     string // prefix/ILIKE match on agency_path_name
	PostedFrom string // date range
	PostedTo   string
	DueFrom    string
	DueTo      string
	Sort       string // posted_desc, due_asc, relevance
	Limit      int    // default 25, max 100
	Cursor     string // base64 JSON cursor
}

// SearchResultV2 represents the search result with cursor pagination
type SearchResultV2 struct {
	Items      []models.Opportunity
	NextCursor string
	Debug      map[string]interface{} // dev only
}

// Cursor represents the keyset pagination cursor
type Cursor struct {
	PostedDate       string `json:"postedDate,omitempty"`
	ResponseDeadline string `json:"responseDeadline,omitempty"`
	NoticeID         string `json:"noticeId"`
}

// encodeCursor encodes a cursor to base64 JSON string
func encodeCursor(cursor Cursor) (string, error) {
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

// decodeCursor decodes a base64 JSON string to a cursor
func decodeCursor(encoded string) (*Cursor, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var cursor Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}
	return &cursor, nil
}

// SearchOpportunitiesV2 searches opportunities with filters, keyset pagination, and full-text search.
func (r *OpportunityRepository) SearchOpportunitiesV2(ctx context.Context, params SearchParamsV2) (*SearchResultV2, error) {
	// Build WHERE clause dynamically
	conditions := []string{}
	args := []interface{}{}
	argPos := 1

	// Keyword search - use computed tsvector (works with or without migration)
	// If search_tsv column exists (after migration), it will be faster, but this works either way
	if params.Q != "" {
		// Use computed tsvector that includes all searchable fields
		// This works whether or not the migration has been run
		conditions = append(conditions, fmt.Sprintf(
			`to_tsvector('english', 
				COALESCE(title, '') || ' ' || 
				COALESCE(solicitation_number, '') || ' ' || 
				COALESCE(agency_path_name, '') || ' ' || 
				COALESCE(description, '')
			) @@ websearch_to_tsquery('english', $%d)`,
			argPos))
		args = append(args, params.Q)
		argPos++
	}

	// NAICS filter - check if any NAICS object in array has matching code
	if params.NAICS != "" {
		conditions = append(conditions, fmt.Sprintf("naics @> $%d::jsonb", argPos))
		naicsJSON := fmt.Sprintf(`[{"code": "%s"}]`, params.NAICS)
		args = append(args, naicsJSON)
		argPos++
	}

	// Set-aside filter
	if params.SetAside != "" {
		conditions = append(conditions, fmt.Sprintf("type_of_set_aside = $%d", argPos))
		args = append(args, params.SetAside)
		argPos++
	}

	// State filter - extract from place_of_performance JSONB
	if params.State != "" {
		conditions = append(conditions, fmt.Sprintf("place_of_performance->>'state' = $%d", argPos))
		args = append(args, params.State)
		argPos++
	}

	// Agency filter - prefix/ILIKE match on agency_path_name
	if params.Agency != "" {
		conditions = append(conditions, fmt.Sprintf("agency_path_name ILIKE $%d", argPos))
		args = append(args, params.Agency+"%")
		argPos++
	}

	// Posted date range
	if params.PostedFrom != "" {
		postedFromDB, err := convertDateFormat(params.PostedFrom)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("posted_date >= $%d", argPos))
			args = append(args, postedFromDB)
			argPos++
		}
	}

	if params.PostedTo != "" {
		postedToDB, err := convertDateFormat(params.PostedTo)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("posted_date <= $%d", argPos))
			args = append(args, postedToDB)
			argPos++
		}
	}

	// Due date range (response_deadline)
	if params.DueFrom != "" {
		dueFromDB, err := convertDateFormat(params.DueFrom)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("response_deadline >= $%d", argPos))
			args = append(args, dueFromDB)
			argPos++
		}
	}

	if params.DueTo != "" {
		dueToDB, err := convertDateFormat(params.DueTo)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("response_deadline <= $%d", argPos))
			args = append(args, dueToDB)
			argPos++
		}
	}

	// Handle cursor for keyset pagination
	var cursor *Cursor
	if params.Cursor != "" {
		decoded, err := decodeCursor(params.Cursor)
		if err == nil {
			cursor = decoded
		}
	}

	// Add cursor conditions based on sort type
	sortType := params.Sort
	if sortType == "" {
		sortType = "posted_desc"
	}

	if cursor != nil {
		switch sortType {
		case "posted_desc":
			if cursor.PostedDate != "" {
				conditions = append(conditions, fmt.Sprintf(
					"(posted_date < $%d OR (posted_date = $%d AND notice_id < $%d))",
					argPos, argPos, argPos+1,
				))
				args = append(args, cursor.PostedDate, cursor.NoticeID)
				argPos += 2
			}
		case "due_asc":
			if cursor.ResponseDeadline != "" {
				conditions = append(conditions, fmt.Sprintf(
					"(response_deadline > $%d OR (response_deadline = $%d AND notice_id > $%d) OR (response_deadline IS NULL AND notice_id > $%d))",
					argPos, argPos, argPos+1, argPos+1,
				))
				args = append(args, cursor.ResponseDeadline, cursor.NoticeID)
				argPos += 2
			}
		case "relevance":
			// Fall back to posted_desc cursor format
			if cursor.PostedDate != "" {
				conditions = append(conditions, fmt.Sprintf(
					"(posted_date < $%d OR (posted_date = $%d AND notice_id < $%d))",
					argPos, argPos, argPos+1,
				))
				args = append(args, cursor.PostedDate, cursor.NoticeID)
				argPos += 2
			}
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Determine limit
	limit := params.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	// Build ORDER BY clause based on sort type
	var orderBy string
	switch sortType {
	case "due_asc":
		orderBy = "response_deadline ASC NULLS LAST, notice_id ASC"
	case "relevance":
		if params.Q != "" {
			// Use ts_rank for relevance when searching (computed tsvector, works with or without migration)
			orderBy = fmt.Sprintf(
				`ts_rank(to_tsvector('english', 
					COALESCE(title, '') || ' ' || 
					COALESCE(solicitation_number, '') || ' ' || 
					COALESCE(agency_path_name, '') || ' ' || 
					COALESCE(description, '')
				), websearch_to_tsquery('english', $%d)) DESC, posted_date DESC NULLS LAST, notice_id ASC`,
				argPos)
			args = append(args, params.Q)
			argPos++
		} else {
			// Fall back to posted_desc if no search query
			orderBy = "posted_date DESC NULLS LAST, notice_id ASC"
		}
	default: // posted_desc
		orderBy = "posted_date DESC NULLS LAST, notice_id ASC"
	}

	// Build SELECT query with LEFT JOIN to opportunity_description for descriptionStatus
	// Note: If migration hasn't been run, solicitation_number and agency_path_name columns won't exist
	// The query will fail with a clear error that should prompt running the migration
	query := fmt.Sprintf(`
		SELECT 
			o.notice_id, o.title, o.organization_type, o.posted_date, o.type, o.base_type,
			o.archive_type, o.archive_date, o.type_of_set_aside, o.type_of_set_aside_desc,
			o.response_deadline, o.naics, o.classification_code, o.active,
			o.point_of_contact, o.place_of_performance, o.description, o.department,
			o.sub_tier, o.office, o.links, o.solicitation_number, o.agency_path_name,
			CASE
				WHEN od.source_type = 'none' OR od.source_type IS NULL THEN 'none'
				WHEN od.fetch_status = 'fetched' THEN 'ready'
				WHEN od.fetch_status = 'not_found' THEN 'not_found'
				WHEN od.fetch_status = 'error' THEN 'error'
				WHEN od.fetch_status = 'not_requested' THEN 'available_unfetched'
				ELSE 'available_unfetched'
			END AS description_status
		FROM opportunity o
		LEFT JOIN opportunity_description od ON o.notice_id = od.notice_id
		%s
		ORDER BY %s
		LIMIT $%d
	`, whereClause, orderBy, argPos)

	args = append(args, limit+1) // Fetch one extra to determine if there's a next page

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		// Check if error is due to missing columns (migration not run)
		errStr := err.Error()
		if strings.Contains(errStr, "solicitation_number") || 
		   strings.Contains(errStr, "agency_path_name") ||
		   (strings.Contains(errStr, "column") && strings.Contains(errStr, "does not exist")) {
			return nil, fmt.Errorf("database migration required: %w. Run: pnpm --filter api db:migrate", err)
		}
		return nil, fmt.Errorf("failed to query opportunities: %w", err)
	}
	defer rows.Close()

	var opportunities []models.Opportunity
	for rows.Next() {
		var opp models.Opportunity
		var naicsJSON, contactJSON, placeJSON, linksJSON json.RawMessage
		var activeBool bool
		var solicitationNumber, agencyPathName *string
		var descriptionStatus *string

		err := rows.Scan(
			&opp.NoticeID, &opp.Title, &opp.OrganizationType, &opp.PostedDate, &opp.Type, &opp.BaseType,
			&opp.ArchiveType, &opp.ArchiveDate, &opp.TypeOfSetAside, &opp.TypeOfSetAsideDesc,
			&opp.ResponseDeadline, &naicsJSON, &opp.ClassificationCode, &activeBool,
			&contactJSON, &placeJSON, &opp.Description, &opp.Department,
			&opp.SubTier, &opp.Office, &linksJSON, &solicitationNumber, &agencyPathName,
			&descriptionStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan opportunity: %w", err)
		}

		// Assign optional fields
		if solicitationNumber != nil {
			opp.SolicitationNumber = *solicitationNumber
		}
		if agencyPathName != nil {
			opp.AgencyPathName = *agencyPathName
		}
		if descriptionStatus != nil {
			opp.DescriptionStatus = *descriptionStatus
		}
		if err != nil {
			return nil, fmt.Errorf("failed to scan opportunity: %w", err)
		}

		opp.Active = models.FlexibleBool(activeBool)

		// Unmarshal JSON fields
		if len(naicsJSON) > 0 {
			json.Unmarshal(naicsJSON, &opp.NAICS)
		}
		if len(contactJSON) > 0 {
			json.Unmarshal(contactJSON, &opp.PointOfContact)
		}
		if len(placeJSON) > 0 {
			json.Unmarshal(placeJSON, &opp.PlaceOfPerformance)
		}
		if len(linksJSON) > 0 {
			json.Unmarshal(linksJSON, &opp.Links)
		}

		opportunities = append(opportunities, opp)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating opportunities: %w", err)
	}

	// Determine next cursor
	var nextCursor string
	if len(opportunities) > limit {
		// We fetched one extra, remove it
		opportunities = opportunities[:limit]
		lastItem := opportunities[len(opportunities)-1]

		// Create cursor based on sort type
		var cursor Cursor
		cursor.NoticeID = lastItem.NoticeID
		switch sortType {
		case "posted_desc", "relevance":
			cursor.PostedDate = lastItem.PostedDate
		case "due_asc":
			cursor.ResponseDeadline = lastItem.ResponseDeadline
		}

		encoded, err := encodeCursor(cursor)
		if err == nil {
			nextCursor = encoded
		}
	}

	// Build debug info (dev only)
	debug := map[string]interface{}{
		"sort":          sortType,
		"appliedFilters": map[string]interface{}{
			"q":          params.Q,
			"naics":      params.NAICS,
			"setAside":   params.SetAside,
			"state":      params.State,
			"agency":     params.Agency,
			"postedFrom": params.PostedFrom,
			"postedTo":   params.PostedTo,
			"dueFrom":    params.DueFrom,
			"dueTo":      params.DueTo,
		},
	}

	return &SearchResultV2{
		Items:      opportunities,
		NextCursor: nextCursor,
		Debug:      debug,
	}, nil
}

// convertDateFormat converts MM/DD/YYYY to YYYY-MM-DD format
// If the input is already in YYYY-MM-DD format, it returns it as-is
func convertDateFormat(dateStr string) (string, error) {
	// Try parsing as MM/DD/YYYY first
	if t, err := time.Parse("01/02/2006", dateStr); err == nil {
		return t.Format("2006-01-02"), nil
	}
	// Try parsing as YYYY-MM-DD (already in correct format)
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t.Format("2006-01-02"), nil
	}
	// Try parsing as RFC3339 or ISO8601
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t.Format("2006-01-02"), nil
	}
	// Return original if we can't parse (let database handle it)
	return dateStr, fmt.Errorf("unable to parse date: %s", dateStr)
}

