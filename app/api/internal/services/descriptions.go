package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"govcon/api/internal/models"
)

// Compiled regex patterns (reused across calls)
var (
	spacePattern = regexp.MustCompile(`\s{2,}`)
	htmlTagPattern = regexp.MustCompile(`<[^>]*>`)
	// Pattern to match punctuation followed by HTML entities like .&nbsp;, ,&nbsp;, ;&nbsp;, etc.
	punctuationEntityPattern = regexp.MustCompile(`([.,;:!?])(&nbsp;|&ensp;|&emsp;|&thinsp;)`)
	// Pattern to match HTML formatting tags to preserve (case-insensitive)
	formattingTagPattern = regexp.MustCompile(`(?i)</?(strong|b|em|i|u|br|p)(\s[^>]*)?/?>`)
)

// DescriptionService provides description-related operations
type DescriptionService struct {
	samAPIKey string
}

// NewDescriptionService creates a new DescriptionService
// Uses the same fallback API key as SAMService for consistency
func NewDescriptionService() *DescriptionService {
	apiKey := os.Getenv("SAM_API_KEY")
	if apiKey == "" {
		apiKey = "SAM-b75dbdc2-c79c-48b1-aaa4-2fc39b0977f4" // fallback to provided key (same as SAMService)
	}
	return &DescriptionService{
		samAPIKey: apiKey,
	}
}

// FetchDescriptionWithKey fetches a description using the service's API key
// Returns: rawText, rawJsonResponse, httpStatus, contentType, error
func (s *DescriptionService) FetchDescriptionWithKey(descURL string) (string, string, int, string, error) {
	if s.samAPIKey == "" {
		return "", "", 0, "", fmt.Errorf("SAM_API_KEY environment variable is required for URL fetching")
	}
	return FetchDescription(descURL, s.samAPIKey)
}

const (
	maxBodySize = 5 * 1024 * 1024 // 5MB
	fetchTimeout = 10 * time.Second
	maxExtractScanLength = 10 * 1024 * 1024 // 10MB max scan length
	maxExtractedLength = 5 * 1024 * 1024    // 5MB max extracted description length
	maxUnwrapRecursion = 2                   // Max recursion depth for UnwrapDescriptionText
	NORMALIZATION_VERSION = 4                // Version of normalization logic - increment when NormalizeRaw, Normalize, or UnwrapDescriptionText changes
)

// DetectSource analyzes the description field and determines the source type
// Returns: sourceType, url (if url), inline (if inline)
func DetectSource(opportunity models.Opportunity) (sourceType models.DescriptionSourceType, urlStr string, inline string) {
	desc := strings.TrimSpace(opportunity.Description)
	
	// If empty or null, return none
	if desc == "" {
		return models.SourceTypeNone, "", ""
	}
	
	// If starts with http:// or https://, treat as URL
	if strings.HasPrefix(desc, "http://") || strings.HasPrefix(desc, "https://") {
		return models.SourceTypeURL, desc, ""
	}
	
	// Otherwise, treat as inline text
	return models.SourceTypeInline, "", desc
}

// parseLenientJSONString parses a JSON string starting at the opening quote index.
// It is lenient: it allows raw \n/\r inside the string (not valid JSON, but seen in practice).
// Handles escape sequences and surrogate pairs.
// Returns (value, endIndexAfterClosingQuote, ok).
func parseLenientJSONString(s string, startQuote int) (string, int, bool) {
	if startQuote < 0 || startQuote >= len(s) || s[startQuote] != '"' {
		return "", 0, false
	}

	var b strings.Builder
	i := startQuote + 1

	for i < len(s) {
		ch := s[i]

		// Closing quote (not escaped)
		if ch == '"' {
			return b.String(), i + 1, true
		}

		// Escape sequence
		if ch == '\\' {
			i++
			if i >= len(s) {
				return "", 0, false
			}
			esc := s[i]
			switch esc {
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			case '/':
				b.WriteByte('/')
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'u':
				// \uXXXX or surrogate pair
				if i+4 >= len(s) {
					return "", 0, false
				}
				hexStr := s[i+1 : i+5]
				u, err := strconv.ParseUint(hexStr, 16, 16)
				if err != nil {
					return "", 0, false
				}
				codePoint := rune(u)

				// Check for surrogate pair (high surrogate: D800-DBFF, low surrogate: DC00-DFFF)
				if codePoint >= 0xD800 && codePoint <= 0xDBFF {
					// High surrogate - check if next is low surrogate
					if i+10 < len(s) && s[i+5] == '\\' && s[i+6] == 'u' {
						hexStr2 := s[i+7 : i+11]
						u2, err2 := strconv.ParseUint(hexStr2, 16, 16)
						if err2 == nil {
							codePoint2 := rune(u2)
							if codePoint2 >= 0xDC00 && codePoint2 <= 0xDFFF {
								// Surrogate pair: combine into single code point
								combined := 0x10000 + (codePoint-0xD800)*0x400 + (codePoint2 - 0xDC00)
								b.WriteRune(rune(combined))
								i += 11 // Skip from 'u' (i) to after second hex (i+10 is last hex char, i+11 is after)
								continue // Skip the i++ at end of switch
							}
						}
					}
				}

				b.WriteRune(codePoint)
				i += 4
			default:
				// Unknown escape; keep it (lenient)
				b.WriteByte(esc)
			}
			i++
			continue
		}

		// Leniently allow raw newlines / carriage returns inside the string
		b.WriteByte(ch)
		i++
	}

	return "", 0, false
}

