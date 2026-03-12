package spec

// Describes a single operation parameter.
// A unique parameter is defined by a combination of a name and location.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#parameter-object
type Parameter struct {
	// A reference to a parameter defined in components/parameters.
	Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`

	// REQUIRED. The name of the parameter. Parameter names are case sensitive.
	// If in is "path", the name field MUST correspond to a template expression occurring within the
	// path field in the Paths Object. See Path Templating for further information.
	// If in is "header" and the name field is "Accept", "Content-Type" or "Authorization", the
	// parameter definition SHALL be ignored.
	// For all other cases, the name corresponds to the parameter name used by the in field.
	Name string `json:"name" yaml:"name"`
	// REQUIRED. The location of the parameter. Possible values are "query", "header", "path" or "cookie".
	In string `json:"in" yaml:"in"`
	// A brief description of the parameter. This could contain examples of use. CommonMark syntax
	// MAY be used for rich text representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Determines whether this parameter is mandatory. If the parameter location is "path", this field
	// is REQUIRED and its value MUST be true. Otherwise, the field MAY be included and its default
	// value is false.
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`
	// Specifies that a parameter is deprecated and SHOULD be transitioned out of usage. Default value
	// is false.
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	// If true, clients MAY pass a zero-length string value in place of parameters that would otherwise
	// be omitted entirely, which the server SHOULD interpret as the parameter being unused. Default
	// value is false. This field is valid only for query parameters. Use of this field is NOT
	// RECOMMENDED, and it is likely to be removed in a later revision.
	AllowEmptyValue bool `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	// Describes how the parameter value will be serialized depending on the type of the parameter
	// value. Default values (based on value of in): for "query" - "form"; for "path" - "simple";
	// for "header" - "simple"; for "cookie" - "form".
	Style string `json:"style,omitempty" yaml:"style,omitempty"`
	// When this is true, parameter values of type array or object generate separate parameters for
	// each value of the array or key-value pair of the map. For other types of parameters this field
	// has no effect. When style is "form", the default value is true. For all other styles, the
	// default value is false.
	Explode *bool `json:"explode,omitempty" yaml:"explode,omitempty"`
	// When this is true, parameter values are serialized using reserved expansion, as defined by
	// RFC6570, which allows RFC3986's reserved character set, as well as percent-encoded triples,
	// to pass through unchanged. This field only applies to parameters with an in value of query.
	// The default value is false.
	AllowReserved bool `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
	// The schema defining the type used for the parameter.
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	// Example of the parameter's potential value; see Working With Examples.
	Example any `json:"example,omitempty" yaml:"example,omitempty"`
	// Examples of the parameter's potential value; see Working With Examples.
	Examples map[string]*Example `json:"examples,omitempty" yaml:"examples,omitempty"`
	// A map containing the representations for the parameter. The key is the media type and the value
	// describes it. The map MUST only contain one entry.
	Content map[string]*MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}
