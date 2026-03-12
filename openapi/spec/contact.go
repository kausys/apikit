package spec

// Contact information for the exposed API.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#contact-object
type Contact struct {
	// The identifying name of the contact person/organization.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// The URL for the contact information. This MUST be in the form of a URL.
	URL string `json:"url,omitempty" yaml:"url,omitempty"`
	// The email address of the contact person/organization. This MUST be in the form of an email address.
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}
