package parser

// Handler Directive
const (
	// DirectiveHandler marks a function as an HTTP handler for code generation
	DirectiveHandler = "apikit:handler"
)

// Field Source Comments
// These comments indicate where parameters come from in HTTP requests
const (
	// SourcePath indicates the parameter comes from URL path
	// Example: // in:path
	SourcePath = "path"

	// SourceQuery indicates the parameter comes from query string
	// Example: // in:query
	SourceQuery = "query"

	// SourceHeader indicates the parameter comes from HTTP headers
	// Example: // in:header
	SourceHeader = "header"

	// SourceCookie indicates the parameter comes from cookies
	// Example: // in:cookie
	SourceCookie = "cookie"

	// SourceForm indicates the parameter comes from form data
	// Example: // in:form
	SourceForm = "form"

	// SourceBody indicates the parameter is the request body
	// Example: // in:body
	SourceBody = "body"
)

// Struct Tags
// Used in struct field tags for parameter extraction
const (
	// TagPath is the struct tag for path parameters
	// Example: `path:"userId"`
	TagPath = "path"

	// TagQuery is the struct tag for query parameters
	// Example: `query:"filter"`
	TagQuery = "query"

	// TagHeader is the struct tag for header parameters
	// Example: `header:"X-API-Key"`
	TagHeader = "header"

	// TagCookie is the struct tag for cookie parameters
	// Example: `cookie:"session_id"`
	TagCookie = "cookie"

	// TagForm is the struct tag for form fields
	// Example: `form:"title"`
	TagForm = "form"

	// TagJSON is the struct tag for JSON fields
	// Example: `json:"name"`
	TagJSON = "json"

	// TagValidate is the struct tag for validation rules
	// Example: `validate:"required,email"`
	TagValidate = "validate"
)

// Special Field Names
// Field names that have special meaning in handlers
const (
	// FieldNameRequest is the name for *http.Request fields
	FieldNameRequest = "Request"
	// FieldNameReq is an alias for Request
	FieldNameReq = "Req"

	// FieldNameResponseWriter is the name for http.ResponseWriter fields
	FieldNameResponseWriter = "ResponseWriter"
	// FieldNameResponse is an alias for ResponseWriter
	FieldNameResponse = "Response"
	// FieldNameWriter is an alias for ResponseWriter
	FieldNameWriter = "Writer"
	// FieldNameRes is an alias for ResponseWriter
	FieldNameRes = "Res"

	// FieldNameRawBody is the name for raw body []byte fields
	FieldNameRawBody = "RawBody"
	// FieldNameRaw is an alias for RawBody
	FieldNameRaw = "Raw"
)

// File Types
// Go types for file upload fields
const (
	// TypeFileHeader is the type for single file uploads
	TypeFileHeader = "*multipart.FileHeader"

	// TypeFileHeaderSlice is the type for multiple file uploads
	TypeFileHeaderSlice = "[]*multipart.FileHeader"
)

// Default Values
const (
	// DefaultMaxMemory is the default max memory for multipart form parsing (32MB)
	DefaultMaxMemory = 33554432

	// CommentPrefix is the prefix for extracting "in:xxx" comments
	CommentPrefix = "in:"

	// CommentPrefixDefault is the prefix for extracting "default:xxx" comments
	CommentPrefixDefault = "default:"
)
