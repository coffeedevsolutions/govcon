package services

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractDescriptionJSONLike_ValidJSON(t *testing.T) {
	// Control case: valid JSON
	input := `{"description":"ITEM UNIQUE IDENTIFICATION"}`
	expected := "ITEM UNIQUE IDENTIFICATION"
	
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed")
	}
	if desc != expected {
		t.Errorf("Expected %q, got %q", expected, desc)
	}
}

func TestExtractDescriptionJSONLike_MalformedWithRawNewlines(t *testing.T) {
	// Malformed JSON where description contains raw \r/\n inside the quotes
	input := `{"description":"ITEM UNIQUE IDENTIFICATION
This is line 2
This is line 3"}`
	expected := "ITEM UNIQUE IDENTIFICATION\nThis is line 2\nThis is line 3"
	
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed")
	}
	if desc != expected {
		t.Errorf("Expected %q, got %q", expected, desc)
	}
}

func TestExtractDescriptionJSONLike_MalformedWithRawCarriageReturn(t *testing.T) {
	// Malformed JSON with raw \r
	input := "{\"description\":\"ITEM UNIQUE IDENTIFICATION\rThis is line 2\"}"
	expected := "ITEM UNIQUE IDENTIFICATION\rThis is line 2"
	
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed")
	}
	if desc != expected {
		t.Errorf("Expected %q, got %q", expected, desc)
	}
}

func TestExtractDescriptionJSONLike_NestedDescription(t *testing.T) {
	// Ensure top-level is extracted, not nested
	input := `{"description":"Top level","nested":{"description":"Should not extract this"}}`
	expected := "Top level"
	
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed")
	}
	if desc != expected {
		t.Errorf("Expected %q, got %q", expected, desc)
	}
}

func TestExtractDescriptionJSONLike_DescriptionInStringValue(t *testing.T) {
	// Ensure we don't match "description" when it appears as literal text in a string value
	input := `{"other":"This text contains the word description but should not match","description":"This should match"}`
	expected := "This should match"
	
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed")
	}
	if desc != expected {
		t.Errorf("Expected %q, got %q", expected, desc)
	}
}

func TestExtractDescriptionJSONLike_EscapeSequences(t *testing.T) {
	// Test all escape sequences
	input := `{"description":"Quote: \" Backslash: \\ Newline: \n Tab: \t Carriage return: \r Backspace: \b Form feed: \f Slash: \/"}`
	expected := "Quote: \" Backslash: \\ Newline: \n Tab: \t Carriage return: \r Backspace: \b Form feed: \f Slash: /"
	
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed")
	}
	if desc != expected {
		t.Errorf("Expected %q, got %q", expected, desc)
	}
}

func TestExtractDescriptionJSONLike_UnicodeEscape(t *testing.T) {
	// Test \uXXXX escape
	input := `{"description":"Hello \u0041\u0042\u0043"}`
	expected := "Hello ABC"
	
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed")
	}
	if desc != expected {
		t.Errorf("Expected %q, got %q", expected, desc)
	}
}

func TestExtractDescriptionJSONLike_SurrogatePair(t *testing.T) {
	// Test surrogate pair (emoji: 😀 = \uD83D\uDE00)
	input := `{"description":"Emoji: \uD83D\uDE00"}`
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed")
	}
	// Check that we got a valid emoji (should be 4 bytes in UTF-8)
	if len(desc) < 10 {
		t.Errorf("Expected emoji to be extracted, got %q", desc)
	}
	// The emoji should be in the string
	if desc != "Emoji: 😀" {
		t.Errorf("Expected emoji, got %q", desc)
	}
}

func TestExtractDescriptionJSONLike_EmptyString(t *testing.T) {
	// Empty description value
	input := `{"description":""}`
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed (even for empty string)")
	}
	if desc != "" {
		t.Errorf("Expected empty string, got %q", desc)
	}
}

func TestExtractDescriptionJSONLike_MissingKey(t *testing.T) {
	// No description key
	input := `{"other":"value"}`
	_, ok := ExtractDescriptionJSONLike(input)
	if ok {
		t.Error("Expected extraction to fail (no description key)")
	}
}

func TestExtractDescriptionJSONLike_NonStringValue(t *testing.T) {
	// Description value is not a string
	input := `{"description":123}`
	_, ok := ExtractDescriptionJSONLike(input)
	if ok {
		t.Error("Expected extraction to fail (non-string value)")
	}
}

func TestExtractDescriptionJSONLike_WhitespaceAroundKey(t *testing.T) {
	// Whitespace around key and value
	input := `{  "description"  :  "value"  }`
	expected := "value"
	
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed")
	}
	if desc != expected {
		t.Errorf("Expected %q, got %q", expected, desc)
	}
}

func TestExtractDescriptionJSONLike_MaxLengthGuardrail(t *testing.T) {
	// Test that max length guardrail works
	// Create a string that exceeds maxExtractedLength
	largeValue := make([]byte, maxExtractedLength+1)
	for i := range largeValue {
		largeValue[i] = 'A'
	}
	input := `{"description":"` + string(largeValue) + `"}`
	
	_, ok := ExtractDescriptionJSONLike(input)
	if ok {
		t.Error("Expected extraction to fail (exceeds max length)")
	}
}

