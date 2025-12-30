package models

import (
	"encoding/json"
	"strings"
)

// FlexibleBool handles both string and bool JSON values
type FlexibleBool bool

func (fb *FlexibleBool) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		// Try to parse as string
		s = strings.ToLower(strings.TrimSpace(s))
		*fb = FlexibleBool(s == "true" || s == "1" || s == "yes")
		return nil
	}
	
	// Try to parse as bool
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*fb = FlexibleBool(b)
		return nil
	}
	
	// Try to parse as number
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*fb = FlexibleBool(n != 0)
		return nil
	}
	
	// Default to false if we can't parse
	*fb = false
	return nil
}

func (fb FlexibleBool) MarshalJSON() ([]byte, error) {
	return json.Marshal(bool(fb))
}

func (fb FlexibleBool) Bool() bool {
	return bool(fb)
}

// FlexibleString handles both string and object JSON values
// If the value is an object, it tries to extract common fields like "value", "code", "name", etc.
type FlexibleString string

func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	// First try to unmarshal as a string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fs = FlexibleString(s)
		return nil
	}

	// If not a string, try to unmarshal as an object and extract a value
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		// Try common field names that might contain the actual value
		commonFields := []string{"value", "code", "name", "description", "text", "label"}
		for _, field := range commonFields {
			if val, ok := obj[field].(string); ok && val != "" {
				*fs = FlexibleString(val)
				return nil
			}
		}
		// If no common field found, try to marshal the object back to JSON as a string representation
		if objBytes, err := json.Marshal(obj); err == nil {
			*fs = FlexibleString(string(objBytes))
			return nil
		}
	}

	// Default to empty string
	*fs = ""
	return nil
}

func (fs FlexibleString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(fs))
}

func (fs FlexibleString) String() string {
	return string(fs)
}

// Opportunity represents a SAM.gov opportunity
type Opportunity struct {
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
	NAICS             []struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	} `json:"naics"`
	ClassificationCode string `json:"classificationCode"`
	Active             FlexibleBool `json:"active"`
	PointOfContact     []struct {
		Fax           string `json:"fax"`
		Type          string `json:"type"`
		Email         string `json:"email"`
		Phone         string `json:"phone"`
		Title         string `json:"title"`
		FullName      string `json:"fullName"`
		AdditionalInfoLink string `json:"additionalInfoLink"`
	} `json:"pointOfContact"`
	PlaceOfPerformance struct {
		StreetAddress FlexibleString `json:"streetAddress"`
		City          FlexibleString `json:"city"`
		State         FlexibleString `json:"state"`
		Zip           FlexibleString `json:"zip"`
		Country       FlexibleString `json:"country"`
	} `json:"placeOfPerformance"`
	Description        string `json:"description"`
	Department         string `json:"department"`
	SubTier            string `json:"subTier"`
	Office            string `json:"office"`
	SolicitationNumber string `json:"solicitationNumber,omitempty"`
	AgencyPathName     string `json:"agencyPathName,omitempty"`
	Links              []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
		Type string `json:"type"`
	} `json:"links"`
	DescriptionStatus string `json:"descriptionStatus,omitempty"` // none | ready | not_found | error | available_unfetched
}

// OpportunitiesResponse represents the SAM.gov API response
type OpportunitiesResponse struct {
	TotalRecords int           `json:"totalRecords"`
	OpportunitiesData []Opportunity `json:"opportunitiesData"`
}

// OpportunitiesRequest represents the request parameters for SAM.gov API
type OpportunitiesRequest struct {
	PostedFrom string `json:"postedFrom"`
	PostedTo   string `json:"postedTo"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	PType      string `json:"ptype"`
}

