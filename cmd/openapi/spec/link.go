package spec

// The Link Object represents a possible design-time link for a response. The presence of a link
// does not guarantee the caller's ability to successfully invoke it, rather it provides a known
// relationship and traversal mechanism between responses and other operations.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#link-object
type Link struct {
	// A URI reference to an OAS operation. This field is mutually exclusive of the operationId field,
	// and MUST point to an Operation Object.
	OperationRef string `json:"operationRef,omitempty" yaml:"operationRef,omitempty"`
	// The name of an existing, resolvable OAS operation, as defined with a unique operationId. This
	// field is mutually exclusive of the operationRef field.
	OperationID string `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	// A map representing parameters to pass to an operation as specified with operationId or
	// identified via operationRef. The key is the parameter name to be used (optionally qualified
	// with the parameter location, e.g. path.id for an id parameter in the path), whereas the value
	// can be a constant or an expression to be evaluated and passed to the linked operation.
	Parameters map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	// A literal value or {expression} to use as a request body when calling the target operation.
	RequestBody any `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	// A description of the link. CommonMark syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// A server object to be used by the target operation.
	Server *Server `json:"server,omitempty" yaml:"server,omitempty"`
}
