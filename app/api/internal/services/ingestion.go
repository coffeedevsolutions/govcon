package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"govcon/api/internal/models"
)

type IngestionStats struct {
	New      int
	Updated  int
	Skipped  int
	Errors   int
	Total    int
}

type IngestionService struct {
	db        *pgxpool.Pool
	samService *SAMService
}

func NewIngestionService(db *pgxpool.Pool, samService *SAMService) *IngestionService {
	return &IngestionService{
		db:        db,
		samService: samService,
	}
}

// IngestOpportunities pulls opportunities from SAM.gov for the given date range,
// handles pagination, and stores them in the database with change detection.
func (s *IngestionService) IngestOpportunities(ctx context.Context, postedFrom, postedTo string) (*IngestionStats, error) {
	stats := &IngestionStats{}
	limit := 100 // SAM API limit per page
	offset := 0

	for {
		// Build request for current page
		req := models.OpportunitiesRequest{
			PostedFrom: postedFrom,
			PostedTo:   postedTo,
			Limit:      limit,
			Offset:     offset,
			PType:      "o", // Default to opportunities
		}

		// Fetch page from SAM API
		response, err := s.samService.SearchOpportunities(req)
		if err != nil {
			return stats, fmt.Errorf("failed to fetch opportunities: %w", err)
		}

		// Process each opportunity
		for _, opp := range response.OpportunitiesData {
			stats.Total++
			result, err := s.ProcessOpportunity(ctx, opp)
			if err != nil {
				stats.Errors++
				// Log error but continue processing
				fmt.Printf("Error processing opportunity %s: %v\n", opp.NoticeID, err)
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

		// Check if we've fetched all pages
		if offset+limit >= response.TotalRecords {
			break
		}

		offset += limit
	}

	return stats, nil
}

// ProcessOpportunity processes a single opportunity: computes hash, checks for changes,
// and updates the database accordingly.
// Returns "new", "updated", or "skipped" to indicate what action was taken.
func (s *IngestionService) ProcessOpportunity(ctx context.Context, opp models.Opportunity) (string, error) {
	// Compute content hash
	hash, err := s.computeContentHash(opp)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	// Serialize raw data for storage
	rawData, err := json.Marshal(opp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw data: %w", err)
	}

	// Check if opportunity exists
	var existingHash string
	var exists bool
	err = s.db.QueryRow(ctx, 
		"SELECT content_hash FROM opportunity WHERE notice_id = $1",
		opp.NoticeID,
	).Scan(&existingHash)

	if err != nil {
		// Opportunity doesn't exist, insert new
		exists = false
	} else {
		exists = true
	}

	now := time.Now()

	if !exists {
		// New opportunity - insert into both tables
		// Insert into opportunity_raw
		_, err = s.db.Exec(ctx, `
			INSERT INTO opportunity_raw (notice_id, raw_data, fetched_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (notice_id) DO UPDATE SET
				raw_data = EXCLUDED.raw_data,
				fetched_at = EXCLUDED.fetched_at
		`, opp.NoticeID, rawData, now)
		if err != nil {
			return "", fmt.Errorf("failed to insert into opportunity_raw: %w", err)
		}

		// Insert into opportunity
		err = s.insertOpportunity(ctx, opp, hash, now, now)
		if err != nil {
			return "", fmt.Errorf("failed to insert opportunity: %w", err)
		}
		return "new", nil
	} else if existingHash != hash {
		// Opportunity exists but hash changed - update
		// Update opportunity_raw first
		_, err = s.db.Exec(ctx, `
			UPDATE opportunity_raw
			SET raw_data = $1, fetched_at = $2
			WHERE notice_id = $3
		`, rawData, now, opp.NoticeID)
		if err != nil {
			return "", fmt.Errorf("failed to update opportunity_raw: %w", err)
		}

		// Insert version log with new hash and new raw snapshot (as per plan)
		_, err = s.db.Exec(ctx, `
			INSERT INTO opportunity_version (notice_id, content_hash, raw_snapshot, fetched_at)
			VALUES ($1, $2, $3, $4)
		`, opp.NoticeID, hash, rawData, now)
		if err != nil {
			return "", fmt.Errorf("failed to insert version: %w", err)
		}

		// Update opportunity
		err = s.updateOpportunity(ctx, opp, hash, now)
		if err != nil {
			return "", fmt.Errorf("failed to update opportunity: %w", err)
		}
		return "updated", nil
	}
	// If hash matches, skip (no changes)
	return "skipped", nil
}

// computeContentHash computes SHA256 hash of all normalized fields (excluding metadata fields).
func (s *IngestionService) computeContentHash(opp models.Opportunity) (string, error) {
	// Create a struct with only the fields we care about for change detection
	hashData := struct {
		NoticeID          string `json:"noticeId"`
		Title             string `json:"title"`
		OrganizationType  string `json:"organizationType"`
		PostedDate        string `json:"postedDate"`
		Type              string `json:"type"`
		BaseType          string `json:"baseType"`
		ArchiveType       string `json:"archiveType"`
		ArchiveDate       string `json:"archiveDate"`
		TypeOfSetAside    string `json:"typeOfSetAside"`
		TypeOfSetAsideDesc string `json:"typeOfSetAsideDesc"`
		ResponseDeadline  string `json:"responseDeadline"`
		NAICS             interface{} `json:"naics"`
		ClassificationCode string `json:"classificationCode"`
		Active            bool   `json:"active"`
		PointOfContact    interface{} `json:"pointOfContact"`
		PlaceOfPerformance interface{} `json:"placeOfPerformance"`
		Description       string `json:"description"`
		Department        string `json:"department"`
		SubTier           string `json:"subTier"`
		Office            string `json:"office"`
		Links             interface{} `json:"links"`
	}{
		NoticeID:          opp.NoticeID,
		Title:             opp.Title,
		OrganizationType:  opp.OrganizationType,
		PostedDate:        opp.PostedDate,
		Type:              opp.Type,
		BaseType:          opp.BaseType,
		ArchiveType:       opp.ArchiveType,
		ArchiveDate:       opp.ArchiveDate,
		TypeOfSetAside:    opp.TypeOfSetAside,
		TypeOfSetAsideDesc: opp.TypeOfSetAsideDesc,
		ResponseDeadline:  opp.ResponseDeadline,
		NAICS:             opp.NAICS,
		ClassificationCode: opp.ClassificationCode,
		Active:            opp.Active.Bool(),
		PointOfContact:    opp.PointOfContact,
		PlaceOfPerformance: opp.PlaceOfPerformance,
		Description:       opp.Description,
		Department:        opp.Department,
		SubTier:           opp.SubTier,
		Office:            opp.Office,
		Links:             opp.Links,
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(hashData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal hash data: %w", err)
	}

	// Compute SHA256 hash
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:]), nil
}

// insertOpportunity inserts a new opportunity into the database.
func (s *IngestionService) insertOpportunity(ctx context.Context, opp models.Opportunity, hash string, firstSeen, lastUpdated time.Time) error {
	naicsJSON, _ := json.Marshal(opp.NAICS)
	contactJSON, _ := json.Marshal(opp.PointOfContact)
	placeJSON, _ := json.Marshal(opp.PlaceOfPerformance)
	linksJSON, _ := json.Marshal(opp.Links)

	_, err := s.db.Exec(ctx, `
		INSERT INTO opportunity (
			notice_id, title, organization_type, posted_date, type, base_type,
			archive_type, archive_date, type_of_set_aside, type_of_set_aside_desc,
			response_deadline, naics, classification_code, active,
			point_of_contact, place_of_performance, description, department,
			sub_tier, office, links, content_hash, first_seen, last_updated
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)
	`,
		opp.NoticeID, opp.Title, opp.OrganizationType, opp.PostedDate, opp.Type, opp.BaseType,
		opp.ArchiveType, opp.ArchiveDate, opp.TypeOfSetAside, opp.TypeOfSetAsideDesc,
		opp.ResponseDeadline, naicsJSON, opp.ClassificationCode, opp.Active.Bool(),
		contactJSON, placeJSON, opp.Description, opp.Department,
		opp.SubTier, opp.Office, linksJSON, hash, firstSeen, lastUpdated,
	)

	return err
}

// updateOpportunity updates an existing opportunity in the database.
func (s *IngestionService) updateOpportunity(ctx context.Context, opp models.Opportunity, hash string, lastUpdated time.Time) error {
	naicsJSON, _ := json.Marshal(opp.NAICS)
	contactJSON, _ := json.Marshal(opp.PointOfContact)
	placeJSON, _ := json.Marshal(opp.PlaceOfPerformance)
	linksJSON, _ := json.Marshal(opp.Links)

	_, err := s.db.Exec(ctx, `
		UPDATE opportunity SET
			title = $2, organization_type = $3, posted_date = $4, type = $5, base_type = $6,
			archive_type = $7, archive_date = $8, type_of_set_aside = $9, type_of_set_aside_desc = $10,
			response_deadline = $11, naics = $12, classification_code = $13, active = $14,
			point_of_contact = $15, place_of_performance = $16, description = $17, department = $18,
			sub_tier = $19, office = $20, links = $21, content_hash = $22, last_updated = $23
		WHERE notice_id = $1
	`,
		opp.NoticeID, opp.Title, opp.OrganizationType, opp.PostedDate, opp.Type, opp.BaseType,
		opp.ArchiveType, opp.ArchiveDate, opp.TypeOfSetAside, opp.TypeOfSetAsideDesc,
		opp.ResponseDeadline, naicsJSON, opp.ClassificationCode, opp.Active.Bool(),
		contactJSON, placeJSON, opp.Description, opp.Department,
		opp.SubTier, opp.Office, linksJSON, hash, lastUpdated,
	)

	return err
}

