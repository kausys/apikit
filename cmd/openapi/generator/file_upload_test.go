package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileUploadSingleFile tests single file upload with multipart/form-data
func TestFileUploadSingleFile(t *testing.T) {
	files := map[string]string{
		"api/upload.go": `package api

// swagger:parameters uploadFile
type UploadFileParams struct {
	// in:body
	Body struct {
		// The file to upload
		// format: binary
		File string ` + "`json:\"file\"`" + `
	}
}

// swagger:route POST /upload files uploadFile
// Consumes:
// - multipart/form-data
//
// Upload a file
func UploadFile() {}
`,
	}

	tmpDir := createTestProject(t, files)
	g := New(WithDir(tmpDir), WithPattern("./..."), WithCache(false))

	spec, err := g.Generate()

	require.NoError(t, err)
	require.NotNil(t, spec)
	require.NotNil(t, spec.Paths)

	// Check the upload endpoint
	pathItem := spec.Paths.PathItems["/upload"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Post)

	// Check request body
	reqBody := pathItem.Post.RequestBody
	require.NotNil(t, reqBody)

	// Should have multipart/form-data content type
	multipartContent, ok := reqBody.Content["multipart/form-data"]
	require.True(t, ok, "Should have multipart/form-data content type")
	require.NotNil(t, multipartContent.Schema)

	// Schema should be type: object
	assert.Equal(t, "object", multipartContent.Schema.Type.Value())

	// Should have 'file' property
	fileSchema, ok := multipartContent.Schema.Properties["file"]
	require.True(t, ok, "Should have 'file' property")
	assert.Equal(t, "string", fileSchema.Type.Value())
	assert.Equal(t, "application/octet-stream", fileSchema.ContentMediaType)
	assert.Empty(t, fileSchema.Format)

	// Should have encoding for the file field
	require.NotNil(t, multipartContent.Encoding)
	fileEncoding, ok := multipartContent.Encoding["file"]
	require.True(t, ok, "Should have encoding for 'file' field")
	assert.Equal(t, "application/octet-stream", fileEncoding.ContentType)
}

// TestFileUploadMultipleFiles tests multiple file uploads
func TestFileUploadMultipleFiles(t *testing.T) {
	files := map[string]string{
		"api/upload.go": `package api

// swagger:parameters uploadMultipleFiles
type UploadMultipleFilesParams struct {
	// in:body
	Body struct {
		// Profile picture
		// format: binary
		Avatar string ` + "`json:\"avatar\"`" + `
		// Document file
		// format: binary
		Document string ` + "`json:\"document\"`" + `
	}
}

// swagger:route POST /upload/multiple files uploadMultipleFiles
// Consumes:
// - multipart/form-data
//
// Upload multiple files
func UploadMultipleFiles() {}
`,
	}

	tmpDir := createTestProject(t, files)
	g := New(WithDir(tmpDir), WithPattern("./..."), WithCache(false))

	spec, err := g.Generate()

	require.NoError(t, err)
	require.NotNil(t, spec)

	pathItem := spec.Paths.PathItems["/upload/multiple"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Post)

	reqBody := pathItem.Post.RequestBody
	require.NotNil(t, reqBody)

	multipartContent := reqBody.Content["multipart/form-data"]
	require.NotNil(t, multipartContent)

	// Should have both file properties
	assert.Contains(t, multipartContent.Schema.Properties, "avatar")
	assert.Contains(t, multipartContent.Schema.Properties, "document")

	// Both should use contentMediaType instead of format: binary
	assert.Equal(t, "application/octet-stream", multipartContent.Schema.Properties["avatar"].ContentMediaType)
	assert.Empty(t, multipartContent.Schema.Properties["avatar"].Format)
	assert.Equal(t, "application/octet-stream", multipartContent.Schema.Properties["document"].ContentMediaType)
	assert.Empty(t, multipartContent.Schema.Properties["document"].Format)

	// Should have encoding for both fields
	assert.Contains(t, multipartContent.Encoding, "avatar")
	assert.Contains(t, multipartContent.Encoding, "document")
}

// TestFileUploadMixedFormData tests file upload with additional form fields
func TestFileUploadMixedFormData(t *testing.T) {
	files := map[string]string{
		"api/upload.go": `package api

// swagger:parameters uploadWithMetadata
type UploadWithMetadataParams struct {
	// in:body
	Body struct {
		// The file to upload
		// format: binary
		File string ` + "`json:\"file\"`" + `
		// File description
		Description string ` + "`json:\"description\"`" + `
		// File tags
		Tags []string ` + "`json:\"tags\"`" + `
	}
}

// swagger:route POST /upload/metadata files uploadWithMetadata
// Consumes:
// - multipart/form-data
//
// Upload file with metadata
func UploadWithMetadata() {}
`,
	}

	tmpDir := createTestProject(t, files)
	g := New(WithDir(tmpDir), WithPattern("./..."), WithCache(false))

	spec, err := g.Generate()

	require.NoError(t, err)
	require.NotNil(t, spec)

	pathItem := spec.Paths.PathItems["/upload/metadata"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Post)

	reqBody := pathItem.Post.RequestBody
	require.NotNil(t, reqBody)

	multipartContent := reqBody.Content["multipart/form-data"]
	require.NotNil(t, multipartContent)

	// Should have all three properties
	assert.Contains(t, multipartContent.Schema.Properties, "file")
	assert.Contains(t, multipartContent.Schema.Properties, "description")
	assert.Contains(t, multipartContent.Schema.Properties, "tags")

	// Only 'file' should have contentMediaType (not format: binary)
	assert.Equal(t, "application/octet-stream", multipartContent.Schema.Properties["file"].ContentMediaType)
	assert.Empty(t, multipartContent.Schema.Properties["file"].Format)
	assert.Equal(t, "string", multipartContent.Schema.Properties["description"].Type.Value())
	assert.Equal(t, "array", multipartContent.Schema.Properties["tags"].Type.Value())

	// Only 'file' should have encoding
	require.NotNil(t, multipartContent.Encoding)
	assert.Contains(t, multipartContent.Encoding, "file")
	assert.NotContains(t, multipartContent.Encoding, "description")
	assert.NotContains(t, multipartContent.Encoding, "tags")
}

