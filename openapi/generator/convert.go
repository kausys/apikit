package generator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kausys/apikit/openapi/spec"
	"github.com/kausys/apikit/scanner"
)

// metaToInfo converts MetaInfo to spec.Info.
func (g *Generator) metaToInfo(meta *scanner.MetaInfo) *spec.Info {
	info := &spec.Info{
		Title:          meta.Title,
		Description:    meta.Description,
		TermsOfService: meta.TermsOfService,
		Version:        meta.Version,
	}

	if meta.Contact != nil {
		info.Contact = &spec.Contact{
			Name:  meta.Contact.Name,
			URL:   meta.Contact.URL,
			Email: meta.Contact.Email,
		}
	}

	if meta.License != nil {
		info.License = &spec.License{
			Name: meta.License.Name,
			URL:  meta.License.URL,
		}
	}

	return info
}

// applyMeta sets info, security schemes, and tags on the OpenAPI spec from metadata.
// If meta is nil, falls back to fallbackMeta. If both are nil, uses default info.
// When fallbackMeta is provided, its security schemes are inherited if meta has none.
func (g *Generator) applyMeta(openAPI *spec.OpenAPI, meta, fallbackMeta *scanner.MetaInfo) {
	effective := meta
	if effective == nil {
		effective = fallbackMeta
	}

	if effective == nil {
		openAPI.Info = &spec.Info{
			Title:   "API",
			Version: "1.0.0",
		}
		return
	}

	openAPI.Info = g.metaToInfo(effective)

	for name, scheme := range effective.SecuritySchemes {
		openAPI.Components.SecuritySchemes[name] = g.securitySchemeToSpec(scheme)
	}

	for _, tag := range effective.Tags {
		openAPI.Tags = append(openAPI.Tags, &spec.Tag{
			Name:        tag.Name,
			Description: tag.Description,
		})
	}

	// If the specific meta had no security schemes, inherit from fallback
	if meta != nil && len(openAPI.Components.SecuritySchemes) == 0 && fallbackMeta != nil {
		for name, scheme := range fallbackMeta.SecuritySchemes {
			openAPI.Components.SecuritySchemes[name] = g.securitySchemeToSpec(scheme)
		}
	}
}

// shortTypeName returns the unqualified type name.
// For "dto.Agent" returns "Agent", for "Agent" returns "Agent".
func shortTypeName(typeName string) string {
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		return typeName[idx+1:]
	}
	return typeName
}

// resolveModelRef resolves a Go type name to a model name in components/schemas.
// Returns the model name and true if found, or empty string and false otherwise.
func (g *Generator) resolveModelRef(typeName string) (string, bool) {
	// Check by direct name
	if _, ok := g.scanner.Structs[typeName]; ok {
		return typeName, true
	}
	// Check Go type → model mapping
	if modelName, ok := g.scanner.TypeToStruct[typeName]; ok {
		if _, exists := g.scanner.Structs[modelName]; exists {
			return modelName, true
		}
	}
	// Check by short (unqualified) name
	short := shortTypeName(typeName)
	if short != typeName {
		if _, ok := g.scanner.Structs[short]; ok {
			return short, true
		}
		if modelName, ok := g.scanner.TypeToStruct[short]; ok {
			if _, exists := g.scanner.Structs[modelName]; exists {
				return modelName, true
			}
		}
	}
	// Check external schemas from config file
	if GetExternalSchema(typeName) != nil {
		return typeName, true
	}
	return "", false
}

// isReferenceType checks if a type should be a $ref (model or enum).
func (g *Generator) isReferenceType(typeName string) bool {
	if _, ok := g.resolveModelRef(typeName); ok {
		return true
	}
	return g.scanner.GetEnumForType(typeName) != nil
}