// ExtractDescriptionJSONLike attempts to extract the value of the top-level "description"
// key from a JSON-ish payload, even if the overall JSON is malformed (e.g., raw newlines
// inside strings). Returns (desc, true) on success.
// Only matches the top-level "description" key to avoid nested or string-literal matches.
func ExtractDescriptionJSONLike(s string) (string, bool) {
	// Guardrails: limit scan length
	if len(s) > maxExtractScanLength {
		return "", false
	}

	key := `"description"`
	keyLen := len(key)
	depth := 0
	inString := false
	escapeNext := false
	i := 0

	// Find opening brace
	for i < len(s) && s[i] != '{' {
		i++
	}
	if i >= len(s) {
		return "", false
	}
	depth = 1
	i++ // Move past '{'

	// Scan character by character
	for i < len(s) {
		ch := s[i]

		if escapeNext {
			escapeNext = false
			i++
			continue
		}

		if ch == '\\' && inString {
			escapeNext = true
			i++
			continue
		}

		if ch == '"' {
			// Check if we're at the top level (depth == 1) and not in a string (just entering a key)
			if depth == 1 && !inString {
				// Potential key match - check if it's "description"
				if i+keyLen <= len(s) && s[i:i+keyLen] == key {
					// Found the key, move past it
					i += keyLen

					// Skip whitespace until colon
					for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
						i++
					}
					if i >= len(s) || s[i] != ':' {
						return "", false
					}
					i++ // past ':'

					// Skip whitespace to value
					for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
						i++
					}
					if i >= len(s) {
						return "", false
					}

					// We only handle string values here: "...."
					if s[i] != '"' {
						return "", false
					}

					// Parse the string value (lenient)
					val, _, ok := parseLenientJSONString(s, i)
					if !ok {
						return "", false
					}

					// Guardrail: limit extracted length
					if len(val) > maxExtractedLength {
						return "", false
					}

					return val, true
				}
			}
			inString = !inString
			i++
			continue
		}

		if !inString {
			if ch == '{' || ch == '[' {
				depth++
				i++
				continue
			}
			if ch == '}' || ch == ']' {
				depth--
				if depth < 0 {
					return "", false
				}
				i++
				continue
			}
		}

		i++
	}

	return "", false
}

// UnwrapDescriptionText tries to extract the real description text from common SAM formats.
// Handles: plain text, {"description":"..."}, and double-encoded JSON strings.
// Uses recursion limit to avoid pathological inputs.
func UnwrapDescriptionText(input string) string {
	return unwrapDescriptionTextRecursive(input, 0)
}

// unwrapDescriptionTextRecursive is the recursive implementation with depth tracking.
func unwrapDescriptionTextRecursive(input string, depth int) string {
	if depth >= maxUnwrapRecursion {
		return input
	}

	s := strings.TrimSpace(input)
	if s == "" {
		return input
	}

	// Case A: input is a JSON object with "description"
	if strings.HasPrefix(s, "{") && strings.Contains(s, "\"description\"") {
		var obj struct {
			Description any `json:"description"`
		}
		if err := json.Unmarshal([]byte(s), &obj); err == nil {
			switch v := obj.Description.(type) {
			case string:
				if strings.TrimSpace(v) != "" {
					// Recurse: some SAM payloads contain another JSON wrapper in the value.
					return unwrapDescriptionTextRecursive(v, depth+1)
				}
			case map[string]any:
				// Handle map by marshaling and recursing
				if marshaled, err := json.Marshal(v); err == nil {
					return unwrapDescriptionTextRecursive(string(marshaled), depth+1)
				}
			case []any:
				// Handle slice by marshaling and recursing
				if marshaled, err := json.Marshal(v); err == nil {
					return unwrapDescriptionTextRecursive(string(marshaled), depth+1)
				}
			}
		} else {
			// Fallback for malformed JSON
			if v, ok := ExtractDescriptionJSONLike(s); ok && strings.TrimSpace(v) != "" {
				// Recurse once in case it was double-wrapped
				return unwrapDescriptionTextRecursive(v, depth+1)
			}
		}
	}

	// Case B: input is a JSON-encoded string (double encoded)
	if strings.HasPrefix(s, "\"") {
		var inner string
		if err := json.Unmarshal([]byte(s), &inner); err == nil {
			// recurse: inner could be {"description":"..."} or plain text
			return unwrapDescriptionTextRecursive(inner, depth+1)
		} else {
			// Inner unmarshal failed - try lenient extraction as fallback
			if v, ok := ExtractDescriptionJSONLike(s); ok && strings.TrimSpace(v) != "" {
				return unwrapDescriptionTextRecursive(v, depth+1)
			}
		}
	}

	return input
}

