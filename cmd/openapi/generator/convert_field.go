package generator

import (
	"strconv"
	"strings"

	"github.com/kausys/apikit/cmd/openapi/spec"
	"github.com/kausys/apikit/cmd/scanner"
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
		g.applyValidations(schema, f)
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
		g.applyValidations(schema, f)
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

	g.setSchemaType(schema, f.Type, f.QualifiedType)

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

// parameterName returns the name a parameter has on the wire: the one its
// binding tag gives (query:"limit"), else its json name, else its Go name.
func (g *Generator) parameterName(f *scanner.FieldInfo) string {
	if f.In != "" {
		if name := f.Tags[f.In]; name != "" {
			return name
		}
	}
	return g.getPropertyName(f)
}

// fieldToParameter converts a FieldInfo to spec.Parameter.
func (g *Generator) fieldToParameter(f *scanner.FieldInfo, path string) *spec.Parameter {
	paramName := g.parameterName(f)
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

	// Check if the field type is an enum — unless the config registered the
	// type, in which case it renders as configured (see configuredType).
	if enumInfo := g.findEnumInfo(f.Type); enumInfo != nil && configuredType(f.QualifiedType, f.Type) == nil {
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
		g.setSchemaType(schema, f.Type, f.QualifiedType)
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

// applyValidations applies validation rules to a schema: the directives the
// field's comments carry, then the rules of its validate tag.
func (g *Generator) applyValidations(schema *spec.Schema, f *scanner.FieldInfo) {
	if format, ok := f.Validations["format"]; ok {
		if format == scanner.FormatBinary {
			schema.ContentMediaType = "application/octet-stream"
		} else {
			schema.Format = format
		}
	}

	if v, ok := parseFloat(f.Validations["min"]); ok {
		schema.Minimum = &v
	}
	if v, ok := parseFloat(f.Validations["max"]); ok {
		schema.Maximum = &v
	}
	if v, ok := parseUint(f.Validations["minLength"]); ok {
		schema.MinLength = v
	}
	if v, ok := parseUint(f.Validations["maxLength"]); ok {
		schema.MaxLength = &v
	}
	if pattern, ok := f.Validations["pattern"]; ok {
		schema.Pattern = pattern
	}
	if v, ok := parseUint(f.Validations["minItems"]); ok {
		schema.MinItems = &v
	}
	if v, ok := parseUint(f.Validations["maxItems"]); ok {
		schema.MaxItems = &v
	}
	if f.Validations["uniqueItems"] == "true" {
		schema.UniqueItems = true
	}
	if f.Validations["readOnly"] == "true" {
		schema.ReadOnly = true
	}
	if f.Validations["writeOnly"] == "true" {
		schema.WriteOnly = true
	}

	applyRules(schema, f.Rules)
}

// applyRules maps the rules of a validate tag onto a schema. A size rule
// (min, max, len, gt, gte, lt, lte) means what the validator means by it on
// that kind of value: a length for a string, a count for an array or an
// object, a bound for a number. Rules after "dive" apply to the items.
func applyRules(schema *spec.Schema, rules []string) {
	target := schema
	for i, rule := range rules {
		key, value, _ := strings.Cut(rule, "=")
		switch key {
		case "dive":
			if target.Items == nil {
				return
			}
			// A $ref cannot carry keywords of its own in every tool; a
			// referenced item keeps its schema and the rules stop here.
			if target.Items.Ref != "" {
				return
			}
			applyRules(target.Items, rules[i+1:])
			return
		case "min", "gte":
			setLowerBound(target, value, false)
		case "gt":
			setLowerBound(target, value, true)
		case "max", "lte":
			setUpperBound(target, value, false)
		case "lt":
			setUpperBound(target, value, true)
		case "len", "eq":
			if key == "eq" && kindOf(target) != scanner.TypeString && kindOf(target) != scanner.TypeArray && kindOf(target) != scanner.TypeObject {
				if v, ok := parseFloat(value); ok {
					target.Const = v
				}
				continue
			}
			setLowerBound(target, value, false)
			setUpperBound(target, value, false)
		case "oneof":
			if values := oneOfValues(value, target); len(values) > 0 && target.Ref == "" {
				target.Enum = values
			}
		case "unique":
			if kindOf(target) == scanner.TypeArray {
				target.UniqueItems = true
			}
		default:
			if format, ok := ruleFormats[key]; ok && target.Format == "" {
				target.Format = format
			}
		}
	}
}

// ruleFormats are the validate rules that name a JSON Schema format.
var ruleFormats = map[string]string{
	"email":        "email",
	"uuid":         "uuid",
	"uuid4":        "uuid",
	"uuid_rfc4122": "uuid",
	"url":          "uri",
	"uri":          "uri",
	"http_url":     "uri",
	"ipv4":         "ipv4",
	"ipv6":         "ipv6",
	"hostname":     "hostname",
	"fqdn":         "hostname",
	"datetime":     "date-time",
	"e164":         "phone",
}

// kindOf returns the JSON type a schema is about, ignoring "null".
func kindOf(schema *spec.Schema) string {
	return schema.Type.Value()
}

func setLowerBound(schema *spec.Schema, value string, exclusive bool) {
	switch kindOf(schema) {
	case scanner.TypeString:
		if v, ok := parseUint(value); ok {
			if exclusive {
				v++
			}
			schema.MinLength = v
		}
	case scanner.TypeArray:
		if v, ok := parseUint(value); ok {
			if exclusive {
				v++
			}
			schema.MinItems = &v
		}
	case scanner.TypeObject:
		if v, ok := parseUint(value); ok {
			if exclusive {
				v++
			}
			schema.MinProperties = &v
		}
	case scanner.TypeInteger, scanner.TypeNumber:
		if v, ok := parseFloat(value); ok {
			if exclusive {
				schema.ExclusiveMinimum = &v
			} else {
				schema.Minimum = &v
			}
		}
	}
}

func setUpperBound(schema *spec.Schema, value string, exclusive bool) {
	switch kindOf(schema) {
	case scanner.TypeString:
		if v, ok := parseUint(value); ok {
			if exclusive && v > 0 {
				v--
			}
			schema.MaxLength = &v
		}
	case scanner.TypeArray:
		if v, ok := parseUint(value); ok {
			if exclusive && v > 0 {
				v--
			}
			schema.MaxItems = &v
		}
	case scanner.TypeObject:
		if v, ok := parseUint(value); ok {
			if exclusive && v > 0 {
				v--
			}
			schema.MaxProperties = &v
		}
	case scanner.TypeInteger, scanner.TypeNumber:
		if v, ok := parseFloat(value); ok {
			if exclusive {
				schema.ExclusiveMaximum = &v
			} else {
				schema.Maximum = &v
			}
		}
	}
}

// oneOfValues splits a oneof rule's space-separated values, typed like the
// schema: numbers for a numeric field, strings otherwise. The validator
// quotes a value with spaces in single quotes.
func oneOfValues(value string, schema *spec.Schema) []any {
	var values []any
	for _, raw := range splitOneOf(value) {
		switch kindOf(schema) {
		case scanner.TypeInteger, scanner.TypeNumber:
			if v, ok := parseFloat(raw); ok {
				values = append(values, v)
			}
		default:
			values = append(values, raw)
		}
	}
	return values
}

func splitOneOf(value string) []string {
	var out []string
	for value != "" {
		value = strings.TrimLeft(value, " ")
		if value == "" {
			break
		}
		if value[0] == '\'' {
			if end := strings.IndexByte(value[1:], '\''); end >= 0 {
				out = append(out, value[1:end+1])
				value = value[end+2:]
				continue
			}
		}
		word, rest, _ := strings.Cut(value, " ")
		out = append(out, word)
		value = rest
	}
	return out
}

func parseFloat(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

func parseUint(s string) (uint64, bool) {
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 64)
	return v, err == nil
}
