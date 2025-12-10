package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile_QuotedHeaderNames(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.go")

	content := `package test

import (
	"context"
	"net/http"
)

// GetUserRequest represents the request with various header formats
type GetUserRequest struct {
	// in:header 'User-Agent'
	UserAgent string

	// in:header 'X-Request-ID'
	RequestID string

	// in:header X-API-Key
	APIKey string

	// in:header 'Content-Type'
	ContentType string

	// in:header 'X-Custom Header'
	CustomHeader string
}

// GetUserResponse represents the response
type GetUserResponse struct {
	Message string ` + "`" + `json:"message"` + "`" + `
}

// apikit:handler
func GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	return &GetUserResponse{Message: "ok"}, nil
}
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	p := New()
	result, err := p.ParseFile(testFile)

	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(result.Handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(result.Handlers))
	}

	reqStruct := result.Structs["GetUserRequest"]
	if reqStruct == nil {
		t.Fatal("expected GetUserRequest struct")
	}

	// Test cases for each field
	tests := []struct {
		fieldIndex       int
		expectedName     string
		expectedInSource string
		expectedInName   string
	}{
		{
			fieldIndex:       0,
			expectedName:     "UserAgent",
			expectedInSource: "header",
			expectedInName:   "User-Agent",
		},
		{
			fieldIndex:       1,
			expectedName:     "RequestID",
			expectedInSource: "header",
			expectedInName:   "X-Request-ID",
		},
		{
			fieldIndex:       2,
			expectedName:     "APIKey",
			expectedInSource: "header",
			expectedInName:   "X-API-Key",
		},
		{
			fieldIndex:       3,
			expectedName:     "ContentType",
			expectedInSource: "header",
			expectedInName:   "Content-Type",
		},
		{
			fieldIndex:       4,
			expectedName:     "CustomHeader",
			expectedInSource: "header",
			expectedInName:   "X-Custom Header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			if tt.fieldIndex >= len(reqStruct.Fields) {
				t.Fatalf("field index %d out of range (total fields: %d)", tt.fieldIndex, len(reqStruct.Fields))
			}

			field := reqStruct.Fields[tt.fieldIndex]

			if field.Name != tt.expectedName {
				t.Errorf("expected field name %q, got %q", tt.expectedName, field.Name)
			}

			if field.InComment != tt.expectedInSource {
				t.Errorf("expected InComment %q, got %q", tt.expectedInSource, field.InComment)
			}

			if field.InCommentName != tt.expectedInName {
				t.Errorf("expected InCommentName %q, got %q", tt.expectedInName, field.InCommentName)
			}
		})
	}
}