// structToSchema converts StructInfo to spec.Schema.
func (g *Generator) structToSchema(s *scanner.StructInfo) *spec.Schema {
	// Handle oneOf/anyOf model schemas (pure composition, no type/properties)
	if s.IsOneOfModel {
		return g.compositionModelToSchema(s, s.OneOfOptions, s.OneOf)
	}
	if s.IsAnyOfModel {
		return g.compositionModelToSchema(s, s.AnyOfOptions, s.AnyOf)
	}

	// Handle non-struct underlying types (arrays, maps, primitives)
	switch s.UnderlyingKind {
	case scanner.KindArray:
		return &spec.Schema{
			Type:        spec.NewSchemaType(scanner.TypeArray),
			Description: s.Description,
			Items:       g.typeToSchema(s.ElementType),
		}
	case scanner.KindMap:
		return &spec.Schema{
			Type:                 spec.NewSchemaType(scanner.TypeObject),
			Description:          s.Description,
			AdditionalProperties: g.typeToSchema(s.ElementType),
		}
	case scanner.KindPrimitive:
		schema := &spec.Schema{
			Description: s.Description,
		}
		g.setSchemaType(schema, s.ElementType)
		return schema
	default:
		// KindStruct and other kinds fall through to struct handling below
	}

	schema := &spec.Schema{
		Type:        spec.NewSchemaType(scanner.TypeObject),
		Description: s.Description,
		Properties:  make(map[string]*spec.Schema),
	}

	var required []string

	for _, field := range s.Fields {
		propName := g.getPropertyName(field)
		if propName == "" || propName == "-" {
			continue
		}

		propSchema := g.fieldToSchema(field)
		schema.Properties[propName] = propSchema

		if field.Required || field.ExplicitRequired {
			required = append(required, propName)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}

	// Handle legacy composition (mark references)
	if len(s.AllOf) > 0 {
		for _, ref := range s.AllOf {
			g.markSchemaAsReferenced(ref)
			schema.AllOf = append(schema.AllOf, &spec.Schema{
				Ref: "#/components/schemas/" + ref,
			})
		}
	}
	if len(s.OneOf) > 0 {
		for _, ref := range s.OneOf {
			g.markSchemaAsReferenced(ref)
			schema.OneOf = append(schema.OneOf, &spec.Schema{
				Ref: "#/components/schemas/" + ref,
			})
		}
	}
	if len(s.AnyOf) > 0 {
		for _, ref := range s.AnyOf {
			g.markSchemaAsReferenced(ref)
			schema.AnyOf = append(schema.AnyOf, &spec.Schema{
				Ref: "#/components/schemas/" + ref,
			})
		}
	}

	return schema
}

// typeToSchema converts a Go type name to a schema.
func (g *Generator) typeToSchema(typeName string) *spec.Schema {
	// Check if it resolves to a model
	if modelName, ok := g.resolveModelRef(typeName); ok {
		g.markSchemaAsReferenced(modelName)
		return &spec.Schema{Ref: "#/components/schemas/" + modelName}
	}

	// Check if it's an enum (using alias resolution)
	if enumInfo := g.scanner.GetEnumForType(typeName); enumInfo != nil {
		if g.config.EnumRefs {
			g.markSchemaAsReferenced(enumInfo.TypeName)
			return &spec.Schema{Ref: "#/components/schemas/" + enumInfo.TypeName}
		}
		return g.createInlineEnumSchema(enumInfo)
	}

	schema := &spec.Schema{}
	g.setSchemaType(schema, typeName)
	return schema
}

// isKnownGoType returns true for built-in Go types that map to JSON Schema primitives.
func isKnownGoType(t string) bool {
	switch t {
	case "string", "bool", "any", "interface{}",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return true
	}
	return false
}

// setSchemaType sets the type and format for a schema based on Go type.
// qualifiedType is optional — if provided, it takes precedence for custom type lookup.
func (g *Generator) setSchemaType(schema *spec.Schema, goType string, qualifiedType ...string) {
	// Check for registered custom types first (try qualified name, then short name)
	var typeInfo *TypeInfo
	for _, qt := range qualifiedType {
		if qt != "" {
			typeInfo = GetCustomType(qt)
			break
		}
	}
	if typeInfo == nil {
		typeInfo = GetCustomType(goType)
	}
	if typeInfo != nil {
		schema.Type = spec.NewSchemaType(typeInfo.Type)
		schema.Format = typeInfo.Format
		if typeInfo.Example != nil && len(schema.Examples) == 0 {
			schema.Examples = []any{typeInfo.Example}
		}
		if typeInfo.Default != nil && schema.Default == nil {
			schema.Default = typeInfo.Default
		}
		for k, v := range typeInfo.Validations {
			if schema.Format == "" && k == "format" {
				schema.Format = v
			}
		}
		return
	}

	switch goType {
	case "string":
		schema.Type = spec.NewSchemaType(scanner.TypeString)
	case "int", "int8", "int16", "int32":
		schema.Type = spec.NewSchemaType(scanner.TypeInteger)
		schema.Format = scanner.FormatInt32
	case "int64":
		schema.Type = spec.NewSchemaType(scanner.TypeInteger)
		schema.Format = scanner.FormatInt64
	case "uint", "uint8", "uint16", "uint32":
		schema.Type = spec.NewSchemaType(scanner.TypeInteger)
		schema.Format = scanner.FormatInt32
	case "uint64":
		schema.Type = spec.NewSchemaType(scanner.TypeInteger)
		schema.Format = scanner.FormatInt64
	case "float32":
		schema.Type = spec.NewSchemaType(scanner.TypeNumber)
		schema.Format = scanner.FormatFloat
	case "float64":
		schema.Type = spec.NewSchemaType(scanner.TypeNumber)
		schema.Format = scanner.FormatDouble
	case "bool":
		schema.Type = spec.NewSchemaType(scanner.TypeBoolean)
	case "any", "interface{}":
		schema.Type = spec.NewSchemaType(scanner.TypeObject)
	default:
		// Check for package-qualified types
		short := shortTypeName(goType)
		if short != goType {
			if _, ok := g.scanner.Structs[short]; ok {
				schema.Ref = "#/components/schemas/" + short
				return
			}
		}
		if !isKnownGoType(goType) {
			fmt.Printf("warning: unknown type %q — add it to .openapi.yaml schemas or register as a custom_type; falling back to string\n", goType)
		}
		schema.Type = spec.NewSchemaType(scanner.TypeString)
	}
}

// castToSchemaType converts a string value to the appropriate Go type
// based on the OpenAPI schema type, so YAML serialization produces the correct type.
func castToSchemaType(value string, schemaType spec.SchemaType) any {
	switch schemaType.Value() {
	case scanner.TypeInteger:
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			return v
		}
	case scanner.TypeNumber:
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		}
	case scanner.TypeBoolean:
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	}
	return value
}
