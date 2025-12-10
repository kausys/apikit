// Package apikit provides runtime helpers for APIKit-generated code
package apikit

import (
	"fmt"
	"time"
)

// CommonTimeFormats are the formats tried by NewTimeFromString
var CommonTimeFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05.999",
	"2006-01-02T15:04:05.999Z",
	"2006-01-02T15:04:05.999-07:00",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// NewTimeFromString parses a time string using common formats
// This function is used by APIKit-generated code to parse time.Time fields
//
// Supported formats (tried in order):
//   - RFC3339: "2006-01-02T15:04:05Z07:00"
//   - "2006-01-02T15:04:05"
//   - "2006-01-02T15:04:05.999"
//   - "2006-01-02T15:04:05.999Z"
//   - "2006-01-02T15:04:05.999-07:00"
//   - "2006-01-02 15:04:05"
//   - "2006-01-02"
//
// Returns the parsed time.Time or an error if no format matches
func NewTimeFromString(s string) (time.Time, error) {
	for _, format := range CommonTimeFormats {
		t, err := time.Parse(format, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: expected RFC3339 or common date format")
}
