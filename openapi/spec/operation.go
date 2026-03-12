package spec

// Describes a single API operation on a path.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#operation-object
type Operation struct {
	// A list of tags for API documentation control. Tags can be used for logical grouping of
	// operations by resources or any other qualifier.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	// A short summary of what the operation does.
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`
	// A verbose explanation of the operation behavior. CommonMark syntax MAY be used for rich text
	// representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Additional external documentation for this operation.
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	// Unique string used to identify the operation. The id MUST be unique among all operations
	// described in the API. The operationId value is case-sensitive. Tools and libraries MAY use
	// the operationId to uniquely identify an operation, therefore, it is RECOMMENDED to follow
	// common programming naming conventions.
	OperationID string `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	// A list of parameters that are applicable for this operation. If a parameter is already defined
	// in the Path Item, the new definition will override it but can never remove it. The list MUST NOT
	// include duplicated parameters. A unique parameter is defined by a combination of a name and
	// location. The list can use the Reference Object to link to parameters that are defined in the
	// OpenAPI Object's components.parameters.
	Parameters []*Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	// The request body applicable for this operation. The requestBody is only supported in HTTP
	// methods where the HTTP 1.1 specification RFC7231 has explicitly defined semantics for request
	// bodies. In other cases where the HTTP spec is vague (such as GET, HEAD and DELETE), requestBody
	// SHALL be ignored by consumers.
	RequestBody *RequestBody `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	// REQUIRED. The list of possible responses as they are returned from executing this operation.
	Responses *Responses `json:"responses" yaml:"responses"`
	// A map of possible out-of band callbacks related to the parent operation. The key is a unique
	// identifier for the Callback Object. Each value in the map is a Callback Object that describes
	// a request that may be initiated by the API provider and the expected responses.
	Callbacks map[string]*Callback `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
	// Declares this operation to be deprecated. Consumers SHOULD refrain from usage of the declared
	// operation. Default value is false.
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	// A declaration of which security mechanisms can be used for this operation. The list of values
	// includes alternative Security Requirement Objects that can be used. Only one of the Security
	// Requirement Objects need to be satisfied to authorize a request. To make security optional,
	// an empty security requirement ({}) can be included in the array. This definition overrides any
	// declared top-level security. To remove a top-level security declaration, an empty array can be used.
	Security []*SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	// An alternative servers array to service this operation. If a servers array is specified at the
	// Path Item Object or OpenAPI Object level, it will be overridden by this value.
	Servers []*Server `json:"servers,omitempty" yaml:"servers,omitempty"`
}
