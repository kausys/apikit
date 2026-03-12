package generator

import (
	"github.com/kausys/apikit/cmd/openapi/spec"
	"github.com/kausys/apikit/cmd/scanner"
)

// fieldToRequestBody converts a FieldInfo to spec.RequestBody.
// consumes specifies the content types from the route's Consumes directive.
func (g *Generator) fieldToRequestBody(f *scanner.FieldInfo, consumes []string) *spec.RequestBody {
	var schema *spec.Schema

	// Handle inline structs
	if f.IsInlineStruct && f.InlineStruct != nil {
		schema = g.inlineStructToSchema(f.InlineStruct)
	} else {
		schema = g.typeToSchema(f.Type)
	}

	// Determine content types to use
	contentTypes := consumes
	if len(contentTypes) == 0 {
		// Default to application/json if no consumes specified
		contentTypes = []string{"application/json"}
	}

	// Build content map for each content type
	content := make(map[string]*spec.MediaType)
	for _, contentType := range contentTypes {
		mediaType := &spec.MediaType{}

		// Handle multipart/form-data specially
		if contentType == scanner.ContentTypeMultipart {
			mediaType.Schema = g.createMultipartSchema(f)
			mediaType.Encoding = g.createMultipartEncoding(f)
		} else {
			// For other content types (JSON, XML, etc.), use the schema as-is
			mediaType.Schema = schema
		}

		content[contentType] = mediaType
	}

	return &spec.RequestBody{
		Description: f.Description,
		Required:    f.Required,
		Content:     content,
	}
}

// inlineStructToSchema converts an inline StructInfo to spec.Schema.
func (g *Generator) inlineStructToSchema(s *scanner.StructInfo) *spec.Schema {
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

	return schema
}

// createMultipartSchema creates a schema for multipart/form-data requests.
// For multipart, the schema must be type: object with properties for each part.
func (g *Generator) createMultipartSchema(f *scanner.FieldInfo) *spec.Schema {
	// If the field is an inline struct, convert it
	if f.IsInlineStruct && f.InlineStruct != nil {
		return g.inlineStructToSchema(f.InlineStruct)
	}

	// If it's a reference to a model, use it as-is
	return g.typeToSchema(f.Type)
}

// createMultipartEncoding creates encoding information for multipart/form-data.
// This is used to specify content types for file upload fields.
func (g *Generator) createMultipartEncoding(f *scanner.FieldInfo) map[string]*spec.Encoding {
	// Only create encoding if we have an inline struct with fields
	if !f.IsInlineStruct || f.InlineStruct == nil {
		return nil
	}

	encoding := make(map[string]*spec.Encoding)

	// Check each field in the struct for file upload fields (format: binary)
	for _, field := range f.InlineStruct.Fields {
		propName := g.getPropertyName(field)
		if propName == "" || propName == "-" {
			continue
		}

		// Check if this field is a file upload (format: binary)
		if format, ok := field.Validations["format"]; ok && format == scanner.FormatBinary {
			// Add encoding for this field
			encoding[propName] = &spec.Encoding{
				ContentType: "application/octet-stream",
			}
		}
	}

	if len(encoding) == 0 {
		return nil
	}

	return encoding
}
