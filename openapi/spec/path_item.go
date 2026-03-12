package spec

// Describes the operations available on a single path.
// A Path Item MAY be empty, due to ACL constraints.
// The path itself is still exposed to the documentation viewer but they will not know which
// operations and parameters are available.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#path-item-object
type PathItem struct {
	// Allows for a referenced definition of this path item. The value MUST be in the form of a URL,
	// and the referenced structure MUST be in the form of a Path Item Object. In case a Path Item
	// Object field appears both in the defined object and the referenced object, the behavior is
	// undefined. See the rules for resolving Relative References.
	Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	// An optional string summary, intended to apply to all operations in this path.
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`
	// An optional string description, intended to apply to all operations in this path. CommonMark
	// syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// A definition of a GET operation on this path.
	Get *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	// A definition of a PUT operation on this path.
	Put *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	// A definition of a POST operation on this path.
	Post *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	// A definition of a DELETE operation on this path.
	Delete *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	// A definition of a OPTIONS operation on this path.
	Options *Operation `json:"options,omitempty" yaml:"options,omitempty"`
	// A definition of a HEAD operation on this path.
	Head *Operation `json:"head,omitempty" yaml:"head,omitempty"`
	// A definition of a PATCH operation on this path.
	Patch *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	// A definition of a TRACE operation on this path.
	Trace *Operation `json:"trace,omitempty" yaml:"trace,omitempty"`
	// An alternative servers array to service all operations in this path. If a servers array is
	// specified at the OpenAPI Object level, it will be overridden by this value.
	Servers []*Server `json:"servers,omitempty" yaml:"servers,omitempty"`
	// A list of parameters that are applicable for all the operations described under this path.
	// These parameters can be overridden at the operation level, but cannot be removed there.
	// The list MUST NOT include duplicated parameters. A unique parameter is defined by a combination
	// of a name and location. The list can use the Reference Object to link to parameters that are
	// defined in the OpenAPI Object's components.parameters.
	Parameters []*Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}
