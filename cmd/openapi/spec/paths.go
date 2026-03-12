package spec

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// Holds the relative paths to the individual endpoints and their operations.
// The path is appended to the URL from the Server Object in order to construct the full URL.
// The Paths Object MAY be empty, due to Access Control List (ACL) constraints.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#paths-object
type Paths struct {
	// A relative path to an individual endpoint. The field name MUST begin with a forward slash (/).
	// The path is appended (no relative URL resolution) to the expanded URL from the Server Object's
	// url field in order to construct the full URL. Path templating is allowed. When matching URLs,
	// concrete (non-templated) paths would be matched before their templated counterparts. Templated
	// paths with the same hierarchy but different templated names MUST NOT exist as they are identical.
	// In case of ambiguous matching, it's up to the tooling to decide which one to use.
	PathItems map[string]*PathItem `json:"-" yaml:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
// It marshals the PathItems map directly as the Paths object.
func (p *Paths) MarshalJSON() ([]byte, error) {
	if p == nil || p.PathItems == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(p.PathItems)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It unmarshals the JSON directly into the PathItems map.
func (p *Paths) UnmarshalJSON(data []byte) error {
	if p == nil {
		return nil
	}

	// Initialize the PathItems map
	p.PathItems = make(map[string]*PathItem)

	// If the data is empty or just {}, return early
	if len(data) == 0 || string(data) == "{}" {
		return nil
	}

	// Unmarshal the data into the PathItems map
	return json.Unmarshal(data, &p.PathItems)
}

// MarshalYAML implements the yaml.Marshaler interface.
// It marshals the PathItems map directly as the Paths object.
func (p *Paths) MarshalYAML() (any, error) {
	if p == nil || p.PathItems == nil {
		return map[string]any{}, nil
	}
	return p.PathItems, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// It unmarshals the YAML directly into the PathItems map.
func (p *Paths) UnmarshalYAML(value *yaml.Node) error {
	if p == nil {
		return nil
	}

	// Initialize the PathItems map
	p.PathItems = make(map[string]*PathItem)

	// If the node is empty, return early
	if value.Kind == 0 {
		return nil
	}

	// Unmarshal the node into the PathItems map
	return value.Decode(&p.PathItems)
}
