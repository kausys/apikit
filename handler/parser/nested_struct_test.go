package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseNestedStructWithFormComment(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.go")

	content := `package test

import (
	"context"
	"mime/multipart"
)

type UpdateAvatarPayload struct {
	// in:path
	UserID string ` + "`" + `json:"userId" validate:"required,uuid"` + "`" + `

	// in:form
	Body struct {
		// Avatar URL
		// in:form
		Avatar *multipart.FileHeader ` + "`" + `json:"avatar" validate:"required"` + "`" + `
	}
}

type UpdateAvatarResponse struct {
	Success bool ` + "`" + `json:"success"` + "`" + `
}

// apikit:handler
func UpdateAvatar(ctx context.Context, req UpdateAvatarPayload) (UpdateAvatarResponse, error) {
	return UpdateAvatarResponse{Success: true}, nil
}
`

	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	p := New()
	result, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Check UpdateAvatarPayload struct
	payload, ok := result.Structs["UpdateAvatarPayload"]
	if !ok {
		t.Fatal("expected UpdateAvatarPayload struct")
	}

	t.Logf("UpdateAvatarPayload has %d fields", len(payload.Fields))
	for i, field := range payload.Fields {
		t.Logf("Field %d: Name=%s, Type=%s, InComment=%s, IsFile=%v, NestedStruct=%v",
			i, field.Name, field.Type, field.InComment, field.IsFile, field.NestedStruct != nil)

		if field.NestedStruct != nil {
			t.Logf("  NestedStruct has %d fields:", len(field.NestedStruct.Fields))
			for j, nf := range field.NestedStruct.Fields {
				t.Logf("    Field %d: Name=%s, Type=%s, InComment=%s, IsFile=%v",
					j, nf.Name, nf.Type, nf.InComment, nf.IsFile)
			}
		}
	}

	// Find Body field
	var bodyField *Field
	for i := range payload.Fields {
		if payload.Fields[i].Name == "Body" {
			bodyField = &payload.Fields[i]
			break
		}
	}

	if bodyField == nil {
		t.Fatal("expected Body field")
	}

	if bodyField.InComment != "form" {
		t.Errorf("expected Body field to have InComment='form', got %q", bodyField.InComment)
	}

	if bodyField.NestedStruct == nil {
		t.Fatal("expected Body field to have NestedStruct")
	}

	if len(bodyField.NestedStruct.Fields) != 1 {
		t.Fatalf("expected Body.NestedStruct to have 1 field, got %d", len(bodyField.NestedStruct.Fields))
	}

	avatarField := bodyField.NestedStruct.Fields[0]
	if avatarField.Name != "Avatar" {
		t.Errorf("expected Avatar field, got %s", avatarField.Name)
	}

	if !avatarField.IsFile {
		t.Error("expected Avatar field to be marked as IsFile")
	}

	if avatarField.InComment != "form" {
		t.Errorf("expected Avatar field to have InComment='form', got %q", avatarField.InComment)
	}
}
