package generator

import (
	"testing"

	"github.com/kausys/apikit/cmd/scanner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A `Responses:` line is free text after the status code, so the directive
// parser reads `- 412: when the account is not ready` as the type "when". Those
// have always been written as `type: string`; the resolve check has to leave them
// alone or it reports a comment that reads perfectly well.
func TestLooksLikeModelRef(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"PaginatedAPIKeyResponse", true},
		{"Transaction", true},
		{"Kyc_Response2", true},
		{"models.User", true},

		{"when", false},
		{"on", false},
		{"the", false},
		{"user", false},
		{"", false},
		{"models.user", false},
		{"a.b.C", false},
		{".User", false},
		{"models.", false},
		{"User-Response", false},
		{"User Response", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, looksLikeModelRef(tt.name))
		})
	}
}

// declaredTypeResolves has to agree with typeToSchema about what produces a $ref,
// because disagreement is either a warning nobody can act on or a real loss that
// goes unreported.
func TestDeclaredTypeResolves(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Structs["APIKeyResponse"] = &scanner.StructInfo{Name: "APIKeyResponse"}
	g.buildStructIndex()

	assert.True(t, g.declaredTypeResolves("APIKeyResponse"), "a registered model resolves")
	assert.True(t, g.declaredTypeResolves("string"), "a built-in resolves")
	assert.True(t, g.declaredTypeResolves("int64"), "a built-in resolves")
	assert.True(t, g.declaredTypeResolves("dto.APIKeyResponse"), "a qualified name whose bare form is a model resolves")

	assert.False(t, g.declaredTypeResolves("PaginatedAPIKeyResponse"),
		"a model the scan never reached does not resolve — this is the case that degraded to type: string")
}

// The check reports; it must not change what gets generated. It stays a warning
// until consumers have a release to clean up against, so a spec with an
// unresolved reference still generates exactly as it did before.
func TestCheckDeclaredSchemasResolveDoesNotFailGeneration(t *testing.T) {
	g := createTestGenerator()
	g.config.Strict = true
	g.scanner.Routes["listKeys"] = &scanner.RouteInfo{
		Method:      "GET",
		Path:        "/api-keys",
		OperationID: "listKeys",
		SourceFile:  "handler.go",
		Responses: []*scanner.ResponseInfo{
			{StatusCode: "200", Type: "PaginatedAPIKeyResponse"},
			{StatusCode: "412", Type: "when"},
		},
	}

	require.NoError(t, g.checkDeclaredSchemasResolve(),
		"an unresolved reference is reported, not fatal")
}
