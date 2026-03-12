package spec

// Adds metadata to a single tag that is used by the Operation Object. It is not mandatory to have
// a Tag Object per tag defined in the Operation Object instances.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#tag-object
type Tag struct {
	// REQUIRED. The name of the tag.
	Name string `json:"name" yaml:"name"`
	// A description for the tag. CommonMark syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Additional external documentation for this tag.
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}
