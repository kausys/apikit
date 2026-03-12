package generator

import (
	"slices"

	"github.com/kausys/apikit/openapi/spec"
	"github.com/kausys/apikit/scanner"
)

// enumToSchema converts EnumInfo to spec.Schema.
func (g *Generator) enumToSchema(e *scanner.EnumInfo) *spec.Schema {
	schema := &spec.Schema{
		Description: e.Description,
	}

	// Set the type based on BaseType
	g.setSchemaType(schema, e.BaseType)

	// Add enum values (sorted by key for consistent output)
	if len(e.Values) > 0 {
		schema.Enum = sortEnumValues(e.Values)
	}

	// Add example if available
	if e.Example != nil {
		schema.Examples = []any{e.Example}
	}

	return schema
}

// sortEnumValues sorts enum values by key and returns a slice of values.
func sortEnumValues(values map[string]any) []any {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	result := make([]any, 0, len(values))
	for _, k := range keys {
		result = append(result, values[k])
	}
	return result
}

// createInlineEnumSchema creates an inline schema for an enum.
func (g *Generator) createInlineEnumSchema(e *scanner.EnumInfo) *spec.Schema {
	schema := &spec.Schema{
		Description: e.Description,
	}
	g.setSchemaType(schema, e.BaseType)

	if len(e.Values) > 0 {
		schema.Enum = sortEnumValues(e.Values)
	}

	if e.Example != nil {
		schema.Examples = []any{e.Example}
	}

	return schema
}

// findEnumInfo finds EnumInfo for a given type name.
func (g *Generator) findEnumInfo(typeName string) *scanner.EnumInfo {
	// Use scanner's GetEnumForType which handles aliases
	return g.scanner.GetEnumForType(typeName)
}
