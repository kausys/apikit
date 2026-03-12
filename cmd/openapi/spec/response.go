package spec

// Describes a single response from an API operation, including design-time, static links to
// operations based on the response.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#response-object
type Response struct {
	// REQUIRED. A description of the response. CommonMark syntax MAY be used for rich text
	// representation.
	Description string `json:"description" yaml:"description"`
	// Maps a header name to its definition. RFC7230 states header names are case insensitive. If a
	// response header is defined with the name "Content-Type", it SHALL be ignored.
	Headers map[string]*Header `json:"headers,omitempty" yaml:"headers,omitempty"`
	// A map containing descriptions of potential response payloads. The key is a media type or media
	// type range and the value describes it. For responses that match multiple keys, only the most
	// specific key is applicable. e.g. "text/plain" overrides "text/*"
	Content map[string]*MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	// A map of operations links that can be followed from the response. The key of the map is a short
	// name for the link, following the naming constraints of the names for Component Objects.
	Links map[string]*Link `json:"links,omitempty" yaml:"links,omitempty"`
}
