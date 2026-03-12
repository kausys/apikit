package tags

import (
	"go/ast"
	"testing"

	"github.com/kausys/apikit/openapi/parser"
	"github.com/kausys/apikit/openapi/spec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Helper Functions ====================

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

// ==================== SingleLineParser Tests ====================

func TestSingleLineParserMatches(t *testing.T) {
	p := NewSingleLineParser("test", "summary:", []parser.Context{parser.ContextRoute}, nil)

	assert.True(t, p.Matches("Summary: test", parser.ContextRoute))
	assert.True(t, p.Matches("summary: test", parser.ContextRoute))
	assert.False(t, p.Matches("description: test", parser.ContextRoute))
	assert.False(t, p.Matches("summary: test", parser.ContextModel)) // wrong context
}

func TestSingleLineParserParse(t *testing.T) {
	p := NewSingleLineParser("test", "summary:", []parser.Context{parser.ContextRoute}, nil)

	tests := []struct {
		name     string
		comments []string
		expected string
	}{
		{
			name:     "simple value",
			comments: []string{"// summary: This is a summary"},
			expected: "This is a summary",
		},
		{
			name:     "with extra spaces",
			comments: []string{"//   summary:   Trimmed value  "},
			expected: "Trimmed value",
		},
		{
			name:     "no match",
			comments: []string{"// description: Not a summary"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cg := createCommentGroup(tt.comments...)
			result, err := p.Parse(cg, parser.ContextRoute)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSingleLineParserApply(t *testing.T) {
	var capturedValue any
	setter := func(target any, value any) error {
		capturedValue = value
		return nil
	}
	p := NewSingleLineParser("test", "summary:", []parser.Context{parser.ContextRoute}, parser.SetterMap{
		parser.ContextRoute: setter,
	})

	err := p.Apply(nil, "test value", parser.ContextRoute)

	require.NoError(t, err)
	assert.Equal(t, "test value", capturedValue)
}

func TestSingleLineParserApplyEmptyValue(t *testing.T) {
	p := NewSingleLineParser("test", "summary:", []parser.Context{parser.ContextRoute}, nil)

	err := p.Apply(nil, "", parser.ContextRoute)

	assert.NoError(t, err) // Should not call setter for empty value
}

// ==================== MultiLineParser Tests ====================

func TestMultiLineParserMatches(t *testing.T) {
	p := NewMultiLineParser("test", "description:", []parser.Context{parser.ContextRoute}, nil)

	assert.True(t, p.Matches("Description: test", parser.ContextRoute))
	assert.True(t, p.Matches("description: test", parser.ContextRoute))
	assert.False(t, p.Matches("summary: test", parser.ContextRoute))
	assert.False(t, p.Matches("description: test", parser.ContextModel)) // wrong context
}

func TestMultiLineParserParse(t *testing.T) {
	p := NewMultiLineParser("test", "description:", []parser.Context{parser.ContextRoute}, nil)

	tests := []struct {
		name     string
		comments []string
		expected string
	}{
		{
			name:     "single line",
			comments: []string{"// description: Single line"},
			expected: "Single line",
		},
		{
			name: "multi line",
			comments: []string{
				"// description:",
				"// Line 1",
				"// Line 2",
			},
			expected: "Line 1\nLine 2",
		},
		{
			name: "stops at next directive",
			comments: []string{
				"// description:",
				"// Line 1",
				"// summary: Next directive",
			},
			expected: "Line 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cg := createCommentGroup(tt.comments...)
			result, err := p.Parse(cg, parser.ContextRoute)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== ListParser Tests ====================

func TestListParserMatches(t *testing.T) {
	p := NewListParser("test", "tags:", ",", []parser.Context{parser.ContextRoute}, nil)

	assert.True(t, p.Matches("Tags: a, b", parser.ContextRoute))
	assert.False(t, p.Matches("summary: test", parser.ContextRoute))
}

func TestListParserParse(t *testing.T) {
	p := NewListParser("test", "tags:", ",", []parser.Context{parser.ContextRoute}, nil)

	tests := []struct {
		name     string
		comments []string
		expected []string
	}{
		{
			name:     "comma separated",
			comments: []string{"// tags: a, b, c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with spaces",
			comments: []string{"// tags:  item1 ,  item2 "},
			expected: []string{"item1", "item2"},
		},
		{
			name:     "no match",
			comments: []string{"// summary: test"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cg := createCommentGroup(tt.comments...)
			result, err := p.Parse(cg, parser.ContextRoute)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestListParserApply(t *testing.T) {
	var capturedValue any
	setter := func(target any, value any) error {
		capturedValue = value
		return nil
	}
	p := NewListParser("test", "tags:", ",", []parser.Context{parser.ContextRoute}, parser.SetterMap{
		parser.ContextRoute: setter,
	})

	err := p.Apply(nil, []string{"a", "b"}, parser.ContextRoute)

	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, capturedValue)
}

func TestListParserApplyEmptyList(t *testing.T) {
	p := NewListParser("test", "tags:", ",", []parser.Context{parser.ContextRoute}, nil)

	err := p.Apply(nil, []string{}, parser.ContextRoute)

	assert.NoError(t, err) // Should not call setter for empty list
}

// ==================== Meta Parsers Tests ====================

func TestTitleParser(t *testing.T) {
	p := NewTitleParser()
	info := &spec.Info{}

	cg := createCommentGroup("// Title: My API")
	value, err := p.Parse(cg, parser.ContextMeta)
	require.NoError(t, err)

	err = p.Apply(info, value, parser.ContextMeta)
	require.NoError(t, err)
	assert.Equal(t, "My API", info.Title)
}

func TestVersionParser(t *testing.T) {
	p := NewVersionParser()
	info := &spec.Info{}

	cg := createCommentGroup("// Version: 1.0.0")
	value, err := p.Parse(cg, parser.ContextMeta)
	require.NoError(t, err)

	err = p.Apply(info, value, parser.ContextMeta)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", info.Version)
}

func TestContactParser(t *testing.T) {
	p := NewContactParser()
	info := &spec.Info{}

	cg := createCommentGroup("// Contact: support@example.com")
	value, err := p.Parse(cg, parser.ContextMeta)
	require.NoError(t, err)

	err = p.Apply(info, value, parser.ContextMeta)
	require.NoError(t, err)
	require.NotNil(t, info.Contact)
	assert.Equal(t, "support@example.com", info.Contact.Email)
}

func TestLicenseParser(t *testing.T) {
	p := NewLicenseParser()
	info := &spec.Info{}

	cg := createCommentGroup("// License: MIT")
	value, err := p.Parse(cg, parser.ContextMeta)
	require.NoError(t, err)

	err = p.Apply(info, value, parser.ContextMeta)
	require.NoError(t, err)
	require.NotNil(t, info.License)
	assert.Equal(t, "MIT", info.License.Name)
}

func TestBasepathParser(t *testing.T) {
	p := NewBasepathParser()
	openAPI := &spec.OpenAPI{}

	cg := createCommentGroup("// BasePath: /api/v1")
	value, err := p.Parse(cg, parser.ContextMeta)
	require.NoError(t, err)

	err = p.Apply(openAPI, value, parser.ContextMeta)
	require.NoError(t, err)
	require.Len(t, openAPI.Servers, 1)
	assert.Equal(t, "/api/v1", openAPI.Servers[0].URL)
}

func TestHostParser(t *testing.T) {
	p := NewHostParser()
	openAPI := &spec.OpenAPI{}

	cg := createCommentGroup("// Host: api.example.com")
	value, err := p.Parse(cg, parser.ContextMeta)
	require.NoError(t, err)

	err = p.Apply(openAPI, value, parser.ContextMeta)
	require.NoError(t, err)
	require.Len(t, openAPI.Servers, 1)
	assert.Equal(t, "https://api.example.com", openAPI.Servers[0].URL)
}

// ==================== Route Parsers Tests ====================

func TestSummaryParser(t *testing.T) {
	p := NewSummaryParser()
	op := &spec.Operation{}

	cg := createCommentGroup("// Summary: Get all users")
	value, err := p.Parse(cg, parser.ContextRoute)
	require.NoError(t, err)

	err = p.Apply(op, value, parser.ContextRoute)
	require.NoError(t, err)
	assert.Equal(t, "Get all users", op.Summary)
}

func TestDescriptionParser(t *testing.T) {
	p := NewDescriptionParser()
	op := &spec.Operation{}

	cg := createCommentGroup("// Description: This is a description")
	value, err := p.Parse(cg, parser.ContextRoute)
	require.NoError(t, err)

	err = p.Apply(op, value, parser.ContextRoute)
	require.NoError(t, err)
	assert.Equal(t, "This is a description", op.Description)
}

func TestTagsParser(t *testing.T) {
	p := NewTagsParser()
	op := &spec.Operation{}

	cg := createCommentGroup("// Tags: users, admin")
	value, err := p.Parse(cg, parser.ContextRoute)
	require.NoError(t, err)

	err = p.Apply(op, value, parser.ContextRoute)
	require.NoError(t, err)
	assert.Equal(t, []string{"users", "admin"}, op.Tags)
}

func TestDeprecatedParser(t *testing.T) {
	p := NewDeprecatedParser()
	op := &spec.Operation{}

	cg := createCommentGroup("// Deprecated: true")
	value, err := p.Parse(cg, parser.ContextRoute)
	require.NoError(t, err)

	err = p.Apply(op, value, parser.ContextRoute)
	require.NoError(t, err)
	assert.True(t, op.Deprecated)
}

func TestOperationIDParser(t *testing.T) {
	p := NewOperationIDParser()
	op := &spec.Operation{}

	cg := createCommentGroup("// OperationID: getUsers")
	value, err := p.Parse(cg, parser.ContextRoute)
	require.NoError(t, err)

	err = p.Apply(op, value, parser.ContextRoute)
	require.NoError(t, err)
	assert.Equal(t, "getUsers", op.OperationID)
}

func TestSecurityParser(t *testing.T) {
	p := NewSecurityParser()
	op := &spec.Operation{}

	cg := createCommentGroup("// Security: bearerAuth, apiKey")
	value, err := p.Parse(cg, parser.ContextRoute)
	require.NoError(t, err)

	err = p.Apply(op, value, parser.ContextRoute)
	require.NoError(t, err)
	require.Len(t, op.Security, 2)
}

// ==================== Model Parsers Tests ====================

func TestExampleParser(t *testing.T) {
	p := NewExampleParser()
	schema := &spec.Schema{}

	cg := createCommentGroup("// Example: John Doe")
	value, err := p.Parse(cg, parser.ContextModel)
	require.NoError(t, err)

	err = p.Apply(schema, value, parser.ContextModel)
	require.NoError(t, err)
	require.Len(t, schema.Examples, 1)
	assert.Equal(t, "John Doe", schema.Examples[0])
}

func TestRequiredParser(t *testing.T) {
	p := NewRequiredParser()
	schema := &spec.Schema{}

	cg := createCommentGroup("// Required: name, email")
	value, err := p.Parse(cg, parser.ContextModel)
	require.NoError(t, err)

	err = p.Apply(schema, value, parser.ContextModel)
	require.NoError(t, err)
	assert.Equal(t, []string{"name", "email"}, schema.Required)
}

func TestMinLengthParser(t *testing.T) {
	p := NewMinLengthParser()
	schema := &spec.Schema{}

	cg := createCommentGroup("// MinLength: 5")
	value, err := p.Parse(cg, parser.ContextField)
	require.NoError(t, err)

	err = p.Apply(schema, value, parser.ContextField)
	require.NoError(t, err)
	assert.Equal(t, uint64(5), schema.MinLength)
}

func TestMaxLengthParser(t *testing.T) {
	p := NewMaxLengthParser()
	schema := &spec.Schema{}

	cg := createCommentGroup("// MaxLength: 100")
	value, err := p.Parse(cg, parser.ContextField)
	require.NoError(t, err)

	err = p.Apply(schema, value, parser.ContextField)
	require.NoError(t, err)
	require.NotNil(t, schema.MaxLength)
	assert.Equal(t, uint64(100), *schema.MaxLength)
}

func TestPatternParser(t *testing.T) {
	p := NewPatternParser()
	schema := &spec.Schema{}

	cg := createCommentGroup("// Pattern: ^[a-z]+$")
	value, err := p.Parse(cg, parser.ContextField)
	require.NoError(t, err)

	err = p.Apply(schema, value, parser.ContextField)
	require.NoError(t, err)
	assert.Equal(t, "^[a-z]+$", schema.Pattern)
}

func TestFormatParser(t *testing.T) {
	p := NewFormatParser()
	schema := &spec.Schema{}

	cg := createCommentGroup("// Format: email")
	value, err := p.Parse(cg, parser.ContextField)
	require.NoError(t, err)

	err = p.Apply(schema, value, parser.ContextField)
	require.NoError(t, err)
	assert.Equal(t, "email", schema.Format)
}

func TestEnumParser(t *testing.T) {
	p := NewEnumParser()
	schema := &spec.Schema{}

	cg := createCommentGroup("// Enum: active, inactive, pending")
	value, err := p.Parse(cg, parser.ContextField)
	require.NoError(t, err)

	err = p.Apply(schema, value, parser.ContextField)
	require.NoError(t, err)
	assert.Len(t, schema.Enum, 3)
}

// ==================== Invalid Target Tests ====================

func TestTitleParserInvalidTarget(t *testing.T) {
	p := NewTitleParser()

	err := p.Apply("invalid", "value", parser.ContextMeta)

	require.Error(t, err)
	var invalidTarget *parser.ErrInvalidTarget
	assert.ErrorAs(t, err, &invalidTarget)
}

func TestSummaryParserInvalidTarget(t *testing.T) {
	p := NewSummaryParser()

	err := p.Apply("invalid", "value", parser.ContextRoute)

	require.Error(t, err)
	var invalidTarget *parser.ErrInvalidTarget
	assert.ErrorAs(t, err, &invalidTarget)
}

// ==================== typeOf Tests ====================

func TestTypeOf(t *testing.T) {
	assert.Equal(t, "nil", typeOf(nil))
	assert.Equal(t, "string", typeOf("test"))
	assert.Equal(t, "*spec.Info", typeOf(&spec.Info{}))
}