// FetchDescription fetches a description from a SAM API URL
// Returns: rawText, rawJsonResponse, httpStatus, contentType, error
func FetchDescription(descURL string, apiKey string) (string, string, int, string, error) {
	// Helper to ensure all returned text is unwrapped and trimmed
	finalize := func(s string) string {
		return strings.TrimSpace(UnwrapDescriptionText(s))
	}

	// Parse URL and append API key safely
	u, err := url.Parse(descURL)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("invalid URL: %w", err)
	}
	
	q := u.Query()
	q.Set("api_key", apiKey)
	u.RawQuery = q.Encode()
	finalURL := u.String()
	
	// Create HTTP request
	httpReq, err := http.NewRequest("GET", finalURL, nil)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Accept", "application/json")
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: fetchTimeout,
	}
	
	// Execute request
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	// Get content type
	contentType := resp.Header.Get("Content-Type")
	
	// Limit body size using LimitReader
	limitedReader := io.LimitReader(resp.Body, maxBodySize)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", "", resp.StatusCode, contentType, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Check if we hit the limit
	if len(bodyBytes) >= maxBodySize {
		return "", "", resp.StatusCode, contentType, fmt.Errorf("response body exceeds maximum size of %d bytes", maxBodySize)
	}
	
	// Store raw JSON response before any processing
	rawJsonResponse := string(bodyBytes)
	
	// Try to parse as JSON and extract description field
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &jsonResponse); err == nil {
		// Successfully parsed as JSON, try to extract description field
		if descValue, ok := jsonResponse["description"]; ok {
			// Handle string description
			if desc, ok := descValue.(string); ok && desc != "" {
				// Unwrap any JSON wrapper before returning
				return finalize(desc), rawJsonResponse, resp.StatusCode, contentType, nil
			}
		}
		// If description field doesn't exist or is empty, check for error messages
		if errorMsg, ok := jsonResponse["error"].(string); ok {
			if strings.Contains(strings.ToLower(errorMsg), "description not found") {
				return "", rawJsonResponse, http.StatusNotFound, contentType, nil
			}
		}
		// If we have JSON but no description field, return the raw JSON as fallback
		rawText := string(bodyBytes)
		rawText = finalize(rawText)
		if resp.StatusCode != http.StatusOK {
			return rawText, rawJsonResponse, resp.StatusCode, contentType, fmt.Errorf("SAM API returned status %d", resp.StatusCode)
		}
		return rawText, rawJsonResponse, resp.StatusCode, contentType, nil
	} else {
		// JSON unmarshal failed - log error if debug is enabled
		if os.Getenv("DEBUG_JSON_UNMARSHAL") == "true" {
			// Log error and first 500 chars for debugging
			previewLen := 500
			if len(bodyBytes) < previewLen {
				previewLen = len(bodyBytes)
			}
			preview := ""
			if previewLen > 0 {
				preview = string(bodyBytes[:previewLen])
			}
			log.Printf("SAM noticedesc JSON unmarshal failed: %v (preview: %s)", err, preview)
		}

		// Fallback: tolerate malformed JSON by extracting "description" manually
		if desc, ok := ExtractDescriptionJSONLike(string(bodyBytes)); ok && strings.TrimSpace(desc) != "" {
			return finalize(desc), rawJsonResponse, resp.StatusCode, contentType, nil
		}
	}
	
	// Not JSON or failed to parse, treat as plain text
	rawText := string(bodyBytes)
	rawText = finalize(rawText)
	
	// Check for "Description not found" response (even if 200)
	if strings.Contains(strings.ToLower(rawText), "description not found") {
		return rawText, rawJsonResponse, http.StatusNotFound, contentType, nil
	}
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return rawText, rawJsonResponse, resp.StatusCode, contentType, fmt.Errorf("SAM API returned status %d", resp.StatusCode)
	}
	
	return rawText, rawJsonResponse, resp.StatusCode, contentType, nil
}

