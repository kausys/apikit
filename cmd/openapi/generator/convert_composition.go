package generator

import (
	"github.com/kausys/apikit/cmd/openapi/spec"
	"github.com/kausys/apikit/cmd/scanner"
)

// compositionModelToSchema converts a swagger:oneOf or swagger:anyOf model to a composition schema.
func (g *Generator) compositionModelToSchema(s *scanner.StructInfo, options, refs []string) *spec.Schema {
	schema := &spec.Schema{
		Description: s.Description,
	}

	schemas := make([]*spec.Schema, 0, len(options)+len(refs))

	// Add options from embedded fields marked with swagger:oneOfOption/anyOfOption
	for _, typeName := range options {
		refName := g.resolveSchemaRef(typeName)
		g.markSchemaAsReferenced(refName)
		schemas = append(schemas, &spec.Schema{
			Ref: "#/components/schemas/" + refName,
		})
	}

	// Also support legacy inline oneOf:/anyOf: directive
	for _, ref := range refs {
		g.markSchemaAsReferenced(ref)
		schemas = append(schemas, &spec.Schema{
			Ref: "#/components/schemas/" + ref,
		})
	}

	if s.IsOneOfModel {
		schema.OneOf = schemas
	} else {
		schema.AnyOf = schemas
	}

	// Add discriminator if present
	if s.Discriminator != nil {
		schema.Discriminator = g.discriminatorToSpec(s.Discriminator)
	}

	return schema
}

// resolveSchemaRef resolves a type name to a schema reference name.
func (g *Generator) resolveSchemaRef(typeName string) string {
	// Check if there's a type mapping (Go type name -> model name)
	if modelName, ok := g.scanner.TypeToStruct[typeName]; ok {
		return modelName
	}

	// Check for package-qualified types (e.g., "dto.Agent")
	short := shortTypeName(typeName)
	if short != typeName {
		if modelName, ok := g.scanner.TypeToStruct[short]; ok {
			return modelName
		}
		if _, ok := g.scanner.Structs[short]; ok {
			return short
		}
	}

	return typeName
}

// discriminatorToSpec converts DiscriminatorInfo to spec.Discriminator.
func (g *Generator) discriminatorToSpec(d *scanner.DiscriminatorInfo) *spec.Discriminator {
	discriminator := &spec.Discriminator{
		PropertyName: d.PropertyName,
	}

	if len(d.Mapping) > 0 {
		discriminator.Mapping = make(map[string]string)
		for key, schemaName := range d.Mapping {
			// Resolve schema name to full reference
			refName := g.resolveSchemaRef(schemaName)
			discriminator.Mapping[key] = "#/components/schemas/" + refName
		}
	}

	return discriminator
}
