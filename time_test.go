package apikit

import (
	"strings"
	"testing"
	"time"
)

func TestNewTimeFromString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		// RFC3339 format
		{"RFC3339", "2024-01-15T10:30:00Z", false},
		{"RFC3339 with timezone", "2024-01-15T10:30:00-05:00", false},

		// ISO 8601 variants
		{"ISO without timezone", "2024-01-15T10:30:00", false},
		{"ISO with milliseconds", "2024-01-15T10:30:00.123", false},
		{"ISO with milliseconds and Z", "2024-01-15T10:30:00.999Z", false},
		{"ISO with milliseconds and timezone", "2024-01-15T10:30:00.999-07:00", false},

		// Date with space separator
		{"datetime with space", "2024-01-15 10:30:00", false},

		// Date only
		{"date only", "2024-01-15", false},

		// Invalid formats
		{"invalid format", "not a date", true},
		{"empty string", "", true},
		{"partial date", "2024-01", true},
		{"time only", "10:30:00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewTimeFromString(tt.input)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
				}
				if !result.IsZero() {
					t.Errorf("expected zero time for invalid input, got %v", result)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if result.IsZero() {
					t.Errorf("expected non-zero time for valid input %q", tt.input)
				}
			}
		})
	}
}

func TestNewTimeFromString_ParsedValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "RFC3339 UTC",
			input:    "2024-01-15T10:30:00Z",
			expected: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:     "date only",
			input:    "2024-12-25",
			expected: time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewTimeFromString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Equal(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNewTimeFromString_ErrorMessage(t *testing.T) {
	_, err := NewTimeFromString("invalid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMsg := "unable to parse time"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error message to contain %q, got %q", expectedMsg, err.Error())
	}
}

func TestCommonTimeFormats(t *testing.T) {
	// Verify that CommonTimeFormats is exported and has expected formats
	if len(CommonTimeFormats) == 0 {
		t.Error("CommonTimeFormats should not be empty")
	}

	// Check for key formats
	expectedFormats := []string{
		time.RFC3339,
		"2006-01-02",
	}

	for _, expected := range expectedFormats {
		found := false
		for _, format := range CommonTimeFormats {
			if format == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected format %q to be in CommonTimeFormats", expected)
		}
	}
}

func TestNewTimeFromString_AllFormats(t *testing.T) {
	// Test that each format in CommonTimeFormats can parse a valid time
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	for _, format := range CommonTimeFormats {
		t.Run(format, func(t *testing.T) {
			// Format the test time using this format
			formatted := testTime.Format(format)

			// Try to parse it back
			parsed, err := NewTimeFromString(formatted)
			if err != nil {
				t.Errorf("failed to parse time formatted with %q: %v", format, err)
			}

			// Note: We can't check exact equality because some formats lose precision
			// (e.g., date-only formats lose time information)
			if parsed.IsZero() {
				t.Errorf("parsed time should not be zero for format %q", format)
			}
		})
	}
}

func TestNewTimeFromString_Timezone(t *testing.T) {
	// Test that timezone information is preserved
	input := "2024-01-15T10:30:00-05:00"
	result, err := NewTimeFromString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The time should be equivalent to UTC time
	expectedUTC := time.Date(2024, 1, 15, 15, 30, 0, 0, time.UTC)
	if !result.Equal(expectedUTC) {
		t.Errorf("expected %v (UTC), got %v", expectedUTC, result)
	}
}
