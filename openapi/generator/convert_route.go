package generator

import (
	"strings"

	"github.com/kausys/apikit/openapi/spec"
	"github.com/kausys/apikit/scanner"
)

// routeToOperation converts RouteInfo to spec.Operation.
func (g *Generator) routeToOperation(r *scanner.RouteInfo) *spec.Operation {
	responses := &spec.Responses{
		StatusCodes: make(map[string]*spec.Response),
	}

	// Add responses
	for _, resp := range r.Responses {
		response := &spec.Response{
			Description: resp.Description,
		}

		if resp.Type != "" {
			schema := g.typeToSchema(resp.Type)
			if resp.IsArray {
				schema = &spec.Schema{
					Type:  spec.NewSchemaType(scanner.TypeArray),
					Items: schema,
				}
			}
			response.Content = map[string]*spec.MediaType{
				"application/json": {
					Schema: schema,
				},
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

	// Add security
	if len(r.Security) > 0 {
		for _, scheme := range r.Security {
			op.Security = append(op.Security, &spec.SecurityRequirement{
				Requirements: map[string][]string{
					scheme: {},
				},
			})
		}
	}

	return op
}

// getOperationParameters finds and converts parameters for an operation.
func (g *Generator) getOperationParameters(r *scanner.RouteInfo) ([]*spec.Parameter, *spec.RequestBody) {
	// Look for a struct marked as swagger:parameters with matching operationID
	paramStruct, ok := g.scanner.Structs[r.OperationID]
	if !ok || !paramStruct.IsParameter {
		return nil, nil
	}

	var params []*spec.Parameter
	var requestBody *spec.RequestBody
	ignoredParams := make(map[string]bool)
	for _, ignored := range r.IgnoredParameters {
		ignoredParams[ignored] = true
	}

	for _, field := range paramStruct.Fields {
		// Skip ignored parameters
		paramName := g.getPropertyName(field)
		if ignoredParams[paramName] {
			continue
		}

		// Handle request body (in:body)
		if field.IsRequestBody || field.In == "body" {
			requestBody = g.fieldToRequestBody(field, r.Consumes)
			continue
		}

		param := g.fieldToParameter(field, r.Path)
		if param != nil {
			params = append(params, param)
		}
	}

	return params, requestBody
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

	switch strings.ToUpper(r.Method) {
	case "GET":
		pathItem.Get = op
	case "POST":
		pathItem.Post = op
	case "PUT":
		pathItem.Put = op
	case "DELETE":
		pathItem.Delete = op
	case "PATCH":
		pathItem.Patch = op
	case "HEAD":
		pathItem.Head = op
	case "OPTIONS":
		pathItem.Options = op
	}
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
