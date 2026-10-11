package generator

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kausys/apikit/cmd/openapi/spec"
	"github.com/kausys/apikit/cmd/scanner"
)

// routeToOperation converts RouteInfo to spec.Operation.
func (g *Generator) routeToOperation(r *scanner.RouteInfo) *spec.Operation {
	responses := &spec.Responses{
		StatusCodes: make(map[string]*spec.Response),
	}

	// What a response body is written as: the route's Produces, or JSON.
	produces := r.Produces
	if len(produces) == 0 {
		produces = []string{"application/json"}
	}

	// Add responses
	for _, resp := range r.Responses {
		// A response must have a description; the status's own name is the
		// least a reader is owed when the route gives none.
		description := resp.Description
		if description == "" {
			description = statusDescription(resp.StatusCode)
		}
		response := &spec.Response{
			Description: description,
		}

		if resp.Type != "" {
			schema := g.typeToSchema(resp.Type)
			if resp.IsArray {
				schema = &spec.Schema{
					Type:  spec.NewSchemaType(scanner.TypeArray),
					Items: schema,
				}
			}
			response.Content = make(map[string]*spec.MediaType, len(produces))
			for _, contentType := range produces {
				response.Content[contentType] = &spec.MediaType{Schema: schema}
			}
		}

		if resp.StatusCode == "default" {
			responses.Default = response
		} else {
			responses.StatusCodes[resp.StatusCode] = response
		}
	}

	op := &spec.Operation{
		OperationID: r.OperationID,
		Summary:     r.Summary,
		Description: r.Description,
		Tags:        r.Tags,
		Deprecated:  r.Deprecated,
		Responses:   responses,
	}

	// Add parameters and request body from swagger:parameters struct matching operationID
	params, requestBody := g.getOperationParameters(r)
	if len(params) > 0 {
		op.Parameters = params
	}

	// Only add requestBody for methods that typically carry a body.
	// GET and HEAD never have a body. DELETE may carry one in some APIs.
	method := strings.ToUpper(r.Method)
	if requestBody != nil && method != "GET" && method != "HEAD" {
		op.RequestBody = requestBody
	}

	// Add security: each scheme is an alternative, with the scopes it needs.
	for _, scheme := range r.Security {
		scopes := r.SecurityScopes[scheme]
		if scopes == nil {
			scopes = []string{}
		}
		op.Security = append(op.Security, &spec.SecurityRequirement{
			Requirements: map[string][]string{scheme: scopes},
		})
	}

	return op
}

// getOperationParameters finds and converts parameters for an operation.
//
// They come from the swagger:parameters struct named after the operation, or,
// when there is none and the route is declared on a handler, from the request
// type the handler takes — read with the tags the generated wrapper binds, so
// what is documented is what is parsed.
func (g *Generator) getOperationParameters(r *scanner.RouteInfo) ([]*spec.Parameter, *spec.RequestBody) {
	paramStruct, ok := g.scanner.Structs[r.OperationID]
	if !ok || !paramStruct.IsParameter {
		paramStruct = g.scanner.RequestStruct(r)
		if paramStruct == nil {
			return nil, nil
		}
	}

	var params []*spec.Parameter
	var requestBody *spec.RequestBody
	ignoredParams := make(map[string]bool)
	for _, ignored := range r.IgnoredParameters {
		ignoredParams[ignored] = true
	}

	var formFields []*scanner.FieldInfo
	for _, field := range paramStruct.Fields {
		// Skip ignored parameters
		paramName := g.parameterName(field)
		if ignoredParams[paramName] {
			continue
		}

		// A form field is one part of a multipart body, not the body.
		if field.FormPart {
			formFields = append(formFields, field)
			continue
		}

		// Handle request body (in:body)
		if field.IsRequestBody || field.In == "body" {
			// Its type needs no swagger:model to be documented.
			if !field.IsInlineStruct {
				if model := g.scanner.EnsureModel(field); model != "" {
					field.Type = model
				}
			}
			requestBody = g.fieldToRequestBody(field, r.Consumes)
			continue
		}

		param := g.fieldToParameter(field, r.Path)
		if param != nil {
			params = append(params, param)
		}
	}

	if len(formFields) > 0 && requestBody == nil {
		requestBody = g.formRequestBody(formFields, r.Consumes)
	}

	return params, requestBody
}

