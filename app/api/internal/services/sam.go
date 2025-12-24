package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"govcon/api/internal/models"
)

type SAMService struct {
	APIKey string
	BaseURL string
}

func NewSAMService() *SAMService {
	apiKey := os.Getenv("SAM_API_KEY")
	if apiKey == "" {
		apiKey = "SAM-b75dbdc2-c79c-48b1-aaa4-2fc39b0977f4" // fallback to provided key
	}

	return &SAMService{
		APIKey:  apiKey,
		BaseURL: "https://api.sam.gov/opportunities/v2/search",
	}
}

func (s *SAMService) SearchOpportunities(req models.OpportunitiesRequest) (*models.OpportunitiesResponse, error) {
	// Build query parameters
	params := url.Values{}
	params.Add("api_key", s.APIKey)
	params.Add("postedFrom", req.PostedFrom)
	params.Add("postedTo", req.PostedTo)
	params.Add("limit", strconv.Itoa(req.Limit))
	params.Add("offset", strconv.Itoa(req.Offset))
	params.Add("ptype", req.PType)

	// Build request URL
	requestURL := fmt.Sprintf("%s?%s", s.BaseURL, params.Encode())

	// Create HTTP request
	httpReq, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SAM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body first for better error messages
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var samResponse struct {
		TotalRecords     int                      `json:"totalRecords"`
		OpportunitiesData []models.Opportunity     `json:"opportunitiesData"`
	}

	if err := json.Unmarshal(bodyBytes, &samResponse); err != nil {
		// Return more detailed error with a snippet of the response
		bodyPreview := string(bodyBytes)
		if len(bodyPreview) > 500 {
			bodyPreview = bodyPreview[:500] + "..."
		}
		return nil, fmt.Errorf("failed to decode response: %w\nResponse preview: %s", err, bodyPreview)
	}

	return &models.OpportunitiesResponse{
		TotalRecords:     samResponse.TotalRecords,
		OpportunitiesData: samResponse.OpportunitiesData,
	}, nil
}

