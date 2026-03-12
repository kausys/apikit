package spec

// Allows referencing an external resource for extended documentation.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#external-documentation-object
type ExternalDocs struct {
	// A description of the target documentation. CommonMark syntax MAY be used for rich text
	// representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// REQUIRED. The URL for the target documentation. This MUST be in the form of a URL.
	URL string `json:"url" yaml:"url"`
}
