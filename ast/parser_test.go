package ast

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package test

import (
	"context"
	"net/http"
)

// User represents a user in the system
// swagger:model
type User struct {
	// ID is the user identifier
	// Example: 123
	ID int ` + "`json:\"id\"`" + `

	// Name is the user's full name
	// Example: John Doe
	Name string ` + "`json:\"name\"`" + `

	// Email is the user's email address
	Email string ` + "`json:\"email\"`" + `
}

// CreateUserRequest is the request for creating a user
// swagger:route POST /users user createUser
// Summary: Create a new user
type CreateUserRequest struct {
	Name  string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
}

// GetUserRequest is the request for getting a user
type GetUserRequest struct {
	ID int ` + "`json:\"id\"`" + `
}

// apikit:handler
func CreateUser(ctx context.Context, req CreateUserRequest) (User, error) {
	return User{}, nil
}

// apikit:handler
func (s *Service) GetUser(ctx context.Context, req GetUserRequest) (User, error) {
	return User{}, nil
}
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	parser := New()
	result, err := parser.Parse(testFile)

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check package
	if result.Package != "test" {
		t.Errorf("expected package 'test', got %q", result.Package)
	}

	// Check imports
	if len(result.Imports) != 2 {
		t.Errorf("expected 2 imports, got %d", len(result.Imports))
	}

	if result.Imports["context"] != "context" {
		t.Errorf("expected context import, got %v", result.Imports)
	}

	// Check structs
	if len(result.Structs) != 3 {
		t.Logf("Found structs: %v", result.Structs)
		for name := range result.Structs {
			t.Logf("  - %s", name)
		}
		t.Fatalf("expected 3 structs, got %d", len(result.Structs))
	}

	// Check User struct
	user, ok := result.Structs["User"]
	if !ok {
		t.Fatal("expected User struct")
	}

	if user.Name != "User" {
		t.Errorf("expected struct name 'User', got %q", user.Name)
	}

	if user.Doc == nil {
		t.Error("expected User struct to have doc comments")
	}

	if len(user.Fields) != 3 {
		t.Errorf("expected 3 fields in User, got %d", len(user.Fields))
	}

	// Check ID field
	idField := user.Fields[0]
	if idField.Name != "ID" {
		t.Errorf("expected field name 'ID', got %q", idField.Name)
	}

	if idField.Type != "int" {
		t.Errorf("expected field type 'int', got %q", idField.Type)
	}

	if idField.Tag != `json:"id"` {
		t.Errorf("expected tag 'json:\"id\"', got %q", idField.Tag)
	}

	if idField.Doc == nil {
		t.Error("expected ID field to have doc comments")
	}

	// Check functions
	if len(result.Functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(result.Functions))
	}

	// Check CreateUser function
	createUser := result.Functions[0]
	if createUser.Name != "CreateUser" {
		t.Errorf("expected function name 'CreateUser', got %q", createUser.Name)
	}

	if createUser.Receiver != "" {
		t.Errorf("expected no receiver, got %q", createUser.Receiver)
	}

	if createUser.Doc == nil {
		t.Error("expected CreateUser to have doc comments")
	}

	if len(createUser.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(createUser.Params))
	}

	if len(createUser.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(createUser.Results))
	}

	// Check GetUser method
	getUser := result.Functions[1]
	if getUser.Name != "GetUser" {
		t.Errorf("expected function name 'GetUser', got %q", getUser.Name)
	}

	if getUser.Receiver != "*Service" {
		t.Errorf("expected receiver '*Service', got %q", getUser.Receiver)
	}
}
