package spec

// This is the root object of the OpenAPI Description.
//
// See: https://spec.openapis.org/oas/v3.1.1.html#openapi-object
type OpenAPI struct {
	// REQUIRED. This string MUST be the version number of the OpenAPI Specification that the OpenAPI
	// Document uses. The openapi field SHOULD be used by tooling to interpret the OpenAPI Document.
	// This is not related to the API info.version string.
	OpenAPI string `json:"openapi" yaml:"openapi"`
	// REQUIRED. Provides metadata about the API. The metadata MAY be used by tooling as required.
	Info *Info `json:"info,omitempty" yaml:"info,omitempty"`
	// The default value for the $schema keyword within Schema Objects contained within this OAS
	// document. This MUST be in the form of a URI.
	JSONSchemaDialect string `json:"jsonSchemaDialect,omitempty" yaml:"jsonSchemaDialect,omitempty"`
	// An array of Server Objects, which provide connectivity information to a target server. If the
	// servers field is not provided, or is an empty array, the default value would be a Server Object
	// with a url value of /.
	Servers []*Server `json:"servers,omitempty" yaml:"servers,omitempty"`
	// REQUIRED. The available paths and operations for the API.
	Paths *Paths `json:"paths,omitempty" yaml:"paths,omitempty"`
	// The incoming webhooks that MAY be received as part of this API and that the API consumer MAY
	// choose to implement. A map of webhook name to PathItem Object describing the request that may
	// be initiated by the API provider.
	Webhooks map[string]*PathItem `json:"webhooks,omitempty" yaml:"webhooks,omitempty"`
	// An element to hold various Objects for the OpenAPI Description.
	Components *Components `json:"components,omitempty" yaml:"components,omitempty"`
	// A declaration of which security mechanisms can be used across the API. The list of values includes
	// alternative Security Requirement Objects that can be used. Only one of the Security Requirement
	// Objects need to be satisfied to authorize a request. Individual operations can override this
	// definition. The list can be incomplete, up to being empty or absent. To make security explicitly
	// optional, an empty security requirement ({}) can be included in the array.
	Security []*SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	// A list of tags used by the OpenAPI Description with additional metadata. The order of the tags can
	// be used to reflect on their order by the parsing tools. Not all tags that are used by the Operation
	// Object must be declared. The tags that are not declared MAY be organized randomly or based on the
	// tools' logic. Each tag name in the list MUST be unique.
	Tags []*Tag `json:"tags,omitempty" yaml:"tags,omitempty"`
	// Additional external documentation.
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}
