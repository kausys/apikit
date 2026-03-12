package parser

import (
	"os"
	"path/filepath"
	"testing"

	coreast "github.com/kausys/apikit/scanner/ast"
)

func TestExtractFromGeneric(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package test

import (
	"context"
	"net/http"
)

// User represents a user
// apikit:dto
type User struct {
	// ID is the user identifier
	ID int ` + "`json:\"id\"`" + `

	// Name is the user's name
	Name string ` + "`json:\"name\"`" + `
}

// CreateUserRequest is the request for creating a user
type CreateUserRequest struct {
	// in:body
	Name  string ` + "`json:\"name\"`" + `
	
	// in:query
	Source string ` + "`json:\"source\"`" + `
}

// apikit:handler
func CreateUser(ctx context.Context, req CreateUserRequest) (User, error) {
	return User{}, nil
}

// apikit:handler
func (s *Service) GetUser(ctx context.Context, req GetUserRequest, w http.ResponseWriter) (User, error) {
	return User{}, nil
}

type GetUserRequest struct {
	// in:path userId
	ID int ` + "`json:\"id\"`" + `
}
`

	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse with generic parser
	genericParser := coreast.New()
	genericResult, err := genericParser.Parse(testFile)
	if err != nil {
		t.Fatalf("generic parse failed: %v", err)
	}

	// Extract APIKit-specific information
	result, err := ExtractFromGeneric(genericResult)
	if err != nil {
		t.Fatalf("ExtractFromGeneric failed: %v", err)
	}

	// Check package
	if result.Source.Package != "test" {
		t.Errorf("expected package 'test', got %q", result.Source.Package)
	}

	// Check structs
	if len(result.Structs) != 3 {
		t.Fatalf("expected 3 structs, got %d", len(result.Structs))
	}

	// Check User struct (should be marked as DTO)
	user, ok := result.Structs["User"]
	if !ok {
		t.Fatal("expected User struct")
	}

	if !user.IsDTO {
		t.Error("expected User struct to be marked as DTO")
	}

	if len(user.Fields) != 2 {
		t.Errorf("expected 2 fields in User, got %d", len(user.Fields))
	}

	// Check CreateUserRequest struct
	createReq, ok := result.Structs["CreateUserRequest"]
	if !ok {
		t.Fatal("expected CreateUserRequest struct")
	}

	if createReq.IsDTO {
		t.Error("expected CreateUserRequest not to be marked as DTO")
	}

	// Check Name field (should have in:body)
	nameField := createReq.Fields[0]
	if nameField.InComment != "body" {
		t.Errorf("expected Name field to have in:body, got %q", nameField.InComment)
	}

	if !nameField.IsBody {
		t.Error("expected Name field to be marked as body")
	}

	// Check Source field (should have in:query)
	sourceField := createReq.Fields[1]
	if sourceField.InComment != "query" {
		t.Errorf("expected Source field to have in:query, got %q", sourceField.InComment)
	}

	// Check handlers
	if len(result.Handlers) != 2 {
		t.Fatalf("expected 2 handlers, got %d", len(result.Handlers))
	}

	// Check CreateUser handler
	createUser := result.Handlers[0]
	if createUser.Name != "CreateUser" {
		t.Errorf("expected handler name 'CreateUser', got %q", createUser.Name)
	}

	if createUser.Receiver != "" {
		t.Errorf("expected no receiver, got %q", createUser.Receiver)
	}

	if createUser.ParamType != "CreateUserRequest" {
		t.Errorf("expected param type 'CreateUserRequest', got %q", createUser.ParamType)
	}

	if createUser.ReturnType != "User" {
		t.Errorf("expected return type 'User', got %q", createUser.ReturnType)
	}

	if createUser.HasResponseWriter {
		t.Error("expected CreateUser not to have ResponseWriter")
	}

	// Check GetUser handler
	getUser := result.Handlers[1]
	if getUser.Name != "GetUser" {
		t.Errorf("expected handler name 'GetUser', got %q", getUser.Name)
	}

	if getUser.Receiver != "*Service" {
		t.Errorf("expected receiver '*Service', got %q", getUser.Receiver)
	}

	if !getUser.HasResponseWriter {
		t.Error("expected GetUser to have ResponseWriter")
	}
}
