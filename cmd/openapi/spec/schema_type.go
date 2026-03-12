package spec

import (
	"encoding/json"
	"slices"

	"gopkg.in/yaml.v3"
)

// SchemaType represents the JSON Schema "type" keyword which can be either a single string
// or an array of strings. In OpenAPI 3.1 (aligned with JSON Schema 2020-12), nullable types
// are expressed as ["string", "null"] instead of using the separate "nullable" keyword.
type SchemaType struct {
	values []string
}

// NewSchemaType creates a SchemaType with a single type value.
func NewSchemaType(t string) SchemaType {
	if t == "" {
		return SchemaType{}
	}
	return SchemaType{values: []string{t}}
}

// NewSchemaTypes creates a SchemaType with multiple type values.
func NewSchemaTypes(types ...string) SchemaType {
	if len(types) == 0 {
		return SchemaType{}
	}
	return SchemaType{values: types}
}

// Value returns the single type string. If multiple types are present, returns the first non-null type.
// Returns empty string if no types are set.
func (st SchemaType) Value() string {
	for _, v := range st.values {
		if v != "null" {
			return v
		}
	}
	if len(st.values) > 0 {
		return st.values[0]
	}
	return ""
}

// Values returns all type values.
func (st SchemaType) Values() []string {
	return st.values
}

// IsEmpty returns true if no types are set.
func (st SchemaType) IsEmpty() bool {
	return len(st.values) == 0
}

// IsZero returns true if the SchemaType has no values (for omitzero support).
func (st SchemaType) IsZero() bool {
	return len(st.values) == 0
}

// Contains returns true if the SchemaType contains the given type.
func (st SchemaType) Contains(t string) bool {
	return slices.Contains(st.values, t)
}

// WithNull returns a new SchemaType that includes "null" in addition to existing types.
// If "null" is already present, returns the same SchemaType.
func (st SchemaType) WithNull() SchemaType {
	if st.Contains("null") {
		return st
	}
	newValues := make([]string, len(st.values), len(st.values)+1)
	copy(newValues, st.values)
	newValues = append(newValues, "null")
	return SchemaType{values: newValues}
}

// MarshalJSON implements json.Marshaler.
// Single value serializes as "string", multiple values as ["string", "null"].
func (st SchemaType) MarshalJSON() ([]byte, error) {
	if len(st.values) == 0 {
		return json.Marshal(nil)
	}
	if len(st.values) == 1 {
		return json.Marshal(st.values[0])
	}
	return json.Marshal(st.values)
}

// UnmarshalJSON implements json.Unmarshaler.
// Accepts both "string" and ["string", "null"] forms.
func (st *SchemaType) UnmarshalJSON(data []byte) error {
	// Handle JSON null
	if string(data) == "null" {
		st.values = nil
		return nil
	}

	// Try single string first
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		st.values = []string{single}
		return nil
	}

	// Try array of strings
	var multi []string
	if err := json.Unmarshal(data, &multi); err == nil {
		st.values = multi
		return nil
	}

	// Unrecognized
	st.values = nil
	return nil
}

// MarshalYAML implements yaml.Marshaler.
// Single value serializes as a scalar, multiple values as a sequence.
func (st SchemaType) MarshalYAML() (any, error) {
	if len(st.values) == 0 {
		return nil, nil //nolint:nilnil // yaml.Marshaler interface requires nil,nil to omit the field
	}
	if len(st.values) == 1 {
		return st.values[0], nil
	}
	return st.values, nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
// Accepts both scalar and sequence forms.
func (st *SchemaType) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		st.values = []string{value.Value}
	case yaml.SequenceNode:
		st.values = make([]string, 0, len(value.Content))
		for _, node := range value.Content {
			st.values = append(st.values, node.Value)
		}
	default:
		st.values = nil
	}
	return nil
}
