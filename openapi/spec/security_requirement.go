package spec

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// Security Requirement Object
//
// Lists the required security schemes to execute this operation. The name used for each property MUST
// correspond to a security scheme declared in the Security Schemes under the Components Object.
//
// A Security Requirement Object MAY refer to multiple security schemes in which case all schemes MUST be
// satisfied for a request to be authorized. This enables support for scenarios where multiple query
// parameters or HTTP headers are required to convey security information.
//
// When the security field is defined on the OpenAPI Object or Operation Object and contains multiple
// Security Requirement Objects, only one of the entries in the list needs to be satisfied to authorize
// the request. This enables support for scenarios where the API allows multiple, independent security schemes.
//
// An empty Security Requirement Object ({}) indicates anonymous access is supported.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#security-requirement-object
type SecurityRequirement struct {
	// Each name MUST correspond to a security scheme which is declared in the Security Schemes under
	// the Components Object. If the security scheme is of type "oauth2" or "openIdConnect", then the
	// value is a list of scope names required for the execution, and the list MAY be empty if
	// authorization does not require a specified scope. For other security scheme types, the array
	// MUST be empty.
	Requirements map[string][]string `json:"-" yaml:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
// It marshals the Requirements map directly as the SecurityRequirement object.
func (s *SecurityRequirement) MarshalJSON() ([]byte, error) {
	if s == nil || s.Requirements == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(s.Requirements)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It unmarshals the JSON directly into the Requirements map.
func (s *SecurityRequirement) UnmarshalJSON(data []byte) error {
	if s == nil {
		return nil
	}

	// Initialize the Requirements map
	s.Requirements = make(map[string][]string)

	// If the data is empty or just {}, return early
	if len(data) == 0 || string(data) == "{}" {
		return nil
	}

	// Unmarshal the data into the Requirements map
	return json.Unmarshal(data, &s.Requirements)
}

// MarshalYAML implements the yaml.Marshaler interface.
// It marshals the Requirements map directly as the SecurityRequirement object.
func (s *SecurityRequirement) MarshalYAML() (any, error) {
	if s == nil || s.Requirements == nil {
		return map[string]any{}, nil
	}
	return s.Requirements, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// It unmarshals the YAML directly into the Requirements map.
func (s *SecurityRequirement) UnmarshalYAML(value *yaml.Node) error {
	if s == nil {
		return nil
	}

	// Initialize the Requirements map
	s.Requirements = make(map[string][]string)

	// If the node is empty, return early
	if value.Kind == 0 {
		return nil
	}

	// Unmarshal the node into the Requirements map
	return value.Decode(&s.Requirements)
}
