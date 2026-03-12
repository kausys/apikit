package spec

// License information for the exposed API.
//
// See: https://spec.openapis.org/oas/v3.1.1.html#license-object
type License struct {
	// REQUIRED. The license name used for the API.
	Name string `json:"name" yaml:"name"`
	// An SPDX license expression for the API. Mutually exclusive with URL.
	Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty"`
	// A URL for the license used for the API. This MUST be in the form of a URL.
	// Mutually exclusive with Identifier.
	URL string `json:"url,omitempty" yaml:"url,omitempty"`
}
