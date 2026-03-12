// Package scanner provides code scanning capabilities to extract OpenAPI
// information from Go source code using structured comments (directives).
package scanner

// MetaInfo contains OpenAPI specification metadata extracted from swagger:meta directive.
type MetaInfo struct {
	Title           string
	Description     string
	Version         string
	TermsOfService  string
	Contact         *ContactInfo
	License         *LicenseInfo
	Host            string
	BasePath        string
	ExternalDocs    *ExternalDocsInfo
	Tags            []*TagInfo
	SecuritySchemes map[string]*SecuritySchemeInfo
	Consumes        []string
	Produces        []string
	Schemes         []string
	Specs           []string // Multi-spec: which specs this meta belongs to (empty = general/default)
}

// ContactInfo represents contact information for the API.
type ContactInfo struct {
	Name  string
	URL   string
	Email string
}

// LicenseInfo represents license information for the API.
type LicenseInfo struct {
	Name string
	URL  string
}

// ExternalDocsInfo represents external documentation.
type ExternalDocsInfo struct {
	Description string
	URL         string
}

// TagInfo represents a tag used to organize API operations.
type TagInfo struct {
	Name         string
	Description  string
	ExternalDocs *ExternalDocsInfo
}

// SecuritySchemeInfo represents a security scheme definition.
type SecuritySchemeInfo struct {
	Type             string // apiKey, http, oauth2, openIdConnect
	Description      string
	Name             string // For apiKey
	In               string // For apiKey: query, header, cookie
	Scheme           string // For http: bearer, basic
	BearerFormat     string // For http bearer
	Flows            *OAuthFlowsInfo
	OpenIdConnectUrl string
}

// OAuthFlowsInfo contains OAuth2 flow configurations.
type OAuthFlowsInfo struct {
	Implicit          *OAuthFlowInfo
	Password          *OAuthFlowInfo
	ClientCredentials *OAuthFlowInfo
	AuthorizationCode *OAuthFlowInfo
}

// OAuthFlowInfo represents a single OAuth2 flow configuration.
type OAuthFlowInfo struct {
	AuthorizationURL string
	TokenURL         string
	RefreshURL       string
	Scopes           map[string]string
}

// EnumInfo contains information about an enum type.
type EnumInfo struct {
	TypeName    string
	BaseType    string
	Example     any
	Values      map[string]any
	Description string
	SourceFile  string
}

// EmbeddedTypeInfo contains information about an embedded type with its position.
type EmbeddedTypeInfo struct {
	Name  string // Type name (e.g., "pagination.Pagination")
	Index int    // Position in the struct where it was declared
}

// UnderlyingKind represents the kind of Go type underlying a swagger:model declaration.
type UnderlyingKind int

const (
	KindStruct    UnderlyingKind = iota // type T struct{...}
	KindArray                           // type T []E
	KindMap                             // type T map[K]V
	KindPrimitive                       // type T string, type T int, etc.
)

// StructInfo contains information about a struct marked as model or parameters.
type StructInfo struct {
	Name              string
	Fields            []*FieldInfo
	EmbeddedTypes     []string            // Embedded types that need to be resolved (e.g., "pagination.Pagination")
	EmbeddedTypeInfos []*EmbeddedTypeInfo // Embedded types with position information
	Description       string
	IsParameter       bool
	IsModel           bool
	SourceFile        string
	OneOf             []string // Legacy: inline oneOf references from "oneOf:" directive
	AllOf             []string // Legacy: inline allOf references from "allOf:" directive
	AnyOf             []string // Legacy: inline anyOf references from "anyOf:" directive
	Specs             []string // Multi-spec: which specs this model belongs to (empty = all specs)

	UnderlyingKind UnderlyingKind // What kind of Go type this model wraps
	ElementType    string         // For arrays: element type name; for maps: value type name
	MapKeyType     string         // For maps: key type name

	// New oneOf/anyOf model support (swagger:oneOf / swagger:anyOf)
	IsOneOfModel  bool               // True if struct is marked with swagger:oneOf
	IsAnyOfModel  bool               // True if struct is marked with swagger:anyOf
	OneOfOptions  []string           // Types marked with swagger:oneOfOption
	AnyOfOptions  []string           // Types marked with swagger:anyOfOption
	Discriminator *DiscriminatorInfo // Discriminator configuration for polymorphism
}

// DiscriminatorInfo contains discriminator configuration for oneOf/anyOf schemas.
type DiscriminatorInfo struct {
	PropertyName string            // The property name that holds the discriminating value
	Mapping      map[string]string // Maps discriminator values to schema names
}

// FieldInfo contains information about a struct field.
type FieldInfo struct {
	Name             string
	Type             string
	Description      string
	Default          string
	Example          string
	Required         bool
	Nullable         bool
	Validations      map[string]string
	Enum             string
	Tags             map[string]string
	IsArray          bool
	IsPointer        bool
	IsMap            bool
	MapKeyType       string
	IsRequestBody    bool
	In               string // Parameter location: query, path, header, cookie, body
	InlineStruct     *StructInfo
	IsInlineStruct   bool
	HasOmitempty     bool
	ExplicitRequired bool
	ExplicitOptional bool
	Index            int // Position in the original struct declaration (for ordering)
}

// RouteInfo contains information about an API route/endpoint.
type RouteInfo struct {
	Method            string
	Path              string
	Tags              []string
	OperationID       string
	Summary           string
	Description       string
	Deprecated        bool
	Responses         []*ResponseInfo
	Security          []string
	Consumes          []string
	Produces          []string
	IgnoredParameters []string
	SourceFile        string
	Specs             []string // Multi-spec: which specs this route belongs to (empty = default spec)
}

// ResponseInfo contains information about an API response.
type ResponseInfo struct {
	StatusCode  string
	Type        string
	Description string
	IsArray     bool
	IsMap       bool
	MapKeyType  string
}
