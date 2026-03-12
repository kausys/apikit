// Package parser provides AST parsing capabilities for Go source files.
// It identifies handlers marked with special comments and extracts
// their metadata for code generation.
package parser

import "go/token"

// Handler represents a function marked with apikit:handler comment
type Handler struct {
	// Name is the function name
	Name string

	// Package is the package name where the handler is defined
	Package string

	// Receiver is the receiver type for methods (empty for functions)
	Receiver string

	// ParamType is the type of the request parameter
	ParamType string

	// ReturnType is the return type of the handler
	ReturnType string

	// Struct contains the parsed request struct information
	Struct *Struct

	// HasResponseWriter indicates if handler has http.ResponseWriter parameter
	HasResponseWriter bool

	// HasRequest indicates if handler has *http.Request parameter
	HasRequest bool

	// Position in source file (for error reporting)
	Pos token.Position
}

// Struct represents a request struct with its fields
type Struct struct {
	// Name is the struct name
	Name string

	// Fields are the struct fields
	Fields []Field

	// IsDTO indicates if this struct is marked with apikit:dto comment
	IsDTO bool
}

// Field represents a struct field with its tags and metadata
type Field struct {
	// Name is the field name
	Name string

	// Type is the Go type (e.g., "string", "*int", "[]string")
	Type string

	// StructTag is the complete struct tag string (e.g., `json:"name" query:"q" validate:"required"`)
	// Extractors should use reflect.StructTag(field.StructTag).Lookup() to get specific tags
	StructTag string

	// Comment-based annotations (e.g., // in:query, // in:path userId)
	InComment     string // Source extracted from "// in:xxx" comment (e.g., "query", "path")
	InCommentName string // Optional parameter name from "// in:xxx paramName" comment

	// Type information
	IsPointer bool   // Is this a pointer type (*string)
	IsSlice   bool   // Is this a slice type ([]string)
	SliceType string // Element type for slices

	// Special field types
	IsEmbedded       bool // Embedded struct
	IsBody           bool // Marked with "// in: body" comment
	IsRawBody        bool // Field named RawBody with type []byte
	IsResponseWriter bool // Field is http.ResponseWriter
	IsRequest        bool // Field is *http.Request
	IsFile           bool // Field is *multipart.FileHeader or []*multipart.FileHeader

	// Nested struct information
	NestedStruct *Struct // If this field is a struct type, contains its definition
	PackagePath  string  // Import path for the type (e.g., "myapp/pagination")
}

// Source represents information about where the handler was found
type Source struct {
	// Filename is the source file path
	Filename string

	// Package is the package name
	Package string
}

// ParseResult contains the results of parsing a file
type ParseResult struct {
	// Handlers found in the file
	Handlers []Handler

	// Structs found in the file (including DTOs)
	Structs map[string]*Struct

	// Source information
	Source Source

	// Errors encountered during parsing (non-fatal)
	Warnings []string
}
