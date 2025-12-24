package main

import (
	"fmt"
	"time"
)

func convertDateFormat(dateStr string) (string, error) {
	// Try parsing as MM/DD/YYYY first
	if t, err := time.Parse("01/02/2006", dateStr); err == nil {
		return t.Format("2006-01-02"), nil
	}
	// Try parsing as YYYY-MM-DD (already in correct format)
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t.Format("2006-01-02"), nil
	}
	// Return original if we can't parse (let database handle it)
	return dateStr, fmt.Errorf("unable to parse date: %s", dateStr)
}

func main() {
	testCases := []string{
		"12/01/2025",
		"12/23/2025",
		"2025-12-01",
		"2025-12-23",
	}

	for _, tc := range testCases {
		result, err := convertDateFormat(tc)
		if err != nil {
			fmt.Printf("❌ %s -> Error: %v\n", tc, err)
		} else {
			fmt.Printf("✅ %s -> %s\n", tc, result)
		}
	}
}

