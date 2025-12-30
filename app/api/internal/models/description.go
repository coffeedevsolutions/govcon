package models

import "time"

// DescriptionSourceType represents the source type of a description
type DescriptionSourceType string

const (
	SourceTypeNone   DescriptionSourceType = "none"
	SourceTypeInline DescriptionSourceType = "inline"
	SourceTypeURL    DescriptionSourceType = "url"
)

// FetchStatus represents the fetch status of a description
type FetchStatus string

const (
	FetchStatusNotRequested FetchStatus = "not_requested"
	FetchStatusFetched      FetchStatus = "fetched"
	FetchStatusNotFound     FetchStatus = "not_found"
	FetchStatusError         FetchStatus = "error"
)

// AiMeta represents structured metadata extracted from opportunity descriptions
type AiMeta struct {
	POCEmails          []string `json:"poc_emails"`
	POCPhones          []string `json:"poc_phones"`
	ImportantURLs      []string `json:"important_urls"`
	SetAsideDetected   *string  `json:"set_aside_detected,omitempty"`
	ClausesKept        []string `json:"clauses_kept"`
	CertsRequired      []string `json:"certs_required"`
	WAWFRequired       *bool    `json:"wawf_required,omitempty"`
	QuoteValidityDays  *int     `json:"quote_validity_days,omitempty"`
	DORated            *bool    `json:"do_rated,omitempty"`
	RequiresIRPODReview *bool   `json:"requires_irpod_review,omitempty"`
	KeyRequirements    []string `json:"key_requirements"`
}

// OpportunityDescription represents a description record in the database
type OpportunityDescription struct {
	NoticeID           string              `json:"noticeId"`
	SourceType         DescriptionSourceType `json:"sourceType"`
	SourceURL          *string             `json:"sourceUrl,omitempty"`
	SourceInline       *string             `json:"sourceInline,omitempty"`
	FetchStatus        FetchStatus         `json:"fetchStatus"`
	HTTPStatus         *int                `json:"httpStatus,omitempty"`
	FetchedAt          *time.Time          `json:"fetchedAt,omitempty"`
	RawText            *string             `json:"rawText,omitempty"`
	RawTextNormalized  *string             `json:"rawTextNormalized,omitempty"`
	TextNormalized     *string             `json:"textNormalized,omitempty"`
	ContentHash        *string             `json:"contentHash,omitempty"`
	ContentType        *string             `json:"contentType,omitempty"`
	LastError          *string             `json:"lastError,omitempty"`
	BriefSummary       *string             `json:"briefSummary,omitempty"`
	BriefSummaryModel  *string             `json:"briefSummaryModel,omitempty"`
	BriefSummaryHash   *string             `json:"briefSummaryHash,omitempty"`
	SummaryUpdatedAt   *time.Time          `json:"summaryUpdatedAt,omitempty"`
	AIInputText        *string             `json:"aiInputText,omitempty"`
	AIInputHash        *string             `json:"aiInputHash,omitempty"`
	AIInputVersion     *int                `json:"aiInputVersion,omitempty"`
	AIGeneratedAt      *time.Time         `json:"aiGeneratedAt,omitempty"`
	AIMeta             *AiMeta             `json:"aiMeta,omitempty"`
	ExcerptText        *string             `json:"excerptText,omitempty"`
	POCEmailPrimary    *string             `json:"pocEmailPrimary,omitempty"`
	RawJsonResponse    *string             `json:"rawJsonResponse,omitempty"`
	NormalizationVersion *int              `json:"normalizationVersion,omitempty"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
}

// DescriptionResponse represents the API response for a description
type DescriptionResponse struct {
	NoticeID          string    `json:"noticeId"`
	Status            string    `json:"status"` // fetched|not_found|none|error
	SourceType        string    `json:"sourceType"` // url|inline|none
	SourceURL         *string   `json:"sourceUrl,omitempty"`
	RawText           *string   `json:"rawText,omitempty"`
	RawPostParseText  *string   `json:"rawPostParseText,omitempty"` // raw_text_normalized
	NormalizedText    *string   `json:"normalizedText,omitempty"`  // text_normalized
	RawJsonResponse    *string   `json:"rawJsonResponse,omitempty"` // raw_json_response
	NormalizationVersion *int    `json:"normalizationVersion,omitempty"` // normalization_version
	FetchedAt         *string   `json:"fetchedAt,omitempty"`
	LastError         *string   `json:"lastError,omitempty"` // Error message if status is "error"
}

