package spec

// Each Media Type Object provides schema and examples for the media type identified by its key.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#media-type-object
type MediaType struct {
	// The schema defining the content of the request, response, parameter, or header.
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	// Example of the media type; see Working With Examples.
	Example any `json:"example,omitempty" yaml:"example,omitempty"`
	// Examples of the media type; see Working With Examples.
	Examples map[string]*Example `json:"examples,omitempty" yaml:"examples,omitempty"`
	// A map between a property name and its encoding information. The key, being the property name,
	// MUST exist in the schema as a property. The encoding field SHALL only apply to Request Body
	// Objects, and only when the media type is multipart or application/x-www-form-urlencoded.
	// If no Encoding Object is provided for a property, the behavior is determined by the default
	// values documented for the Encoding Object.
	Encoding map[string]*Encoding `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}
