package scanner

// Core directives for top-level OpenAPI elements
const (
	// MetaDirective marks a comment block as containing OpenAPI metadata
	MetaDirective = "swagger:meta"
	// ModelDirective marks a struct as an OpenAPI schema/model
	ModelDirective = "swagger:model"
	// ParameterDirective marks a struct as OpenAPI parameters
	ParameterDirective = "swagger:parameters"
	// RouteDirective marks a function as an API endpoint
	RouteDirective = "swagger:route"
	// EnumDirective marks a type as an enum
	EnumDirective = "swagger:enum"
	// IgnoreDirective marks a field to be ignored
	IgnoreDirective = "swagger:ignore"
	// OneOfModelDirective marks a struct as a oneOf schema (polymorphic union)
	OneOfModelDirective = "swagger:oneOf"
	// AnyOfModelDirective marks a struct as an anyOf schema
	AnyOfModelDirective = "swagger:anyOf"
	// OneOfOptionDirective marks an embedded field as a oneOf option
	OneOfOptionDirective = "swagger:oneOfOption"
	// AnyOfOptionDirective marks an embedded field as an anyOf option
	AnyOfOptionDirective = "swagger:anyOfOption"
)

// Meta section directives
const (
	TitleDirective           = "Title:"
	VersionDirective         = "Version:"
	DescriptionDirective     = "Description:"
	TermsOfServiceDirective  = "TermsOfService:"
	ContactDirective         = "Contact:"
	LicenseDirective         = "License:"
	HostDirective            = "Host:"
	BasePathDirective        = "BasePath:"
	ExternalDocsDirective    = "ExternalDocs:"
	TagsDirective            = "Tags:"
	SecuritySchemesDirective = "SecuritySchemes:"
	SecurityDirective        = "Security:"
	ConsumesDirective        = "Consumes:"
	ProducesDirective        = "Produces:"
	ResponsesDirective       = "Responses:"
	ParametersDirective      = "Parameters:"
	SummaryDirective         = "Summary:"
	ServersDirective         = "Servers:"
)

// Schema composition directives (legacy inline style)
const (
	OneOfDirective = "oneOf:"
	AllOfDirective = "allOf:"
	AnyOfDirective = "anyOf:"
)

// Discriminator directives for oneOf/anyOf polymorphism
const (
	// DiscriminatorDirective specifies the property name for discriminator
	// Format: discriminator: propertyName
	DiscriminatorDirective = "discriminator:"
)

// Field-level directives
const (
	ExampleDirective     = "example:"
	DefaultDirective     = "default:"
	RequiredDirective    = "required:"
	NullableDirective    = "nullable:"
	FormatDirective      = "format:"
	InDirective          = "in:"
	MinimumDirective     = "min:"
	MaximumDirective     = "max:"
	MinLengthDirective   = "minLength:"
	MaxLengthDirective   = "maxLength:"
	PatternDirective     = "pattern:"
	MinItemsDirective    = "minItems:"
	MaxItemsDirective    = "maxItems:"
	UniqueItemsDirective = "uniqueItems:"
	ReadOnlyDirective    = "readOnly:"
	WriteOnlyDirective   = "writeOnly:"
)

// Route-specific directives
const (
	SummaryFieldDirective      = "summary:"
	DescriptionFieldDirective  = "description:"
	DeprecatedFieldDirective   = "deprecated"
	IgnoredParametersDirective = "IgnoredParameters:"
)

// Multi-spec directive
const (
	// SpecDirective specifies which spec(s) an element belongs to
	// Format: spec: name1 name2 name3
	SpecDirective = "spec:"
	// DefaultSpec is the name of the default spec for elements without spec: directive
	DefaultSpec = "default"
)

// Common prefixes and patterns
const (
	SwaggerPrefix = "swagger:"
	DashPrefix    = "-"
)

// HTTP methods
const (
	MethodGet     = "GET"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodPatch   = "PATCH"
	MethodHead    = "HEAD"
	MethodOptions = "OPTIONS"
)

// Content types
const (
	ContentTypeJSON      = "application/json"
	ContentTypeXML       = "application/xml"
	ContentTypeForm      = "application/x-www-form-urlencoded"
	ContentTypeMultipart = "multipart/form-data"
)

// Security scheme types
const (
	SecurityTypeAPIKey        = "apiKey"
	SecurityTypeHTTP          = "http"
	SecurityTypeOAuth2        = "oauth2"
	SecurityTypeOpenIDConnect = "openIdConnect"
)

// Parameter locations
const (
	ParamInQuery  = "query"
	ParamInHeader = "header"
	ParamInPath   = "path"
	ParamInCookie = "cookie"
)

// Schema types
const (
	TypeString  = "string"
	TypeNumber  = "number"
	TypeInteger = "integer"
	TypeBoolean = "boolean"
	TypeArray   = "array"
	TypeObject  = "object"
)

// String formats
const (
	FormatDate     = "date"
	FormatDateTime = "date-time"
	FormatPassword = "password"
	FormatByte     = "byte"   // For base64 encoded files
	FormatBinary   = "binary" // For file uploads in multipart/form-data
	FormatEmail    = "email"
	FormatUUID     = "uuid"
	FormatURI      = "uri"
	FormatInt32    = "int32"
	FormatInt64    = "int64"
	FormatFloat    = "float"
	FormatDouble   = "double"
)
