package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"govcon/api/internal/models"
)

type DescriptionRepository struct {
	db *pgxpool.Pool
}

func NewDescriptionRepository(db *pgxpool.Pool) *DescriptionRepository {
	return &DescriptionRepository{db: db}
}

// UpsertDescription upserts a description record with conflict handling on notice_id
func (r *DescriptionRepository) UpsertDescription(ctx context.Context, desc *models.OpportunityDescription) error {
	now := time.Now()
	
	// Marshal ai_meta to JSONB (if present)
	var aiMetaJSON []byte
	var err error
	if desc.AIMeta != nil {
		aiMetaJSON, err = json.Marshal(desc.AIMeta)
		if err != nil {
			return fmt.Errorf("failed to marshal ai_meta: %w", err)
		}
	}
	
	// Ensure AIInputVersion is always set to satisfy NOT NULL constraint
	// PostgreSQL's DEFAULT only applies when column is omitted, not when NULL is explicitly provided
	if desc.AIInputVersion == nil {
		defaultVersion := 1
		desc.AIInputVersion = &defaultVersion
	}
	
	query := `
		INSERT INTO opportunity_description (
			notice_id, source_type, source_url, source_inline,
			fetch_status, http_status, fetched_at,
			raw_text, raw_text_normalized, text_normalized,
			content_hash, content_type, last_error,
			ai_input_text, ai_input_hash, ai_input_version, ai_generated_at, ai_meta,
			excerpt_text, poc_email_primary,
			raw_json_response, normalization_version,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)
		ON CONFLICT (notice_id) DO UPDATE SET
			source_type = EXCLUDED.source_type,
			source_url = EXCLUDED.source_url,
			source_inline = EXCLUDED.source_inline,
			fetch_status = EXCLUDED.fetch_status,
			http_status = EXCLUDED.http_status,
			fetched_at = EXCLUDED.fetched_at,
			raw_text = EXCLUDED.raw_text,
			raw_text_normalized = EXCLUDED.raw_text_normalized,
			text_normalized = EXCLUDED.text_normalized,
			content_hash = EXCLUDED.content_hash,
			content_type = EXCLUDED.content_type,
			last_error = EXCLUDED.last_error,
			ai_input_text = EXCLUDED.ai_input_text,
			ai_input_hash = EXCLUDED.ai_input_hash,
			ai_input_version = EXCLUDED.ai_input_version,
			ai_generated_at = EXCLUDED.ai_generated_at,
			ai_meta = EXCLUDED.ai_meta,
			excerpt_text = EXCLUDED.excerpt_text,
			poc_email_primary = EXCLUDED.poc_email_primary,
			raw_json_response = EXCLUDED.raw_json_response,
			normalization_version = EXCLUDED.normalization_version,
			updated_at = EXCLUDED.updated_at
	`
	
	_, err = r.db.Exec(ctx, query,
		desc.NoticeID,
		desc.SourceType,
		desc.SourceURL,
		desc.SourceInline,
		desc.FetchStatus,
		desc.HTTPStatus,
		desc.FetchedAt,
		desc.RawText,
		desc.RawTextNormalized,
		desc.TextNormalized,
		desc.ContentHash,
		desc.ContentType,
		desc.LastError,
		desc.AIInputText,
		desc.AIInputHash,
		desc.AIInputVersion,
		desc.AIGeneratedAt,
		aiMetaJSON, // JSONB field
		desc.ExcerptText,
		desc.POCEmailPrimary,
		desc.RawJsonResponse,
		desc.NormalizationVersion,
		now,
	)
	
	if err != nil {
		return fmt.Errorf("failed to upsert description: %w", err)
	}
	
	return nil
}

// GetDescription retrieves a full description record by notice_id
func (r *DescriptionRepository) GetDescription(ctx context.Context, noticeID string) (*models.OpportunityDescription, error) {
	var desc models.OpportunityDescription
	var sourceType, fetchStatus string
	var createdAt, updatedAt time.Time
	var fetchedAt, summaryUpdatedAt, aiGeneratedAt *time.Time
	var aiMetaJSON []byte
	
	err := r.db.QueryRow(ctx, `
		SELECT 
			notice_id, source_type, source_url, source_inline,
			fetch_status, http_status, fetched_at,
			raw_text, raw_text_normalized, text_normalized,
			content_hash, content_type, last_error,
			brief_summary, brief_summary_model, brief_summary_hash, summary_updated_at,
			ai_input_text, ai_input_hash, ai_input_version, ai_generated_at, ai_meta,
			excerpt_text, poc_email_primary,
			raw_json_response, normalization_version,
			created_at, updated_at
		FROM opportunity_description
		WHERE notice_id = $1
	`, noticeID).Scan(
		&desc.NoticeID,
		&sourceType,
		&desc.SourceURL,
		&desc.SourceInline,
		&fetchStatus,
		&desc.HTTPStatus,
		&fetchedAt,
		&desc.RawText,
		&desc.RawTextNormalized,
		&desc.TextNormalized,
		&desc.ContentHash,
		&desc.ContentType,
		&desc.LastError,
		&desc.BriefSummary,
		&desc.BriefSummaryModel,
		&desc.BriefSummaryHash,
		&summaryUpdatedAt,
		&desc.AIInputText,
		&desc.AIInputHash,
		&desc.AIInputVersion,
		&aiGeneratedAt,
		&aiMetaJSON, // JSONB field
		&desc.ExcerptText,
		&desc.POCEmailPrimary,
		&desc.RawJsonResponse,
		&desc.NormalizationVersion,
		&createdAt,
		&updatedAt,
	)
	
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("description not found")
		}
		return nil, fmt.Errorf("failed to get description: %w", err)
	}
	
	// Convert string types to enum types
	desc.SourceType = models.DescriptionSourceType(sourceType)
	desc.FetchStatus = models.FetchStatus(fetchStatus)
	
	// Set time pointers (these can be nil if NULL in database)
	desc.FetchedAt = fetchedAt
	desc.SummaryUpdatedAt = summaryUpdatedAt
	desc.AIGeneratedAt = aiGeneratedAt
	desc.CreatedAt = createdAt
	desc.UpdatedAt = updatedAt
	
	// Unmarshal ai_meta JSONB field
	if len(aiMetaJSON) > 0 {
		var aiMeta models.AiMeta
		if err := json.Unmarshal(aiMetaJSON, &aiMeta); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ai_meta: %w", err)
		}
		desc.AIMeta = &aiMeta
	}
	
	return &desc, nil
}

// GetDescriptionStatus computes description status from source_type and fetch_status
// This is a helper that can be used for list endpoints
func (r *DescriptionRepository) GetDescriptionStatus(ctx context.Context, noticeID string) (string, error) {
	var sourceType, fetchStatus *string
	
	err := r.db.QueryRow(ctx, `
		SELECT source_type, fetch_status
		FROM opportunity_description
		WHERE notice_id = $1
	`, noticeID).Scan(&sourceType, &fetchStatus)
	
	if err != nil {
		if err.Error() == "no rows in result set" {
			return "none", nil // No record means no description
		}
		return "", fmt.Errorf("failed to get description status: %w", err)
	}
	
	// Compute status using same logic as SQL CASE statement
	if sourceType == nil || *sourceType == "none" {
		return "none", nil
	}
	
	if fetchStatus == nil {
		return "available_unfetched", nil
	}
	
	switch *fetchStatus {
	case "fetched":
		return "ready", nil
	case "not_found":
		return "not_found", nil
	case "error":
		return "error", nil
	case "not_requested":
		return "available_unfetched", nil
	default:
		return "available_unfetched", nil
	}
}

