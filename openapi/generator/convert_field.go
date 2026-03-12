package generator

import (
	"strconv"
	"strings"

	"github.com/kausys/apikit/openapi/spec"
	"github.com/kausys/apikit/scanner"
)

// getPropertyName returns the JSON property name for a field.
func (g *Generator) getPropertyName(f *scanner.FieldInfo) string {
	if jsonName, ok := f.Tags["json"]; ok && jsonName != "" {
		return jsonName
	}
	return f.Name
}

// fieldToSchema converts FieldInfo to spec.Schema.
func (g *Generator) fieldToSchema(f *scanner.FieldInfo) *spec.Schema {
	// Handle arrays
	if f.IsArray {
		schema := &spec.Schema{
			Type:        spec.NewSchemaType(scanner.TypeArray),
			Description: f.Description,
			Items:       g.typeToSchema(f.Type),
		}
		if f.Nullable {
			schema.Type = schema.Type.WithNull()
		}
		return schema
	}

	// Handle maps
	if f.IsMap {
		schema := &spec.Schema{
			Type:                 spec.NewSchemaType(scanner.TypeObject),
			Description:          f.Description,
			AdditionalProperties: g.typeToSchema(f.Type),
		}
		if f.Nullable {
			schema.Type = schema.Type.WithNull()
		}
		return schema
	}

	// Check if this is a reference type (model or enum)
	if g.isReferenceType(f.Type) {
		schema := g.typeToSchema(f.Type)
		// For enums, apply field-level overrides (description, example)
		if schema.Ref == "" && schema.Enum != nil {
			if f.Description != "" {
				schema.Description = f.Description
			}
			if f.Example != "" {
				schema.Examples = []any{f.Example}
			}
		}
		return schema
	}

	// Handle basic/primitive types
	schema := &spec.Schema{
		Description: f.Description,
	}

	g.setSchemaType(schema, f.Type)

	if f.Nullable {
		schema.Type = schema.Type.WithNull()
	}

	g.applyValidations(schema, f)

	// Set example (cast to schema type)
	if f.Example != "" {
		schema.Examples = []any{castToSchemaType(f.Example, schema.Type)}
	}

	// Set default (cast to schema type)
	if f.Default != "" {
		schema.Default = castToSchemaType(f.Default, schema.Type)
	}

	return schema
}

// fieldToParameter converts a FieldInfo to spec.Parameter.
func (g *Generator) fieldToParameter(f *scanner.FieldInfo, path string) *spec.Parameter {
	paramName := g.getPropertyName(f)
	if paramName == "" || paramName == "-" {
		return nil
	}

	// Determine parameter location (in)
	in := f.In
	if in == "" {
		in = "query" // default
	}
	// Check if it's a path parameter
	if strings.Contains(path, "{"+paramName+"}") {
		in = "path"
	}

	// Create schema for the parameter
	var schema *spec.Schema

	// Check if the field type is an enum
	if enumInfo := g.findEnumInfo(f.Type); enumInfo != nil {
		if g.config.EnumRefs {
			g.markSchemaAsReferenced(enumInfo.TypeName)
			schema = &spec.Schema{Ref: "#/components/schemas/" + enumInfo.TypeName}
		} else {
			schema = g.createInlineEnumSchema(enumInfo)
			// Override example if field has its own
			if f.Example != "" {
				schema.Examples = []any{castToSchemaType(f.Example, schema.Type)}
			}
		}
	} else {
		schema = &spec.Schema{}
		g.setSchemaType(schema, f.Type)
		g.applyValidations(schema, f)

		if f.Example != "" {
			schema.Examples = []any{castToSchemaType(f.Example, schema.Type)}
		}
		if f.Default != "" {
			schema.Default = castToSchemaType(f.Default, schema.Type)
		}
	}

	param := &spec.Parameter{
		Name:        paramName,
		In:          in,
		Description: f.Description,
		Required:    f.Required || in == "path", // path parameters are always required
		Schema:      schema,
	}

	return param
}

// applyValidations applies validation rules to a schema.
func (g *Generator) applyValidations(schema *spec.Schema, f *scanner.FieldInfo) {
	if format, ok := f.Validations["format"]; ok {
		if format == scanner.FormatBinary {
			schema.ContentMediaType = "application/octet-stream"
		} else {
			schema.Format = format
		}
	}

	if min, ok := f.Validations["min"]; ok {
		if v, err := strconv.ParseFloat(min, 64); err == nil {
			schema.Minimum = new(v)
		}
	}

	if max, ok := f.Validations["max"]; ok {
		if v, err := strconv.ParseFloat(max, 64); err == nil {
			schema.Maximum = new(v)
		}
	}

	if minLen, ok := f.Validations["minLength"]; ok {
		if v, err := strconv.ParseUint(minLen, 10, 64); err == nil {
			schema.MinLength = v
		}
	}

	if maxLen, ok := f.Validations["maxLength"]; ok {
		if v, err := strconv.ParseUint(maxLen, 10, 64); err == nil {
			schema.MaxLength = new(v)
		}
	}

	if pattern, ok := f.Validations["pattern"]; ok {
		schema.Pattern = pattern
	}
}
