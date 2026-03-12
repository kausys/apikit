package sdkgen

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/kausys/apikit/cmd/openapi/spec"
)

// schemaConverter handles OpenAPI schema → Go type conversions.
type schemaConverter struct {
	customTypes map[string]CustomTypeConfig
	imports     map[string]string       // import path → alias (or "")
	schemas     map[string]*spec.Schema // components/schemas for $ref resolution
}

func newSchemaConverter(customTypes map[string]CustomTypeConfig) *schemaConverter {
	return &schemaConverter{
		customTypes: customTypes,
		imports:     make(map[string]string),
	}
}

// goType converts an OpenAPI schema to a Go type string.
// It returns the Go type and whether the schema represents an array.
func (sc *schemaConverter) goType(schema *spec.Schema, required bool) string {
	if schema == nil {
		return "any"
	}

	// Handle $ref
	if schema.Ref != "" {
		typeName := extractRefName(schema.Ref)
		if !required {
			return "*" + typeName
		}
		return typeName
	}

	// Check custom types first (by format)
	if schema.Format != "" && sc.customTypes != nil {
		if ct, ok := sc.customTypes[schema.Format]; ok {
			sc.addImport(ct.Import, "")
			return ct.GoType
		}
	}

	switch schema.Type.Value() {
	case "string":
		return sc.stringType(schema)
	case "integer":
		return sc.integerType(schema)
	case "number":
		return sc.numberType(schema)
	case "boolean":
		return "bool"
	case "array":
		itemType := sc.goType(schema.Items, true)
		return "[]" + itemType
	case "object":
		return sc.objectType(schema)
	default:
		return "any"
	}
}

// stringType maps OpenAPI string types (with format) to Go types.
func (sc *schemaConverter) stringType(schema *spec.Schema) string {
	switch schema.Format {
	case "date-time":
		sc.addImport("time", "")
		return "time.Time"
	case "date":
		sc.addImport("time", "")
		return "time.Time"
	default:
		return "string"
	}
}

// integerType maps OpenAPI integer types to Go types.
func (sc *schemaConverter) integerType(schema *spec.Schema) string {
	switch schema.Format {
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	default:
		return "int"
	}
}

// numberType maps OpenAPI number types to Go types.
func (sc *schemaConverter) numberType(schema *spec.Schema) string {
	switch schema.Format {
	case "float":
		return "float32"
	case "double":
		return "float64"
	default:
		return "float64"
	}
}

// objectType maps OpenAPI object types to Go types.
func (sc *schemaConverter) objectType(schema *spec.Schema) string {
	if schema.AdditionalProperties != nil {
		valueType := sc.goType(schema.AdditionalProperties, true)
		return "map[string]" + valueType
	}
	// Inline objects with properties become named structs elsewhere
	return "any"
}

// addImport registers an import.
func (sc *schemaConverter) addImport(path, alias string) {
	if path == "" {
		return
	}
	sc.imports[path] = alias
}

// sortedImports returns the collected imports sorted.
func (sc *schemaConverter) sortedImports() []ImportData {
	imports := make([]ImportData, 0, len(sc.imports))
	for path, alias := range sc.imports {
		imports = append(imports, ImportData{Path: path, Alias: alias})
	}
	sort.Slice(imports, func(i, j int) bool {
		return imports[i].Path < imports[j].Path
	})
	return imports
}

// extractRefName extracts the schema name from a $ref string.
// e.g., "#/components/schemas/Balance" → "Balance"
func extractRefName(ref string) string {
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	// Fallback: return last segment
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

// schemaToStruct converts a named OpenAPI schema with properties into a StructData.
// Supports allOf composition by merging properties from all sub-schemas.
func (sc *schemaConverter) schemaToStruct(name string, schema *spec.Schema) *StructData {
	if schema == nil {
		return nil
	}

	// Merge properties and required fields from allOf sub-schemas
	mergedProps := make(map[string]*spec.Schema)
	requiredSet := make(map[string]bool)

	// Collect from allOf first
	for _, sub := range schema.AllOf {
		if sub == nil {
			continue
		}
		resolved := sub
		// Resolve $ref to the actual schema from components/schemas
		if sub.Ref != "" && sc.schemas != nil {
			refName := extractRefName(sub.Ref)
			if refSchema, ok := sc.schemas[refName]; ok {
				resolved = refSchema
			}
		}
		maps.Copy(mergedProps, resolved.Properties)
		for _, r := range resolved.Required {
			requiredSet[r] = true
		}
	}

	// Then collect direct properties (override allOf if same name)
	maps.Copy(mergedProps, schema.Properties)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	if len(mergedProps) == 0 {
		return nil
	}

	var fields []FieldData
	// Sort property names for deterministic output
	propNames := sortedKeys(mergedProps)
	for _, propName := range propNames {
		propSchema := mergedProps[propName]
		isRequired := requiredSet[propName]

		goFieldType := sc.goType(propSchema, isRequired)

		jsonTag := propName
		if !isRequired {
			jsonTag += ",omitempty"
		}

		fields = append(fields, FieldData{
			Name:     toPascalCase(propName),
			Type:     goFieldType,
			JSONTag:  jsonTag,
			Comment:  propSchema.Description,
			Required: isRequired,
		})
	}

	return &StructData{
		Name:    name,
		Comment: schema.Description,
		Fields:  fields,
	}
}

// schemaToEnum converts an OpenAPI schema with enum values into an EnumData.
func (sc *schemaConverter) schemaToEnum(name string, schema *spec.Schema) *EnumData {
	if schema == nil || len(schema.Enum) == 0 {
		return nil
	}

	var values []EnumValueData
	for _, v := range schema.Enum {
		strVal := fmt.Sprintf("%v", v)
		constName := name + toPascalCase(strVal)
		values = append(values, EnumValueData{
			Name:  constName,
			Value: strVal,
		})
	}

	return &EnumData{
		Name:    name,
		Type:    "string", // All enums are string-based in our SDKs
		Comment: schema.Description,
		Values:  values,
	}
}