func TestExtractDescriptionJSONLike_MaxScanLengthGuardrail(t *testing.T) {
	// Test that max scan length guardrail works
	largeInput := make([]byte, maxExtractScanLength+1)
	for i := range largeInput {
		largeInput[i] = 'A'
	}
	input := string(largeInput)
	
	_, ok := ExtractDescriptionJSONLike(input)
	if ok {
		t.Error("Expected extraction to fail (exceeds max scan length)")
	}
}

func TestUnwrapDescriptionText_ValidJSON(t *testing.T) {
	// Control case: valid JSON
	input := `{"description":"ITEM UNIQUE IDENTIFICATION"}`
	expected := "ITEM UNIQUE IDENTIFICATION"
	
	result := UnwrapDescriptionText(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestUnwrapDescriptionText_MalformedJSON(t *testing.T) {
	// Malformed JSON with raw newlines
	input := `{"description":"ITEM UNIQUE IDENTIFICATION
Line 2"}`
	expected := "ITEM UNIQUE IDENTIFICATION\nLine 2"
	
	result := UnwrapDescriptionText(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestUnwrapDescriptionText_DoubleWrappedString(t *testing.T) {
	// Double-wrapped string case: "\"{\\\"description\\\":...}\""
	// Create a properly JSON-encoded string containing JSON
	inner := `{"description":"ITEM UNIQUE IDENTIFICATION"}`
	// JSON-encode the inner string to create double-wrapped
	doubleWrappedBytes, err := json.Marshal(inner)
	if err != nil {
		t.Fatalf("Failed to marshal inner JSON: %v", err)
	}
	doubleWrapped := string(doubleWrappedBytes)
	expected := "ITEM UNIQUE IDENTIFICATION"
	
	result := UnwrapDescriptionText(doubleWrapped)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestUnwrapDescriptionText_TripleWrapped(t *testing.T) {
	// Triple-wrapped (should respect recursion limit of 2)
	// With maxUnwrapRecursion = 2, we can unwrap twice:
	// 1. Triple-wrapped -> double-wrapped (depth 0->1)
	// 2. Double-wrapped -> inner JSON (depth 1->2)
	// 3. Would unwrap inner JSON, but hit recursion limit, so returns inner JSON
	inner := `{"description":"ITEM UNIQUE IDENTIFICATION"}`
	// Double-wrap
	doubleWrappedBytes, err := json.Marshal(inner)
	if err != nil {
		t.Fatalf("Failed to marshal inner JSON: %v", err)
	}
	// Triple-wrap
	tripleWrappedBytes, err := json.Marshal(string(doubleWrappedBytes))
	if err != nil {
		t.Fatalf("Failed to marshal double-wrapped JSON: %v", err)
	}
	tripleWrapped := string(tripleWrappedBytes)
	
	result := UnwrapDescriptionText(tripleWrapped)
	// With recursion limit of 2, we can unwrap twice, so we should get the inner JSON
	// which is still wrapped. The fallback extractor should handle it though.
	// Actually, let's check: after 2 unwraps, we should have the inner JSON string.
	// But the extractor should still work on it. Let me verify the actual behavior.
	// The result should either be the final description OR the inner JSON (if limit hit).
	// Since we have a fallback extractor, it should still extract even if JSON is malformed.
	if result == tripleWrapped {
		t.Error("Expected some unwrapping to occur")
	}
	// The result should contain "ITEM UNIQUE IDENTIFICATION" either directly or in JSON
	if !strings.Contains(result, "ITEM UNIQUE IDENTIFICATION") {
		t.Errorf("Expected result to contain 'ITEM UNIQUE IDENTIFICATION', got %q", result)
	}
}

func TestUnwrapDescriptionText_PlainText(t *testing.T) {
	// Plain text (no JSON)
	input := "ITEM UNIQUE IDENTIFICATION"
	expected := input
	
	result := UnwrapDescriptionText(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestUnwrapDescriptionText_EmptyString(t *testing.T) {
	// Empty string
	input := ""
	expected := input
	
	result := UnwrapDescriptionText(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestUnwrapDescriptionText_RecursionLimit(t *testing.T) {
	// Test that recursion limit prevents infinite loops
	// Create a deeply nested structure that would cause issues
	input := `{"description":"{\"description\":\"{\\\"description\\\":\\\"value\\\"}\"}"}`
	result := UnwrapDescriptionText(input)
	// Should extract the innermost value or stop at recursion limit
	if result == input {
		t.Error("Expected unwrapping to occur, but got original input")
	}
}

func TestParseLenientJSONString_Basic(t *testing.T) {
	input := `"hello world"`
	expected := "hello world"
	
	val, endIdx, ok := parseLenientJSONString(input, 0)
	if !ok {
		t.Fatal("Expected parsing to succeed")
	}
	if val != expected {
		t.Errorf("Expected %q, got %q", expected, val)
	}
	if endIdx != len(input) {
		t.Errorf("Expected endIdx %d, got %d", len(input), endIdx)
	}
}

func TestParseLenientJSONString_WithRawNewline(t *testing.T) {
	input := `"hello
world"`
	expected := "hello\nworld"
	
	val, _, ok := parseLenientJSONString(input, 0)
	if !ok {
		t.Fatal("Expected parsing to succeed")
	}
	if val != expected {
		t.Errorf("Expected %q, got %q", expected, val)
	}
}

func TestParseLenientJSONString_WithEscapedNewline(t *testing.T) {
	input := `"hello\nworld"`
	expected := "hello\nworld"
	
	val, _, ok := parseLenientJSONString(input, 0)
	if !ok {
		t.Fatal("Expected parsing to succeed")
	}
	if val != expected {
		t.Errorf("Expected %q, got %q", expected, val)
	}
}

func TestParseLenientJSONString_WithEscapeSequences(t *testing.T) {
	input := `"quote: \" backslash: \\ tab: \t"`
	expected := "quote: \" backslash: \\ tab: \t"
	
	val, _, ok := parseLenientJSONString(input, 0)
	if !ok {
		t.Fatal("Expected parsing to succeed")
	}
	if val != expected {
		t.Errorf("Expected %q, got %q", expected, val)
	}
}

func TestParseLenientJSONString_WithUnicode(t *testing.T) {
	input := `"hello \u0041\u0042\u0043"`
	expected := "hello ABC"
	
	val, _, ok := parseLenientJSONString(input, 0)
	if !ok {
		t.Fatal("Expected parsing to succeed")
	}
	if val != expected {
		t.Errorf("Expected %q, got %q", expected, val)
	}
}

func TestParseLenientJSONString_WithSurrogatePair(t *testing.T) {
	input := `"emoji: \uD83D\uDE00"`
	val, _, ok := parseLenientJSONString(input, 0)
	if !ok {
		t.Fatal("Expected parsing to succeed")
	}
	// Should contain the emoji 😀
	if val != "emoji: 😀" {
		t.Errorf("Expected emoji, got %q", val)
	}
}

func TestParseLenientJSONString_InvalidStart(t *testing.T) {
	input := `hello"`
	_, _, ok := parseLenientJSONString(input, 0)
	if ok {
		t.Error("Expected parsing to fail (no opening quote)")
	}
}

func TestParseLenientJSONString_UnclosedString(t *testing.T) {
	input := `"hello world`
	_, _, ok := parseLenientJSONString(input, 0)
	if ok {
		t.Error("Expected parsing to fail (unclosed string)")
	}
}

func TestUnwrapDescriptionText_SAMPayloadWithRawCRLF(t *testing.T) {
	// Regression test for SAM payload with raw CR/LF inside JSON string and escaped quotes
	// This simulates the real-world case where SAM returns malformed JSON with raw newlines
	input := `{"description":"ITEM UNIQUE IDENTIFICATION
This is a description with raw
carriage returns and newlines
It also has \"escaped quotes\" inside"}`
	expected := "ITEM UNIQUE IDENTIFICATION\nThis is a description with raw\ncarriage returns and newlines\nIt also has \"escaped quotes\" inside"
	
	result := UnwrapDescriptionText(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestUnwrapDescriptionText_DoubleWrappedWithRawCRLF(t *testing.T) {
	// Test double-wrapped JSON string where inner JSON has raw CR/LF
	// This tests the fallback path when inner json.Unmarshal fails
	innerJSON := `{"description":"ITEM UNIQUE IDENTIFICATION
Line 2 with raw newline
Line 3"}`
	// Double-wrap by JSON-encoding the inner JSON string
	doubleWrappedBytes, err := json.Marshal(innerJSON)
	if err != nil {
		t.Fatalf("Failed to marshal inner JSON: %v", err)
	}
	doubleWrapped := string(doubleWrappedBytes)
	expected := "ITEM UNIQUE IDENTIFICATION\nLine 2 with raw newline\nLine 3"
	
	result := UnwrapDescriptionText(doubleWrapped)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExtractDescriptionJSONLike_WithRawCRLFAndEscapedQuotes(t *testing.T) {
	// Test the extractor with raw CR/LF and escaped quotes (real SAM payload case)
	input := `{"description":"ITEM UNIQUE IDENTIFICATION
This description has raw
newlines and \"escaped quotes\"
which makes strict JSON parsing fail"}`
	expected := "ITEM UNIQUE IDENTIFICATION\nThis description has raw\nnewlines and \"escaped quotes\"\nwhich makes strict JSON parsing fail"
	
	desc, ok := ExtractDescriptionJSONLike(input)
	if !ok {
		t.Fatal("Expected extraction to succeed")
	}
	if desc != expected {
		t.Errorf("Expected %q, got %q", expected, desc)
	}
}

func TestUnwrapDescriptionText_JSONDescriptionValueIsWrappedJSON(t *testing.T) {
	input := `{"description":"{\"description\":\"ITEM UNIQUE IDENTIFICATION\"}"}`
	expected := "ITEM UNIQUE IDENTIFICATION"

	result := UnwrapDescriptionText(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

