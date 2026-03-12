package spec

// A single encoding definition applied to a single schema property.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#encoding-object
type Encoding struct {
	// The Content-Type for encoding a specific property. The value is a comma-separated list, each
	// element of which is either a specific media type (e.g. image/png) or a wildcard media type
	// (e.g. image/*). Default value depends on the property type.
	ContentType string `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	// A map allowing additional information to be provided as headers. Content-Type is described
	// separately and SHALL be ignored in this section. This field SHALL be ignored if the request
	// body media type is not a multipart.
	Headers map[string]*Header `json:"headers,omitempty" yaml:"headers,omitempty"`
	// Describes how a specific property value will be serialized depending on its type. See Parameter
	// Object for details on the style field. The behavior follows the same values as query parameters,
	// including default values. This field SHALL be ignored if the request body media type is not
	// application/x-www-form-urlencoded.
	Style string `json:"style,omitempty" yaml:"style,omitempty"`
	// When this is true, property values of type array or object generate separate parameters for each
	// value of the array, or key-value-pair of the map. For other types of properties this field has
	// no effect. When style is "form", the default value is true. For all other styles, the default
	// value is false. This field SHALL be ignored if the request body media type is not
	// application/x-www-form-urlencoded.
	Explode bool `json:"explode,omitempty" yaml:"explode,omitempty"`
	// When this is true, parameter values are serialized using reserved expansion, as defined by
	// RFC6570, which allows RFC3986's reserved character set, as well as percent-encoded triples, to
	// pass through unchanged. The default value is false. This field SHALL be ignored if the request
	// body media type is not application/x-www-form-urlencoded.
	AllowReserved bool `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
}
