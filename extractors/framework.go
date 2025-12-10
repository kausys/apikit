// Package extractors provides framework-specific parameter extraction
package extractors

import "github.com/kausys/apikit/parser"

// FrameworkExtractor defines how to extract parameters for a specific HTTP framework
type FrameworkExtractor interface {
	// Name returns the framework name (e.g., "http", "fiber", "gin")
	Name() string

	// Imports returns the framework-specific imports needed
	Imports() []string

	// HandlerSignature returns the handler function signature
	// e.g., "func(w http.ResponseWriter, r *http.Request)" for net/http
	HandlerSignature() string

	// ParseFuncSignature returns the parse function signature
	// e.g., "func parse%s(w http.ResponseWriter, r *http.Request, payload *%s) error"
	ParseFuncSignature() string

	// Extraction methods - each returns (code, imports)
	ExtractQuery(field *parser.Field, paramName, fieldName string) (string, []string)
	ExtractQuerySlice(field *parser.Field, paramName, fieldName string) (string, []string)
	ExtractPath(field *parser.Field, paramName, fieldName string) (string, []string)
	ExtractHeader(field *parser.Field, paramName, fieldName string) (string, []string)
	ExtractHeaderSlice(field *parser.Field, paramName, fieldName string) (string, []string)
	ExtractCookie(field *parser.Field, paramName, fieldName string) (string, []string)
	ExtractForm(field *parser.Field, paramName, fieldName string) (string, []string)
	ExtractFormSlice(field *parser.Field, paramName, fieldName string) (string, []string)
	ExtractFormFile(field *parser.Field, paramName, fieldPath string) (string, []string)
	ExtractFormFiles(field *parser.Field, paramName, fieldPath string) (string, []string)
	ExtractRequest(field *parser.Field) (string, []string)
	ExtractResponse(field *parser.Field) (string, []string)

	// Body parsing - returns the full body parsing code block
	ParseBody(structName string) string

	// Response methods
	WriteJSON(varName string) string
	WriteError(statusCode int, message string) string
}

// NewHTTPExtractor creates a new HTTP extractor
func NewHTTPExtractor() FrameworkExtractor {
	return &HTTPExtractor{}
}
