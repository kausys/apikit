package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kausys/apikit/cmd/handler/parser"
)

func TestGenerate_WithFormFields(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.go")

	content := `package test

import (
	"context"
	"mime/multipart"
)

type UploadRequest struct {
	// in:form custom_title
	Title string

	// in:form
	Description string

	// in:form user_avatar
	Avatar *multipart.FileHeader
}

type UploadResponse struct {
	ID string ` + "`" + `json:"id"` + "`" + `
}

// apikit:handler
func Upload(ctx context.Context, req UploadRequest) (UploadResponse, error) {
	return UploadResponse{}, nil
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

	// Verify custom_title is used for Title field
	if !strings.Contains(generatedStr, `r.FormValue("custom_title")`) {
		t.Error("expected generated code to use 'custom_title' from comment")
	}

	// Verify description is used for Description field (camelCase conversion)
	if !strings.Contains(generatedStr, `r.FormValue("description")`) {
		t.Error("expected generated code to use 'description' (camelCase)")
	}

	// Verify user_avatar is used for Avatar field
	if !strings.Contains(generatedStr, `r.FormFile("user_avatar")`) {
		t.Error("expected generated code to use 'user_avatar' from comment for file field")
	}

	// Verify payload assignments
	if !strings.Contains(generatedStr, "payload.Title") {
		t.Error("expected generated code to assign to payload.Title")
	}

	if !strings.Contains(generatedStr, "payload.Description") {
		t.Error("expected generated code to assign to payload.Description")
	}

	if !strings.Contains(generatedStr, "payload.Avatar") {
		t.Error("expected generated code to assign to payload.Avatar")
	}

	t.Logf("Generated code:\n%s", generatedStr)
}

func TestGenerate_WithMixedFormTagsAndComments(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.go")

	content := `package test

import (
	"context"
	"mime/multipart"
)

type UploadRequest struct {
	// Tag should take priority over comment
	// in:form comment_name
	Title string ` + "`" + `form:"tag_name"` + "`" + `

	// Comment should be used when no tag
	// in:form custom_desc
	Description string

	// Tag should take priority for file fields too
	// in:form comment_avatar
	Avatar *multipart.FileHeader ` + "`" + `form:"tag_avatar"` + "`" + `
}

type UploadResponse struct {
	ID string ` + "`" + `json:"id"` + "`" + `
}

// apikit:handler
func Upload(ctx context.Context, req UploadRequest) (UploadResponse, error) {
	return UploadResponse{}, nil
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

	// Tag should take priority over comment
	if !strings.Contains(generatedStr, `r.FormValue("tag_name")`) {
		t.Error("expected tag 'tag_name' to take priority over comment 'comment_name'")
	}

	if strings.Contains(generatedStr, `r.FormValue("comment_name")`) {
		t.Error("comment name should not be used when tag is present")
	}

	// Comment should be used when no tag
	if !strings.Contains(generatedStr, `r.FormValue("custom_desc")`) {
		t.Error("expected comment 'custom_desc' to be used when no tag is present")
	}

	// Tag should take priority for file fields
	if !strings.Contains(generatedStr, `r.FormFile("tag_avatar")`) {
		t.Error("expected tag 'tag_avatar' to take priority over comment 'comment_avatar' for file field")
	}

	if strings.Contains(generatedStr, `r.FormFile("comment_avatar")`) {
		t.Error("comment name should not be used for file field when tag is present")
	}
}