// TestFileUploadWithJSONFallback tests that JSON is used when no consumes specified
func TestFileUploadWithJSONFallback(t *testing.T) {
	files := map[string]string{
		"api/upload.go": `package api

// swagger:parameters createUser
type CreateUserParams struct {
	// in:body
	Body struct {
		Name  string ` + "`json:\"name\"`" + `
		Email string ` + "`json:\"email\"`" + `
	}
}

// swagger:route POST /users users createUser
//
// Create a user
func CreateUser() {}
`,
	}

	tmpDir := createTestProject(t, files)
	g := New(WithDir(tmpDir), WithPattern("./..."), WithCache(false))

	spec, err := g.Generate()

	require.NoError(t, err)
	require.NotNil(t, spec)

	pathItem := spec.Paths.PathItems["/users"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Post)

	reqBody := pathItem.Post.RequestBody
	require.NotNil(t, reqBody)

	// Should default to application/json
	jsonContent, ok := reqBody.Content["application/json"]
	require.True(t, ok, "Should have application/json content type by default")
	require.NotNil(t, jsonContent.Schema)

	// Should NOT have multipart/form-data
	_, hasMultipart := reqBody.Content["multipart/form-data"]
	assert.False(t, hasMultipart, "Should not have multipart/form-data without Consumes directive")
}

// TestFileUploadMultipleContentTypes tests multiple content types in Consumes
func TestFileUploadMultipleContentTypes(t *testing.T) {
	files := map[string]string{
		"api/upload.go": `package api

// swagger:parameters uploadFlexible
type UploadFlexibleParams struct {
	// in:body
	Body struct {
		// format: binary
		File string ` + "`json:\"file\"`" + `
	}
}

// swagger:route POST /upload/flexible files uploadFlexible
// Consumes:
// - multipart/form-data
// - application/json
//
// Upload with multiple content types
func UploadFlexible() {}
`,
	}

	tmpDir := createTestProject(t, files)
	g := New(WithDir(tmpDir), WithPattern("./..."), WithCache(false))

	spec, err := g.Generate()

	require.NoError(t, err)
	require.NotNil(t, spec)

	pathItem := spec.Paths.PathItems["/upload/flexible"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Post)

	reqBody := pathItem.Post.RequestBody
	require.NotNil(t, reqBody)

	// Should have both content types
	assert.Contains(t, reqBody.Content, "multipart/form-data")
	assert.Contains(t, reqBody.Content, "application/json")

	// Multipart should have encoding
	multipartContent := reqBody.Content["multipart/form-data"]
	require.NotNil(t, multipartContent.Encoding)

	// JSON should not have encoding
	jsonContent := reqBody.Content["application/json"]
	assert.Nil(t, jsonContent.Encoding)
}

// TestFileUploadWithInFormDirective tests that in:form works like in:body
func TestFileUploadWithInFormDirective(t *testing.T) {
	files := map[string]string{
		"api/upload.go": `package api

// swagger:parameters uploadDocument
type UploadDocumentParams struct {
	// The account ID
	// in: query
	// required: true
	AccountID string ` + "`json:\"accountId\"`" + `

	// in:form
	Body struct {
		// The file to upload
		// format: binary
		File string ` + "`json:\"file\"`" + `
		// Document description
		Description string ` + "`json:\"description\"`" + `
	}
}

// swagger:route POST /upload/document files uploadDocument
// Consumes:
// - multipart/form-data
//
// Upload document with in:form directive
func UploadDocument() {}
`,
	}

	tmpDir := createTestProject(t, files)
	g := New(WithDir(tmpDir), WithPattern("./..."), WithCache(false))

	spec, err := g.Generate()

	require.NoError(t, err)
	require.NotNil(t, spec)

	pathItem := spec.Paths.PathItems["/upload/document"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Post)

	// Should have query parameter
	require.Len(t, pathItem.Post.Parameters, 1)
	assert.Equal(t, "accountId", pathItem.Post.Parameters[0].Name)
	assert.Equal(t, "query", pathItem.Post.Parameters[0].In)

	// Should have request body with multipart/form-data
	reqBody := pathItem.Post.RequestBody
	require.NotNil(t, reqBody)

	multipartContent := reqBody.Content["multipart/form-data"]
	require.NotNil(t, multipartContent)

	// Should have both file and description properties
	assert.Contains(t, multipartContent.Schema.Properties, "file")
	assert.Contains(t, multipartContent.Schema.Properties, "description")

	// File should have contentMediaType instead of format: binary
	assert.Equal(t, "application/octet-stream", multipartContent.Schema.Properties["file"].ContentMediaType)
	assert.Empty(t, multipartContent.Schema.Properties["file"].Format)

	// Should have encoding for file
	require.NotNil(t, multipartContent.Encoding)
	assert.Contains(t, multipartContent.Encoding, "file")
	assert.Equal(t, "application/octet-stream", multipartContent.Encoding["file"].ContentType)
}
