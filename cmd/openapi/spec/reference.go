package spec

// A simple object to allow referencing other components in the OpenAPI Description, internally
// and externally.
//
// The Reference Object is defined by JSON Reference and follows the same structure, behavior and
// rules. For this specification, reference resolution is accomplished as defined by the JSON
// Reference specification and not by the JSON Schema specification.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#reference-object
type Reference struct {
	// REQUIRED. The reference string.
	Ref string `json:"$ref" yaml:"$ref"`
}
