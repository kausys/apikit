package spec

import (
	"encoding/json"
	"maps"

	"gopkg.in/yaml.v3"
)

// A container for the expected responses of an operation. The container maps a HTTP response code
// to the expected response.
//
// The documentation is not necessarily expected to cover all possible HTTP response codes because
// they may not be known in advance. However, documentation is expected to cover a successful
// operation response and any known errors.
//
// The default MAY be used as a default Response Object for all HTTP codes that are not covered
// individually by the Responses Object.
//
// The Responses Object MUST contain at least one response code, and if only one response code is
// provided it SHOULD be the response for a successful operation call.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#responses-object
type Responses struct {
	// The documentation of responses other than the ones declared for specific HTTP response codes.
	// Use this field to cover undeclared responses.
	Default *Response `json:"default,omitempty" yaml:"default,omitempty"`
	// Any HTTP status code can be used as the property name, but only one property per code, to
	// describe the expected response for that HTTP status code. This field MUST be enclosed in
	// quotation marks (for example, "200") for compatibility between JSON and YAML. To define a
	// range of response codes, this field MAY contain the uppercase wildcard character X. For
	// example, 2XX represents all response codes between 200 and 299.
	StatusCodes map[string]*Response `json:"-" yaml:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
// It marshals the StatusCodes map and Default response into a single JSON object.
func (r *Responses) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("{}"), nil
	}

	// Create a map to hold all responses
	result := make(map[string]*Response)

	// Add status codes
	if r.StatusCodes != nil {
		maps.Copy(result, r.StatusCodes)
	}

	// Add default response if present
	if r.Default != nil {
		result["default"] = r.Default
	}

	return json.Marshal(result)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It unmarshals a JSON object into StatusCodes map and Default response.
func (r *Responses) UnmarshalJSON(data []byte) error {
	if r == nil {
		return nil
	}

	// Initialize the StatusCodes map
	r.StatusCodes = make(map[string]*Response)

	// If the data is empty or just {}, return early
	if len(data) == 0 || string(data) == "{}" {
		return nil
	}

	// Unmarshal into a temporary map
	var temp map[string]*Response
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Process the map
	for code, response := range temp {
		if code == "default" {
			r.Default = response
		} else {
			r.StatusCodes[code] = response
		}
	}

	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
// It marshals the StatusCodes map and Default response into a single YAML object.
func (r *Responses) MarshalYAML() (any, error) {
	if r == nil {
		return map[string]any{}, nil
	}

	// Create a map to hold all responses
	result := make(map[string]*Response)

	// Add status codes
	if r.StatusCodes != nil {
		maps.Copy(result, r.StatusCodes)
	}

	// Add default response if present
	if r.Default != nil {
		result["default"] = r.Default
	}

	return result, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// It unmarshals a YAML node into StatusCodes map and Default response.
func (r *Responses) UnmarshalYAML(value *yaml.Node) error {
	if r == nil {
		return nil
	}

	// Initialize the StatusCodes map
	r.StatusCodes = make(map[string]*Response)

	// If the node is empty, return early
	if value.Kind == 0 {
		return nil
	}

	// Unmarshal into a temporary map
	var temp map[string]*Response
	if err := value.Decode(&temp); err != nil {
		return err
	}

	// Process the map
	for code, response := range temp {
		if code == "default" {
			r.Default = response
		} else {
			r.StatusCodes[code] = response
		}
	}

	return nil
}