// NormalizeRaw performs minimal normalization (raw post-parse)
// Converts \r\n to \n, converts standalone \r to \n, trims trailing whitespace per line
// Does NOT strip HTML tags - those are preserved for raw post-parse view
// NOTE: DEBUG_NORMALIZE_RAW should only be enabled in development/debugging scenarios
// as it logs actual description content which may contain sensitive procurement information.
func NormalizeRaw(rawText string) string {
	// Sanity check: verify we're receiving plain text, not JSON (only log if debug enabled)
	if os.Getenv("DEBUG_NORMALIZE_RAW") == "true" {
		if strings.HasPrefix(strings.TrimSpace(rawText), "{") && strings.Contains(rawText, "\"description\"") {
			log.Printf("WARNING: NormalizeRaw received JSON-like input (starts with { and contains 'description' key)")
		}
	}
	
	// Replace \r\n with \n first (handles Windows line endings)
	normalized := strings.ReplaceAll(rawText, "\r\n", "\n")
	// Convert all remaining standalone \r characters to \n (preserves line structure)
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	
	// Split into lines, clean up each line, rejoin
	lines := strings.Split(normalized, "\n")
	var cleanedLines []string
	for _, line := range lines {
		// Trim trailing whitespace
		cleaned := strings.TrimRight(line, " \t")
		cleanedLines = append(cleanedLines, cleaned)
	}
	
	result := strings.Join(cleanedLines, "\n")
	
	// Sanity check and preview logging (only if debug enabled)
	if os.Getenv("DEBUG_NORMALIZE_RAW") == "true" {
		hasCR := strings.Contains(result, "\r")
		hasLF := strings.Contains(result, "\n")
		log.Printf("NormalizeRaw: hasCR=%v hasLF=%v", hasCR, hasLF)
		if hasCR {
			log.Printf("WARNING: NormalizeRaw output still contains CR characters - normalization may not be working correctly")
		}
		
		// Log preview of normalized text to verify unwrapping worked
		previewLen := 500
		if len(result) < previewLen {
			previewLen = len(result)
		}
		if previewLen > 0 {
			log.Printf("NormalizeRaw preview (first %d chars):\n%s", previewLen, result[:previewLen])
		}
	}
	
	return result
}

// stripNonFormattingTags removes HTML tags except formatting tags (strong, b, em, i, u, br, p)
func stripNonFormattingTags(text string) string {
	// Use ReplaceAllStringFunc to process each HTML tag
	return htmlTagPattern.ReplaceAllStringFunc(text, func(tag string) string {
		// If it's a formatting tag, keep it
		if formattingTagPattern.MatchString(tag) {
			return tag
		}
		// Otherwise, replace with space
		return " "
	})
}

// Normalize performs full normalization for display/search
// Preserves HTML formatting tags (strong, b, em, i, u, br, p), strips other HTML tags, 
// applies raw normalization, then cleans up pipe patterns, drops filler lines, and collapses excessive blank lines
func Normalize(rawText string) string {
	// Strip non-formatting HTML tags first (preserve formatting tags like <strong>, <em>, etc.)
	normalized := stripNonFormattingTags(rawText)
	
	// Clean up specific HTML entity patterns like .&nbsp; → . (remove the entity, keep punctuation)
	normalized = punctuationEntityPattern.ReplaceAllString(normalized, "$1")
	
	// Decode remaining HTML entities (e.g., &rsquo; → ', &amp; → &)
	normalized = html.UnescapeString(normalized)
	
	// Then apply raw normalization (line endings, whitespace)
	normalized = NormalizeRaw(normalized)
	
	// Split into lines for processing
	lines := strings.Split(normalized, "\n")
	var processedLines []string
	blankLineCount := 0
	
	// Patterns for cleaning up pipe-related artifacts
	// Match patterns like |1|, |2|, |3|, etc. (pipe, number, pipe)
	pipeNumberPattern := regexp.MustCompile(`\|[0-9]+\|`)
	// Match patterns like || (double pipes)
	doublePipePattern := regexp.MustCompile(`\|\|+`)
	// Match lines that are only pipes/whitespace
	pipeOnlyPattern := regexp.MustCompile(`^[\s|]+$`)
	// Match pipe patterns at start/end of lines
	leadingPipePattern := regexp.MustCompile(`^\|+[\s]*`)
	trailingPipePattern := regexp.MustCompile(`[\s]*\|+$`)
	
	for _, line := range lines {
		// Drop lines that are only pipes/whitespace (filler clause table lines)
		if pipeOnlyPattern.MatchString(line) {
			continue
		}
		
		// Clean up pipe patterns within the line
		cleaned := line
		// Replace pipe-number-pipe patterns like |1|, |2|, etc. with space (prevents token concatenation)
		cleaned = pipeNumberPattern.ReplaceAllString(cleaned, " ")
		// Replace multiple consecutive pipes with single space
		cleaned = doublePipePattern.ReplaceAllString(cleaned, " ")
		// Remove leading pipes and whitespace
		cleaned = leadingPipePattern.ReplaceAllString(cleaned, "")
		// Remove trailing pipes and whitespace
		cleaned = trailingPipePattern.ReplaceAllString(cleaned, "")
		// Clean up multiple spaces (using pre-compiled pattern)
		cleaned = spacePattern.ReplaceAllString(cleaned, " ")
		// Trim whitespace
		cleaned = strings.TrimSpace(cleaned)
		
		// Track consecutive blank lines
		if cleaned == "" {
			blankLineCount++
			// Collapse 3+ blank lines to 2
			if blankLineCount <= 2 {
				processedLines = append(processedLines, "")
			}
		} else {
			blankLineCount = 0
			processedLines = append(processedLines, cleaned)
		}
	}
	
	return strings.Join(processedLines, "\n")
}

