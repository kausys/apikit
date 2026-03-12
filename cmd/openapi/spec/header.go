package spec

// Describes a single header for HTTP responses and for individual parts in multipart
// representations.
//
// The Header Object follows the structure of the Parameter Object, including determining its
// serialization strategy based on whether schema or content is present, with the following changes:
// 1. name MUST NOT be specified, it is given in the corresponding headers map.
// 2. in MUST NOT be specified, it is implicitly in header.
// 3. All traits that are affected by the location MUST be applicable to a location of header.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#header-object
type Header struct {
	// A brief description of the header. This could contain examples of use. CommonMark syntax MAY
	// be used for rich text representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Determines whether this header is mandatory. The default value is false.
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`
	// Specifies that the header is deprecated and SHOULD be transitioned out of usage. Default value
	// is false.
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	// Describes how the header value will be serialized. The default (and only legal value for
	// headers) is "simple".
	Style string `json:"style,omitempty" yaml:"style,omitempty"`
	// When this is true, header values of type array or object generate a single header whose value
	// is a comma-separated list of the array items or key-value pairs of the map. For other data
	// types this field has no effect. The default value is false.
	Explode *bool `json:"explode,omitempty" yaml:"explode,omitempty"`
	// The schema defining the type used for the header.
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	// Example of the header's potential value; see Working With Examples.
	Example any `json:"example,omitempty" yaml:"example,omitempty"`
	// Examples of the header's potential value; see Working With Examples.
	Examples map[string]*Example `json:"examples,omitempty" yaml:"examples,omitempty"`
	// A map containing the representations for the header. The key is the media type and the value
	// describes it. The map MUST only contain one entry.
	Content map[string]*MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}
