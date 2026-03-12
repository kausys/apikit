package spec

// An object grouping an internal or external example value with basic summary and description
// metadata. This object is typically used in fields named examples (plural), and is a referenceable
// alternative to older example (singular) fields that do not support referencing or metadata.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#example-object
type Example struct {
	// Short description for the example.
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`
	// Long description for the example. CommonMark syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Embedded literal example. The value field and externalValue field are mutually exclusive. To
	// represent examples of media types that cannot naturally represented in JSON or YAML, use a
	// string value to contain the example, escaping where necessary.
	Value any `json:"value,omitempty" yaml:"value,omitempty"`
	// A URL that points to the literal example. This provides the capability to reference examples
	// that cannot easily be included in JSON or YAML documents. The value field and externalValue
	// field are mutually exclusive.
	ExternalValue string `json:"externalValue,omitempty" yaml:"externalValue,omitempty"`
}