// ComputeContentHash computes SHA256 hash of text for change detection
func ComputeContentHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// AI processing configuration constants
const (
	defaultAIMaxChars = 8000
	defaultAIMaxParas = 40
)

// getAIMaxChars returns the maximum characters for AI input text (from env or default)
func getAIMaxChars() int {
	if maxStr := os.Getenv("AI_DESC_MAX_CHARS"); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil && max > 0 {
			return max
		}
	}
	return defaultAIMaxChars
}

// getAIMaxParas returns the maximum paragraphs for AI input text (from env or default)
func getAIMaxParas() int {
	if maxStr := os.Getenv("AI_DESC_MAX_PARAS"); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil && max > 0 {
			return max
		}
	}
	return defaultAIMaxParas
}

// isTableRow detects if a line is table-ish (contains | and has a first field that looks like a clause title)
func isTableRow(line string) bool {
	if !strings.Contains(line, "|") {
		return false
	}
	
	// Extract first field (everything before the first pipe)
	first := strings.TrimSpace(strings.SplitN(line, "|", 2)[0])
	
	// First field should be at least 8 characters to avoid junk
	if len(first) < 8 {
		return false
	}
	
	// First field should not be too long (likely not a clause title if > 100 chars)
	if len(first) > 100 {
		return false
	}
	
	return true
}

// parseClauseLine extracts clause titles and filters for relevance
// Extracts title as everything before the first pipe, optionally handling date patterns
func parseClauseLine(line string) (title string, isRelevant bool) {
	if !strings.Contains(line, "|") {
		return "", false
	}
	
	// Extract first field (everything before the first pipe)
	first := strings.TrimSpace(strings.SplitN(line, "|", 2)[0])
	
	// Avoid junk - first field should be at least 8 characters
	if len(first) < 8 {
		return "", false
	}
	
	// Extract title - handle date patterns like "(JAN 2023)" / "(OCT 2020)" as part of title
	// The date pattern is already part of the first field, so we just use it as-is
	title = first
	
	titleLower := strings.ToLower(title)
	
	// Keywords for relevant clauses
	relevantKeywords := []string{
		"small business", "set-aside", "set aside", "cybersecurity", "cmmc",
		"wawf", "wide area workflow", "priority rating", "payment", "certificate",
		"compliance", "delivery", "submission", "quote", "validity", "irpod",
		"do rated", "rated order", "certification", "certificate of compliance",
	}
	
	for _, keyword := range relevantKeywords {
		if strings.Contains(titleLower, keyword) {
			return title, true
		}
	}
	
	return title, false
}

// extractContacts extracts emails, phone numbers, and URLs from text
func extractContacts(text string) (emails []string, phones []string, urls []string) {
	// Email pattern
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emailMatches := emailPattern.FindAllString(text, -1)
	emails = deduplicateStrings(emailMatches)
	
	// Phone pattern (various formats)
	phonePattern := regexp.MustCompile(`(\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}|\d{3}-\d{3}-\d{4}|\d{10})`)
	phoneMatches := phonePattern.FindAllString(text, -1)
	phones = deduplicateStrings(phoneMatches)
	
	// URL pattern
	urlPattern := regexp.MustCompile(`https?://[^\s<>"{}|\\^`+"`"+`\[\]]+`)
	urlMatches := urlPattern.FindAllString(text, -1)
	urls = deduplicateStrings(urlMatches)
	
	return emails, phones, urls
}

