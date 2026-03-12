package scanner

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Scanner Configuration Tests ====================

func TestNew(t *testing.T) {
	s := New()

	assert.NotNil(t, s)
	assert.Equal(t, "./...", s.config.Pattern)
	assert.Equal(t, ".", s.config.Dir)
	assert.NotNil(t, s.Enums)
	assert.NotNil(t, s.Structs)
	assert.NotNil(t, s.Routes)
}

func TestNewWithOptions(t *testing.T) {
	s := New(
		WithPattern("./api/..."),
		WithDir("/test/dir"),
		WithIgnorePaths("vendor", "testdata"),
	)

	assert.Equal(t, "./api/...", s.config.Pattern)
	assert.Equal(t, "/test/dir", s.config.Dir)
	assert.Contains(t, s.config.IgnorePaths, "vendor")
	assert.Contains(t, s.config.IgnorePaths, "testdata")
}

// ==================== Utility Function Tests ====================

func TestShouldIgnorePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		ignorePaths []string
		expected    bool
	}{
		{
			name:        "no ignore paths",
			path:        "/api/handlers.go",
			ignorePaths: []string{},
			expected:    false,
		},
		{
			name:        "path matches vendor",
			path:        "/project/vendor/lib/file.go",
			ignorePaths: []string{"vendor"},
			expected:    true,
		},
		{
			name:        "path doesn't match",
			path:        "/api/handlers.go",
			ignorePaths: []string{"vendor", "testdata"},
			expected:    false,
		},
		{
			name:        "path matches testdata",
			path:        "/project/testdata/test.go",
			ignorePaths: []string{"testdata"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIgnorePath(tt.path, tt.ignorePaths)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCompiledRegex(t *testing.T) {
	pattern := `swagger:\w+`

	// First call - should compile and cache
	re1 := getCompiledRegex(pattern)
	assert.NotNil(t, re1)

	// Second call - should return cached
	re2 := getCompiledRegex(pattern)
	assert.Equal(t, re1, re2)
}

// Helper to create AST CommentGroup from text
func createCommentGroup(comments ...string) *ast.CommentGroup {
	if len(comments) == 0 {
		return nil
	}
	list := make([]*ast.Comment, len(comments))
	for i, c := range comments {
		list[i] = &ast.Comment{Text: c}
	}
	return &ast.CommentGroup{List: list}
}

func TestHasDirective(t *testing.T) {
	tests := []struct {
		name      string
		comments  []string
		directive string
		expected  bool
	}{
		{
			name:      "nil doc",
			comments:  nil,
			directive: "swagger:model",
			expected:  false,
		},
		{
			name:      "has directive",
			comments:  []string{"// swagger:model User"},
			directive: "swagger:model",
			expected:  true,
		},
		{
			name:      "no directive",
			comments:  []string{"// This is a comment"},
			directive: "swagger:model",
			expected:  false,
		},
		{
			name:      "directive in block comment",
			comments:  []string{"/* swagger:meta */"},
			directive: "swagger:meta",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := createCommentGroup(tt.comments...)
			result := hasDirective(doc, tt.directive)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDirectiveValue(t *testing.T) {
	tests := []struct {
		name      string
		comments  []string
		directive string
		expected  string
	}{
		{
			name:      "nil doc",
			comments:  nil,
			directive: "swagger:model",
			expected:  "",
		},
		{
			name:      "extract model name",
			comments:  []string{"// swagger:model User"},
			directive: "swagger:model",
			expected:  "User",
		},
		{
			name:      "no value",
			comments:  []string{"// swagger:model"},
			directive: "swagger:model",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := createCommentGroup(tt.comments...)
			result := extractDirectiveValue(doc, tt.directive)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimComments(t *testing.T) {
	tests := []struct {
		name     string
		comments []string
		expected []string
	}{
		{
			name:     "nil doc",
			comments: nil,
			expected: nil,
		},
		{
			name:     "single line comment",
			comments: []string{"// Hello World"},
			expected: []string{"Hello World"},
		},
		{
			name:     "block comment",
			comments: []string{"/* Multi line */"},
			expected: []string{"Multi line"},
		},
		{
			name:     "multiple comments",
			comments: []string{"// Line 1", "// Line 2"},
			expected: []string{"Line 1", "Line 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := createCommentGroup(tt.comments...)
			result := trimComments(doc)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name       string
		comments   []string
		directives []string
		expected   string
	}{
		{
			name:       "nil doc",
			comments:   nil,
			directives: []string{},
			expected:   "",
		},
		{
			name:       "simple description",
			comments:   []string{"// This is a description"},
			directives: []string{},
			expected:   "This is a description",
		},
		{
			name:       "excludes directives",
			comments:   []string{"// Description", "// swagger:model User"},
			directives: []string{"swagger:"},
			expected:   "Description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := createCommentGroup(tt.comments...)
			result := extractDescription(doc, tt.directives)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSpecs(t *testing.T) {
	tests := []struct {
		name     string
		comments []string
		expected []string
	}{
		{
			name:     "nil doc",
			comments: nil,
			expected: nil,
		},
		{
			name:     "single spec",
			comments: []string{"// spec: admin"},
			expected: []string{"admin"},
		},
		{
			name:     "multiple specs",
			comments: []string{"// spec: admin public"},
			expected: []string{"admin", "public"},
		},
		{
			name:     "no spec directive",
			comments: []string{"// swagger:model User"},
			expected: nil,
		},
		{
			name:     "empty spec value",
			comments: []string{"// spec:"},
			expected: nil,
		},
		{
			name:     "case insensitive",
			comments: []string{"// Spec: Admin"},
			expected: []string{"admin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := createCommentGroup(tt.comments...)
			result := extractSpecs(doc)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== Integration Tests ====================

func createTestProject(t *testing.T, files map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0750))
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	}

	// Create go.mod
	goMod := "module testproject\n\ngo 1.21\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0600))

	return tmpDir
}

func TestScanSimpleModel(t *testing.T) {
	files := map[string]string{
		"models/user.go": "package models\n\n// swagger:model User\n// This type represents a user in the system\ntype User struct {\n\t// Unique identifier\n\tID string " + "`json:\"id\"`" + `
	// The user name
	Name string ` + "`json:\"name\"`" + `
}
`,
	}

	tmpDir := createTestProject(t, files)
	s := New(WithDir(tmpDir), WithPattern("./..."))

	err := s.Scan()

	require.NoError(t, err)
	assert.Contains(t, s.Structs, "User")
	user := s.Structs["User"]
	assert.True(t, user.IsModel)
}

func TestScanSimpleRoute(t *testing.T) {
	files := map[string]string{
		"handlers/users.go": `package handlers

// swagger:route GET /users users listUsers
// List all users
// Returns a list of users
// responses:
//   200: []User
func ListUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)
	s := New(WithDir(tmpDir), WithPattern("./..."))

	err := s.Scan()

	require.NoError(t, err)
	assert.Contains(t, s.Routes, "listUsers")
	route := s.Routes["listUsers"]
	assert.Equal(t, "GET", route.Method)
	assert.Equal(t, "/users", route.Path)
}

func TestScanMeta(t *testing.T) {
	files := map[string]string{
		"doc.go": `// swagger:meta
//
// Title: My API
// Version: 1.0.0
// Description: This is my API
//
// Contact:
//   name: API Support
//   email: support@example.com
//
// License:
//   name: MIT
//
package main
`,
	}

	tmpDir := createTestProject(t, files)
	s := New(WithDir(tmpDir), WithPattern("./..."))

	err := s.Scan()

	require.NoError(t, err)
	require.NotNil(t, s.Meta)
	assert.Equal(t, "My API", s.Meta.Title)
	assert.Equal(t, "1.0.0", s.Meta.Version)
}

func TestScanMultiSpec(t *testing.T) {
	files := map[string]string{
		"handlers/admin.go": `package handlers

// swagger:route GET /admin/users admin listAdminUsers
// spec: admin
// List admin users
func ListAdminUsers() {}
`,
		"handlers/public.go": `package handlers

// swagger:route GET /users users listPublicUsers
// spec: public
// List public users
func ListPublicUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)
	s := New(WithDir(tmpDir), WithPattern("./..."))

	err := s.Scan()

	require.NoError(t, err)
	require.Contains(t, s.Routes, "listAdminUsers")
	require.Contains(t, s.Routes, "listPublicUsers")

	adminRoute := s.Routes["listAdminUsers"]
	assert.Contains(t, adminRoute.Specs, "admin")

	publicRoute := s.Routes["listPublicUsers"]
	assert.Contains(t, publicRoute.Specs, "public")
}

func TestScanEnum(t *testing.T) {
	files := map[string]string{
		"models/status.go": `package models

// swagger:enum Status
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)
`,
	}

	tmpDir := createTestProject(t, files)
	s := New(WithDir(tmpDir), WithPattern("./..."))

	err := s.Scan()

	require.NoError(t, err)
	assert.Contains(t, s.Enums, "Status")
	enum := s.Enums["Status"]
	assert.Equal(t, "string", enum.BaseType)
	assert.Contains(t, enum.Values, "StatusActive")
	assert.Contains(t, enum.Values, "StatusInactive")
}

func TestScanParameters(t *testing.T) {
	files := map[string]string{
		"models/params.go": `package models

// swagger:parameters listUsers
type ListUsersParams struct {
	// The page number
	// in: query
	Page int ` + "`json:\"page\"`" + `
	// The page size
	// in: query
	Limit int ` + "`json:\"limit\"`" + `
}
`,
	}

	tmpDir := createTestProject(t, files)
	s := New(WithDir(tmpDir), WithPattern("./..."))

	err := s.Scan()

	require.NoError(t, err)
	assert.Contains(t, s.Structs, "listUsers")
	params := s.Structs["listUsers"]
	assert.True(t, params.IsParameter)
}

func TestScanIgnorePaths(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api
// swagger:route GET /users users listUsers
func ListUsers() {}
`,
		"vendor/lib/file.go": `package lib
// swagger:route GET /other other listOther
func ListOther() {}
`,
	}

	tmpDir := createTestProject(t, files)
	s := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithIgnorePaths("vendor"),
	)

	err := s.Scan()

	require.NoError(t, err)
	assert.Contains(t, s.Routes, "listUsers")
	// vendor routes should be ignored
}
