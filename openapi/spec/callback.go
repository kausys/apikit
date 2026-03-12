package spec

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// A map of possible out-of band callbacks related to the parent operation. Each value in the map
// is a Path Item Object that describes a set of requests that may be initiated by the API provider
// and the expected responses. The key value used to identify the Path Item Object is an expression,
// evaluated at runtime, that identifies a URL to use for the callback operation.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#callback-object
type Callback struct {
	// A Path Item Object used to define a callback request and expected responses.
	PathItems map[string]*PathItem `json:"-" yaml:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
// It marshals the PathItems map directly as the Callback object.
func (c *Callback) MarshalJSON() ([]byte, error) {
	if c == nil || c.PathItems == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(c.PathItems)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It unmarshals the JSON directly into the PathItems map.
func (c *Callback) UnmarshalJSON(data []byte) error {
	if c == nil {
		return nil
	}

	// Initialize the PathItems map
	c.PathItems = make(map[string]*PathItem)

	// If the data is empty or just {}, return early
	if len(data) == 0 || string(data) == "{}" {
		return nil
	}

	// Unmarshal the data into the PathItems map
	return json.Unmarshal(data, &c.PathItems)
}

// MarshalYAML implements the yaml.Marshaler interface.
// It marshals the PathItems map directly as the Callback object.
func (c *Callback) MarshalYAML() (any, error) {
	if c == nil || c.PathItems == nil {
		return map[string]any{}, nil
	}
	return c.PathItems, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// It unmarshals the YAML directly into the PathItems map.
func (c *Callback) UnmarshalYAML(value *yaml.Node) error {
	if c == nil {
		return nil
	}

	// Initialize the PathItems map
	c.PathItems = make(map[string]*PathItem)

	// If the node is empty, return early
	if value.Kind == 0 {
		return nil
	}

	// Unmarshal the node into the PathItems map
	return value.Decode(&c.PathItems)
}