// extractKeyFacts extracts key facts like IRPOD, quote validity, ROTIs, certificates, etc.
func extractKeyFacts(text string) (facts []string) {
	textLower := strings.ToLower(text)
	
	// IRPOD
	if strings.Contains(textLower, "irpod") || strings.Contains(textLower, "requires irpod") {
		facts = append(facts, "Requires IRPOD review")
	}
	
	// Quote validity - handle patterns like "pricing for this quotation is valid for 60 days"
	quotePattern := regexp.MustCompile(`(?i)(?:pricing\s+for\s+this\s+)?(?:quote|quotation|offer)\s+(?:is\s+)?(?:valid|validity|good)\s+(?:for\s+)?(\d+)\s*days?`)
	if matches := quotePattern.FindStringSubmatch(text); len(matches) > 1 {
		facts = append(facts, fmt.Sprintf("Quote validity: %s days", matches[1]))
	}
	
	// ROTIs - Reports of Test and Inspection (not "request for technical information")
	if strings.Contains(textLower, "rotis") || strings.Contains(textLower, "reports of test and inspection") {
		facts = append(facts, "ROTIs (Reports of Test and Inspection) required")
		// Extract lead times like "due 40 days prior to delivery"
		rotiLeadTimePattern := regexp.MustCompile(`(?i)(?:rotis?|reports\s+of\s+test\s+and\s+inspection).*?(?:due|required)\s+(\d+)\s+days?\s+prior`)
		if matches := rotiLeadTimePattern.FindStringSubmatch(text); len(matches) > 1 {
			facts = append(facts, fmt.Sprintf("ROTIs due %s days prior to delivery", matches[1]))
		}
	}
	
	// MIL-P-24503
	if strings.Contains(textLower, "mil-p-24503") || strings.Contains(textLower, "mil p 24503") {
		facts = append(facts, "MIL-P-24503 specification")
	}
	
	// Certificates
	certPattern := regexp.MustCompile(`(?i)(?:certificate|certification|cert)\s+(?:of\s+)?(?:compliance|conformance|origin|insurance)`)
	if certPattern.MatchString(text) {
		facts = append(facts, "Certificate required")
	}
	
	// DO-rated orders
	if strings.Contains(textLower, "do rated") || strings.Contains(textLower, "rated order") {
		facts = append(facts, "DO-rated order")
	}
	
	// WAWF
	if strings.Contains(textLower, "wawf") || strings.Contains(textLower, "wide area workflow") {
		facts = append(facts, "WAWF (Wide Area Workflow) required")
	}
	
	// CMMC
	if strings.Contains(textLower, "cmmc") {
		facts = append(facts, "CMMC certification required")
	}
	
	return deduplicateStrings(facts)
}

// deduplicateStrings removes duplicates while preserving order
func deduplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// scoreParagraph scores a paragraph by keyword matches (positive keywords) and penalties (negative keywords)
func scoreParagraph(para string) int {
	paraLower := strings.ToLower(para)
	score := 0
	
	// Positive keywords
	positiveKeywords := []string{
		"scope", "requirements", "delivery", "submission", "certificate",
		"quote", "valid", "due", "close", "amendment", "irpod", "wawf",
		"cmmc", "easa", "faa", "rotis", "specification", "deliverable",
		"contract", "order", "purchase", "acquisition",
	}
	
	for _, keyword := range positiveKeywords {
		if strings.Contains(paraLower, keyword) {
			score += 2
		}
	}
	
	// Penalties for boilerplate
	if isBoilerplateParagraph(para) {
		score -= 10
	}
	
	return score
}

// isBoilerplateParagraph checks for negative keywords/patterns
func isBoilerplateParagraph(para string) bool {
	paraTrimmed := strings.TrimSpace(para)
	if paraTrimmed == "" {
		return true
	}
	
	paraLower := strings.ToLower(paraTrimmed)
	
	// Check for negative keywords
	negativePatterns := []string{
		"block 1:", "dd form 1423", "inspection acceptance",
		"information regarding abbreviations",
	}
	
	for _, pattern := range negativePatterns {
		if strings.Contains(paraLower, pattern) {
			return true
		}
	}
	
	// Check if 80% uppercase and > 100 chars (often boilerplate)
	if len(paraTrimmed) > 100 {
		upperCount := 0
		letterCount := 0
		for _, r := range paraTrimmed {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				letterCount++
				if r >= 'A' && r <= 'Z' {
					upperCount++
				}
			}
		}
		if letterCount > 0 && upperCount*100/letterCount >= 80 {
			return true
		}
	}
	
	return false
}

