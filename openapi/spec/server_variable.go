package spec

// An object representing a Server Variable for server URL template substitution.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#server-variable-object
type ServerVariable struct {
	// An enumeration of string values to be used if the substitution options are from a limited set.
	// The array SHOULD NOT be empty.
	Enum []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	// REQUIRED. The default value to use for substitution, which SHALL be sent if an alternate value
	// is not supplied. If the enum is defined, the value SHOULD exist in the enum's values. Note that
	// this behavior is different from the Schema Object's default keyword, which documents the
	// receiver's behavior rather than inserting the value into the data.
	Default string `json:"default" yaml:"default"`
	// An optional description for the server variable. CommonMark syntax MAY be used for rich text
	// representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}
