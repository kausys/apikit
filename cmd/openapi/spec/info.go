package spec

// The object provides metadata about the API.
// The metadata MAY be used by the clients if needed, and MAY be presented in editing or
// documentation generation tools for convenience.
//
// See: https://spec.openapis.org/oas/v3.1.1.html#info-object
type Info struct {
	// REQUIRED. The title of the API.
	Title string `json:"title" yaml:"title"`
	// A short summary of the API.
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`
	// A description of the API. CommonMark syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// A URL for the Terms of Service for the API. This MUST be in the form of a URL.
	TermsOfService string `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	// The contact information for the exposed API.
	Contact *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	// The license information for the exposed API.
	License *License `json:"license,omitempty" yaml:"license,omitempty"`
	// REQUIRED. The version of the OpenAPI Document (which is distinct from the OpenAPI Specification
	// version or the version of the API being described or the version of the OpenAPI Description).
	Version string `json:"version" yaml:"version"`
}
