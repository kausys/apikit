package spec

// XML Object
//
// A metadata object that allows for more fine-tuned XML model definitions.
//
// When using arrays, XML element names are not inferred (for singular/plural forms) and the name field
// SHOULD be used to add that information. See examples for expected behavior.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#xml-object
type XML struct {
	// Replaces the name of the element/attribute used for the described schema property. When defined
	// within items, it will affect the name of the individual XML elements within the list. When defined
	// alongside type being "array" (outside the items), it will affect the wrapping element if and only
	// if wrapped is true. If wrapped is false, it will be ignored.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// The URI of the namespace definition. Value MUST be in the form of a non-relative URI.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	// The prefix to be used for the name.
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	// Declares whether the property definition translates to an attribute instead of an element.
	// Default value is false.
	Attribute bool `json:"attribute,omitempty" yaml:"attribute,omitempty"`
	// MAY be used only for an array definition. Signifies whether the array is wrapped (for example,
	// <books><book/><book/></books>) or unwrapped (<book/><book/>). Default value is false. The definition
	// takes effect only when defined alongside type being "array" (outside the items).
	Wrapped bool `json:"wrapped,omitempty" yaml:"wrapped,omitempty"`
}
