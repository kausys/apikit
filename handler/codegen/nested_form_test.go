package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kausys/apikit/handler/parser"
)

func TestGenerate_WithNestedFormBody(t *testing.T) {
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

	p := parser.New()
	result, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	g, err := New()
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	generated, err := g.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	generatedStr := string(generated)

	// Verify ParseMultipartForm is called
	if !strings.Contains(generatedStr, "ParseMultipartForm") {
		t.Error("expected generated code to call ParseMultipartForm")
	}

	// Verify path parameter extraction (uses json tag value: json:"userId")
	if !strings.Contains(generatedStr, `r.PathValue("userId")`) {
		t.Error("expected generated code to extract userId from path")
	}

	if !strings.Contains(generatedStr, "payload.UserID") {
		t.Error("expected generated code to assign to payload.UserID")
	}

	// Verify form file extraction for nested Avatar field
	if !strings.Contains(generatedStr, `r.FormFile("avatar")`) {
		t.Error("expected generated code to use r.FormFile for avatar field")
	}

	if !strings.Contains(generatedStr, "payload.Body.Avatar") {
		t.Error("expected generated code to assign to payload.Body.Avatar")
	}

	// Verify validation is enabled
	if !strings.Contains(generatedStr, "validator.StructCtx") {
		t.Error("expected generated code to call validator.StructCtx")
	}

	// Verify imports
	if !strings.Contains(generatedStr, `"github.com/kausys/apikit/validator"`) {
		t.Error("expected validator import")
	}

	t.Logf("Generated code:\n%s", generatedStr)
}

func TestGenerate_WithMultipleNestedFormFields(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.go")

	content := `package test

import (
	"context"
	"mime/multipart"
)

type UpdateProfilePayload struct {
	// in:path
	UserID string ` + "`" + `json:"userId" validate:"required,uuid"` + "`" + `

	// in:form
	Body struct {
		// in:form
		Name string ` + "`" + `json:"name" validate:"required"` + "`" + `
		
		// in:form
		Bio string ` + "`" + `json:"bio"` + "`" + `
		
		// in:form
		Avatar *multipart.FileHeader ` + "`" + `json:"avatar"` + "`" + `
		
		// in:form
		CoverImage *multipart.FileHeader ` + "`" + `json:"cover_image"` + "`" + `
	}
}

type UpdateProfileResponse struct {
	Success bool ` + "`" + `json:"success"` + "`" + `
}

// apikit:handler
func UpdateProfile(ctx context.Context, req UpdateProfilePayload) (UpdateProfileResponse, error) {
	return UpdateProfileResponse{Success: true}, nil
}
`

	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	p := parser.New()
	result, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	g, err := New()
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	generated, err := g.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	generatedStr := string(generated)

	// Verify all form fields are extracted
	// Note: GetParameterName uses json tag when no form tag is present
	expectedFields := []string{
		`r.FormValue("name")`,
		`r.FormValue("bio")`,
		`r.FormFile("avatar")`,
		`r.FormFile("cover_image")`, // Uses json:"cover_image" tag value
	}

	for _, expected := range expectedFields {
		if !strings.Contains(generatedStr, expected) {
			t.Errorf("expected generated code to contain: %s", expected)
		}
	}

	// Verify all assignments to nested Body fields
	expectedAssignments := []string{
		"payload.Body.Name",
		"payload.Body.Bio",
		"payload.Body.Avatar",
		"payload.Body.CoverImage",
	}

	for _, expected := range expectedAssignments {
		if !strings.Contains(generatedStr, expected) {
			t.Errorf("expected generated code to assign to: %s", expected)
		}
	}

	t.Logf("Generated code:\n%s", generatedStr)
}
