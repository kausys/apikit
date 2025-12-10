package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	p := New()

	if p == nil {
		t.Fatal("expected parser to be created")
	}

	if p.fset == nil {
		t.Error("expected fset to be initialized")
	}

	if p.structs == nil {
		t.Error("expected structs map to be initialized")
	}

	if p.externalStructs == nil {
		t.Error("expected externalStructs map to be initialized")
	}
}

func TestParseFile_SimpleHandler(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.go")

	content := `package test

import "context"

// CreateUserRequest represents the request
type CreateUserRequest struct {
	Name  string ` + "`" + `json:"name" validate:"required"` + "`" + `
	Email string ` + "`" + `json:"email" validate:"required,email"` + "`" + `
}

// CreateUserResponse represents the response
type CreateUserResponse struct {
	ID   int    ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// apikit:handler
func CreateUser(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
	return CreateUserResponse{}, nil
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

	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	// Check handlers
	if len(result.Handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(result.Handlers))
	}

	handler := result.Handlers[0]
	if handler.Name != "CreateUser" {
		t.Errorf("expected handler name 'CreateUser', got %q", handler.Name)
	}

	if handler.Package != "test" {
		t.Errorf("expected package 'test', got %q", handler.Package)
	}

	if handler.ParamType != "CreateUserRequest" {
		t.Errorf("expected param type 'CreateUserRequest', got %q", handler.ParamType)
	}

	if handler.ReturnType != "CreateUserResponse" {
		t.Errorf("expected return type 'CreateUserResponse', got %q", handler.ReturnType)
	}

	// Check structs
	if len(result.Structs) != 2 {
		t.Fatalf("expected 2 structs, got %d", len(result.Structs))
	}

	reqStruct, ok := result.Structs["CreateUserRequest"]
	if !ok {
		t.Fatal("expected CreateUserRequest struct to be parsed")
	}

	if len(reqStruct.Fields) != 2 {
		t.Errorf("expected 2 fields in CreateUserRequest, got %d", len(reqStruct.Fields))
	}

	// Check first field
	if reqStruct.Fields[0].Name != "Name" {
		t.Errorf("expected first field name 'Name', got %q", reqStruct.Fields[0].Name)
	}

	if reqStruct.Fields[0].Type != "string" {
		t.Errorf("expected first field type 'string', got %q", reqStruct.Fields[0].Type)
	}
}

func TestParseFile_WithPathAndQuery(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.go")

	content := `package test

import "context"

type GetUserRequest struct {
	UserID string ` + "`" + `path:"userId"` + "`" + `
	Filter string ` + "`" + `query:"filter"` + "`" + `
}

type GetUserResponse struct {
	ID   int    ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// apikit:handler
func GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	return GetUserResponse{}, nil
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

	// Check path tag
	userIDField := reqStruct.Fields[0]
	if userIDField.Name != "UserID" {
		t.Errorf("expected field name 'UserID', got %q", userIDField.Name)
	}
}

func TestParseFile_InvalidFile(t *testing.T) {
	p := New()
	_, err := p.ParseFile("nonexistent.go")

	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseFile_NoHandlers(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "nohandler.go")

	content := `package test

type User struct {
	Name string
}

func RegularFunction() {
	// Not a handler
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

	if len(result.Handlers) != 0 {
		t.Errorf("expected 0 handlers, got %d", len(result.Handlers))
	}
}

func TestParseFile_WithPointerFields(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.go")

	content := `package test

import "context"

type UpdateUserRequest struct {
	Name  *string ` + "`" + `json:"name"` + "`" + `
	Age   *int    ` + "`" + `json:"age"` + "`" + `
}

type UpdateUserResponse struct {
	Success bool ` + "`" + `json:"success"` + "`" + `
}

// apikit:handler
func UpdateUser(ctx context.Context, req UpdateUserRequest) (UpdateUserResponse, error) {
	return UpdateUserResponse{}, nil
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

	reqStruct := result.Structs["UpdateUserRequest"]
	if reqStruct == nil {
		t.Fatal("expected UpdateUserRequest struct")
	}

	// Check pointer field
	nameField := reqStruct.Fields[0]
	if !nameField.IsPointer {
		t.Error("expected Name field to be marked as pointer")
	}

	if nameField.Type != "*string" {
		t.Errorf("expected type '*string', got %q", nameField.Type)
	}
}

func TestParseFile_WithSliceFields(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.go")

	content := `package test

import "context"

type SearchRequest struct {
	Tags []string ` + "`" + `query:"tags"` + "`" + `
	IDs  []int    ` + "`" + `query:"ids"` + "`" + `
}

type SearchResponse struct {
	Results []string ` + "`" + `json:"results"` + "`" + `
}

// apikit:handler
func Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	return SearchResponse{}, nil
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

	reqStruct := result.Structs["SearchRequest"]
	if reqStruct == nil {
		t.Fatal("expected SearchRequest struct")
	}

	// Check slice field
	tagsField := reqStruct.Fields[0]
	if !tagsField.IsSlice {
		t.Error("expected Tags field to be marked as slice")
	}

	if tagsField.SliceType != "string" {
		t.Errorf("expected slice type 'string', got %q", tagsField.SliceType)
	}
}

func TestParseFile_WithCommentAnnotations(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.go")

	content := `package test

import "context"

type GetUserRequest struct {
	// in:path userId
	UserID string
	// in:query filter
	Filter string
}

type GetUserResponse struct {
	ID   int    ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// apikit:handler
func GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	return GetUserResponse{}, nil
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

	reqStruct := result.Structs["GetUserRequest"]
	if reqStruct == nil {
		t.Fatal("expected GetUserRequest struct")
	}

	// Check comment-based annotations
	userIDField := reqStruct.Fields[0]
	if userIDField.InComment != "path" {
		t.Errorf("expected InComment 'path', got %q", userIDField.InComment)
	}

	if userIDField.InCommentName != "userId" {
		t.Errorf("expected InCommentName 'userId', got %q", userIDField.InCommentName)
	}
}

func TestExtractInComment(t *testing.T) {
	tests := []struct {
		name           string
		comment        string
		expectedSource string
		expectedName   string
	}{
		{
			name:           "query without name",
			comment:        "// in:query",
			expectedSource: "query",
			expectedName:   "",
		},
		{
			name:           "path with name",
			comment:        "// in:path userId",
			expectedSource: "path",
			expectedName:   "userId",
		},
		{
			name:           "header without quotes",
			comment:        "// in:header X-API-Key",
			expectedSource: "header",
			expectedName:   "X-API-Key",
		},
		{
			name:           "header with single quotes",
			comment:        "// in:header 'User-Agent'",
			expectedSource: "header",
			expectedName:   "User-Agent",
		},
		{
			name:           "header with single quotes and spaces",
			comment:        "// in:header 'X-Custom Header'",
			expectedSource: "header",
			expectedName:   "X-Custom Header",
		},
		{
			name:           "header with single quotes and dashes",
			comment:        "// in:header 'X-Request-ID'",
			expectedSource: "header",
			expectedName:   "X-Request-ID",
		},
		{
			name:           "body",
			comment:        "// in:body",
			expectedSource: "body",
			expectedName:   "",
		},
		{
			name:           "body with space",
			comment:        "// in: body",
			expectedSource: "body",
			expectedName:   "",
		},
		{
			name:           "cookie with name",
			comment:        "// in:cookie sessionId",
			expectedSource: "cookie",
			expectedName:   "sessionId",
		},
		{
			name:           "cookie with quoted name",
			comment:        "// in:cookie 'session-id'",
			expectedSource: "cookie",
			expectedName:   "session-id",
		},
		{
			name:           "no in comment",
			comment:        "// some other comment",
			expectedSource: "",
			expectedName:   "",
		},
		{
			name:           "empty comment",
			comment:        "//",
			expectedSource: "",
			expectedName:   "",
		},
		{
			name:           "block comment",
			comment:        "/* in:query */",
			expectedSource: "query",
			expectedName:   "",
		},
		{
			name:           "block comment with name",
			comment:        "/* in:path userId */",
			expectedSource: "path",
			expectedName:   "userId",
		},
		{
			name:           "block comment with quoted name",
			comment:        "/* in:header 'Content-Type' */",
			expectedSource: "header",
			expectedName:   "Content-Type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, name := extractInComment(tt.comment)

			if source != tt.expectedSource {
				t.Errorf("expected source %q, got %q", tt.expectedSource, source)
			}

			if name != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, name)
			}
		})
	}
}