// formRequestBody documents form fields as the parts of one body:
// multipart/form-data unless the route says otherwise, a file field as binary.
func (g *Generator) formRequestBody(fields []*scanner.FieldInfo, consumes []string) *spec.RequestBody {
	schema := &spec.Schema{
		Type:       spec.NewSchemaType(scanner.TypeObject),
		Properties: make(map[string]*spec.Schema),
	}
	encoding := make(map[string]*spec.Encoding)
	required := false
	for _, field := range fields {
		name := g.parameterName(field)
		if isFileType(field.Type) {
			fileSchema := &spec.Schema{Type: spec.NewSchemaType(scanner.TypeString), ContentMediaType: "application/octet-stream"}
			if field.IsArray {
				fileSchema = &spec.Schema{Type: spec.NewSchemaType(scanner.TypeArray), Items: fileSchema}
			}
			fileSchema.Description = field.Description
			schema.Properties[name] = fileSchema
			encoding[name] = &spec.Encoding{ContentType: "application/octet-stream"}
		} else {
			schema.Properties[name] = g.fieldToSchema(field)
		}
		if field.Required {
			schema.Required = append(schema.Required, name)
			required = true
		}
	}
	if len(encoding) == 0 {
		encoding = nil
	}

	contentTypes := consumes
	if len(contentTypes) == 0 {
		contentTypes = []string{scanner.ContentTypeMultipart}
	}
	content := make(map[string]*spec.MediaType, len(contentTypes))
	for _, contentType := range contentTypes {
		mediaType := &spec.MediaType{Schema: schema}
		if contentType == scanner.ContentTypeMultipart {
			mediaType.Encoding = encoding
		}
		content[contentType] = mediaType
	}
	return &spec.RequestBody{Required: required, Content: content}
}

// isFileType reports an uploaded file: *multipart.FileHeader.
func isFileType(goType string) bool {
	return goType == "multipart.FileHeader"
}

// statusDescription names a response status: "Not Found" for 404, "Client
// Error" for 4XX, "Default" for default.
func statusDescription(code string) string {
	if n, err := strconv.Atoi(code); err == nil {
		if text := http.StatusText(n); text != "" {
			return text
		}
	}
	switch strings.ToUpper(code) {
	case "1XX":
		return "Informational"
	case "2XX":
		return "Success"
	case "3XX":
		return "Redirection"
	case "4XX":
		return "Client Error"
	case "5XX":
		return "Server Error"
	}
	return "Default"
}

// addRoute adds a route to the OpenAPI spec.
func (g *Generator) addRoute(openAPI *spec.OpenAPI, r *scanner.RouteInfo) {
	if openAPI.Paths == nil {
		openAPI.Paths = &spec.Paths{
			PathItems: make(map[string]*spec.PathItem),
		}
	}

	pathItem, exists := openAPI.Paths.PathItems[r.Path]
	if !exists {
		pathItem = &spec.PathItem{}
		openAPI.Paths.PathItems[r.Path] = pathItem
	}

	op := g.routeToOperation(r)

	var slot **spec.Operation
	switch strings.ToUpper(r.Method) {
	case "GET":
		slot = &pathItem.Get
	case "POST":
		slot = &pathItem.Post
	case "PUT":
		slot = &pathItem.Put
	case "DELETE":
		slot = &pathItem.Delete
	case "PATCH":
		slot = &pathItem.Patch
	case "HEAD":
		slot = &pathItem.Head
	case "OPTIONS":
		slot = &pathItem.Options
	default:
		return
	}
	// Two routes on one method and path used to keep whichever came last,
	// in map order, and lose the other without a word.
	if *slot != nil {
		g.problems = append(g.problems, fmt.Sprintf("%s %s: declared by both %s and %s (%s)",
			strings.ToUpper(r.Method), r.Path, (*slot).OperationID, op.OperationID, r.SourceFile))
		if (*slot).OperationID < op.OperationID {
			return
		}
	}
	*slot = op
}

// securitySchemeToSpec converts SecuritySchemeInfo to spec.SecurityScheme.
func (g *Generator) securitySchemeToSpec(s *scanner.SecuritySchemeInfo) *spec.SecurityScheme {
	return &spec.SecurityScheme{
		Type:         s.Type,
		Description:  s.Description,
		Name:         s.Name,
		In:           s.In,
		Scheme:       s.Scheme,
		BearerFormat: s.BearerFormat,
	}
}