// OptimizeForAI processes raw normalized text to create AI-ready input with structured metadata
func OptimizeForAI(rawPostParse string) (aiInputText string, excerptText string, aiMeta models.AiMeta, pocEmailPrimary *string, err error) {
	if rawPostParse == "" {
		return "", "", models.AiMeta{}, nil, nil
	}
	
	// Extract structured data from raw_post_parse (before Normalize destroys table structure)
	lines := strings.Split(rawPostParse, "\n")
	var clauseTitles []string
	var allEmails []string
	var allPhones []string
	var allURLs []string
	
	// Parse clause table lines
	for _, line := range lines {
		if title, isRelevant := parseClauseLine(line); isRelevant {
			clauseTitles = append(clauseTitles, title)
		}
	}
	
	// Extract contacts from full text
	allEmails, allPhones, allURLs = extractContacts(rawPostParse)
	
	// Set primary POC email (first email found)
	if len(allEmails) > 0 {
		pocEmailPrimary = &allEmails[0]
	}
	
	// Extract key facts
	keyFacts := extractKeyFacts(rawPostParse)
	
	// Build boilerplate-stripped text using state machine
	// Also extract useful signals from boilerplate section before dropping
	var cleanedLines []string
	var boilerplateSection []string // Collect boilerplate lines for signal extraction
	inBoilerplate := false
	boilerplateEnterPattern := regexp.MustCompile(`(?i)information regarding abbreviations.*dd form 1423`)
	boilerplateExitPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)date of first submission`),
		regexp.MustCompile(`(?i)submit at the time of material delivery`),
		regexp.MustCompile(`(?i)certificate of compliance`),
	}
	
	for _, line := range lines {
		// Check for boilerplate entry
		if boilerplateEnterPattern.MatchString(line) {
			inBoilerplate = true
			boilerplateSection = []string{} // Reset boilerplate section
			continue
		}
		
		// Check for boilerplate exit
		if inBoilerplate {
			shouldExit := false
			for _, exitPattern := range boilerplateExitPatterns {
				if exitPattern.MatchString(line) {
					shouldExit = true
					break
				}
			}
			if shouldExit {
				// Extract useful signals from boilerplate section before exiting
				boilerplateText := strings.Join(boilerplateSection, "\n")
				boilerplateTextLower := strings.ToLower(boilerplateText)
				
				// Extract NOFORN / Need-to-know / foreign nationals restrictions
				if strings.Contains(boilerplateTextLower, "noforn") {
					keyFacts = append(keyFacts, "NOFORN restrictions apply")
				}
				if strings.Contains(boilerplateTextLower, "need-to-know") || strings.Contains(boilerplateTextLower, "need to know") {
					keyFacts = append(keyFacts, "Need-to-know restrictions apply")
				}
				if strings.Contains(boilerplateTextLower, "foreign national") {
					keyFacts = append(keyFacts, "Foreign nationals restrictions may apply")
				}
				
				inBoilerplate = false
				boilerplateSection = nil // Clear after processing
				cleanedLines = append(cleanedLines, line)
				continue
			}
			
			// While in boilerplate mode, collect lines for signal extraction but skip them from output
			boilerplateSection = append(boilerplateSection, line)
			continue // Skip ALL lines while in boilerplate mode
		}
		
		// Not in boilerplate mode, keep the line
		cleanedLines = append(cleanedLines, line)
	}
	
	// Build paragraphs from lines (handles single-newline format)
	// Accumulate lines until a blank line or heading marker
	headingPattern := regexp.MustCompile(`^\d+\.\s+`) // Lines starting with "1. ", "2. ", etc.
	var paragraphs []string
	var currentPara []string
	
	for _, line := range cleanedLines {
		lineTrimmed := strings.TrimSpace(line)
		
		// Check if line is a heading marker
		isHeading := headingPattern.MatchString(lineTrimmed)
		
		// Check if line is all-caps and short (likely a heading)
		if !isHeading && len(lineTrimmed) > 0 && len(lineTrimmed) < 80 {
			upperCount := 0
			letterCount := 0
			for _, r := range lineTrimmed {
				if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
					letterCount++
					if r >= 'A' && r <= 'Z' {
						upperCount++
					}
				}
			}
			if letterCount > 0 && upperCount*100/letterCount >= 80 {
				isHeading = true
			}
		}
		
		// If blank line or heading, finalize current paragraph
		if lineTrimmed == "" || isHeading {
			if len(currentPara) > 0 {
				paraText := strings.Join(currentPara, "\n")
				if strings.TrimSpace(paraText) != "" {
					paragraphs = append(paragraphs, paraText)
				}
				currentPara = []string{}
			}
			// If it's a heading, start a new paragraph with it
			if isHeading && lineTrimmed != "" {
				currentPara = append(currentPara, lineTrimmed)
			}
		} else {
			// Add line to current paragraph
			currentPara = append(currentPara, lineTrimmed)
		}
	}
	
	// Don't forget the last paragraph
	if len(currentPara) > 0 {
		paraText := strings.Join(currentPara, "\n")
		if strings.TrimSpace(paraText) != "" {
			paragraphs = append(paragraphs, paraText)
		}
	}
	
	// Score paragraphs
	type scoredPara struct {
		text  string
		score int
	}
	var scoredParagraphs []scoredPara
	
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		score := scoreParagraph(para)
		scoredParagraphs = append(scoredParagraphs, scoredPara{text: para, score: score})
	}
	
	// Sort by score (descending) and take top paragraphs
	// Simple bubble sort (fine for small lists)
	for i := 0; i < len(scoredParagraphs)-1; i++ {
		for j := 0; j < len(scoredParagraphs)-i-1; j++ {
			if scoredParagraphs[j].score < scoredParagraphs[j+1].score {
				scoredParagraphs[j], scoredParagraphs[j+1] = scoredParagraphs[j+1], scoredParagraphs[j]
			}
		}
	}
	
	// Select top paragraphs up to max chars (apply cap AFTER assembling header)
	maxChars := getAIMaxChars()
	maxParas := getAIMaxParas()
	
	var selectedParagraphs []string
	totalChars := 0
	headerText := "KEY FACTS:\n" + strings.Join(keyFacts, "\n") + "\n\nRELEVANT EXCERPT:\n"
	headerChars := len(headerText)
	
	// Reserve space for header
	availableChars := maxChars - headerChars
	
	for i, sp := range scoredParagraphs {
		if i >= maxParas {
			break
		}
		if sp.score <= 0 {
			break // Stop at negative or zero scores
		}
		paraLen := len(sp.text)
		if totalChars+paraLen > availableChars {
			break
		}
		selectedParagraphs = append(selectedParagraphs, sp.text)
		totalChars += paraLen + 2 // +2 for \n\n
	}
	
	// Build final AI input text
	aiInputText = headerText + strings.Join(selectedParagraphs, "\n\n")
	
	// Generate excerpt text (first 800-1200 chars of best paragraphs)
	excerptTarget := 1000 // Target 1000 chars
	if len(selectedParagraphs) > 0 {
		excerptBuilder := strings.Builder{}
		for _, para := range selectedParagraphs {
			if excerptBuilder.Len() >= excerptTarget {
				break
			}
			if excerptBuilder.Len() > 0 {
				excerptBuilder.WriteString("\n\n")
			}
			remaining := excerptTarget - excerptBuilder.Len()
			if len(para) <= remaining {
				excerptBuilder.WriteString(para)
			} else {
				excerptBuilder.WriteString(para[:remaining-3])
				excerptBuilder.WriteString("...")
				break
			}
		}
		excerptText = excerptBuilder.String()
	}
	
	// Extract actual certificate requirements from text
	var certsRequired []string
	certPattern := regexp.MustCompile(`(?i)(?:certificate|certification|cert)\s+(?:of\s+)?(?:compliance|conformance|origin|insurance|quality)`)
	certMatches := certPattern.FindAllString(rawPostParse, -1)
	for _, match := range certMatches {
		// Normalize and deduplicate
		matchLower := strings.ToLower(strings.TrimSpace(match))
		found := false
		for _, existing := range certsRequired {
			if strings.ToLower(existing) == matchLower {
				found = true
				break
			}
		}
		if !found {
			certsRequired = append(certsRequired, strings.TrimSpace(match))
		}
	}
	
	// Populate aiMeta
	aiMeta = models.AiMeta{
		POCEmails:        allEmails,
		POCPhones:        allPhones,
		ImportantURLs:    allURLs,
		ClausesKept:      clauseTitles, // Store clause titles separately
		CertsRequired:    certsRequired, // Actual certificate requirements extracted from text
		KeyRequirements:  keyFacts,
	}
	
	// Detect set-aside
	setAsidePattern := regexp.MustCompile(`(?i)(?:set[-\s]?aside|small\s+business)\s*:?\s*([^\n]+)`)
	if matches := setAsidePattern.FindStringSubmatch(rawPostParse); len(matches) > 1 {
		setAside := strings.TrimSpace(matches[1])
		aiMeta.SetAsideDetected = &setAside
	}
	
	// Detect WAWF requirement
	if strings.Contains(strings.ToLower(rawPostParse), "wawf") || strings.Contains(strings.ToLower(rawPostParse), "wide area workflow") {
		wawfRequired := true
		aiMeta.WAWFRequired = &wawfRequired
	}
	
	// Detect DO-rated
	if strings.Contains(strings.ToLower(rawPostParse), "do rated") || strings.Contains(strings.ToLower(rawPostParse), "rated order") {
		doRated := true
		aiMeta.DORated = &doRated
	}
	
	// Detect IRPOD requirement
	if strings.Contains(strings.ToLower(rawPostParse), "irpod") {
		irpodRequired := true
		aiMeta.RequiresIRPODReview = &irpodRequired
	}
	
	// Extract quote validity days - handle patterns like "pricing for this quotation is valid for 60 days"
	quoteValPattern := regexp.MustCompile(`(?i)(?:pricing\s+for\s+this\s+)?(?:quote|quotation|offer)\s+(?:is\s+)?(?:valid|validity|good)\s+(?:for\s+)?(\d+)\s*days?`)
	if matches := quoteValPattern.FindStringSubmatch(rawPostParse); len(matches) > 1 {
		if days, err := strconv.Atoi(matches[1]); err == nil {
			aiMeta.QuoteValidityDays = &days
		}
	}
	
	return aiInputText, excerptText, aiMeta, pocEmailPrimary, nil
}

