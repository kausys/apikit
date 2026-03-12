package scanner

import (
	"go/ast"
	"strings"
)

// processRoutes processes swagger:route directives.
func (s *Scanner) processRoutes(filePath string, file *ast.File) error {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Doc == nil {
			continue
		}

		if !hasDirective(funcDecl.Doc, RouteDirective) {
			continue
		}

		routeValue := extractDirectiveValue(funcDecl.Doc, RouteDirective)
		method, path, tags, operationID := parseRouteDirective(routeValue)

		if method == "" || path == "" || operationID == "" {
			continue
		}

		route := &RouteInfo{
			Method:            method,
			Path:              path,
			Tags:              tags,
			OperationID:       operationID,
			Summary:           extractDirectiveValue(funcDecl.Doc, SummaryFieldDirective),
			Description:       extractRouteDescription(funcDecl.Doc),
			Deprecated:        hasDirective(funcDecl.Doc, DeprecatedFieldDirective),
			Responses:         []*ResponseInfo{},
			Security:          []string{},
			Consumes:          []string{},
			Produces:          []string{},
			IgnoredParameters: []string{},
			SourceFile:        filePath,
			Specs:             extractSpecs(funcDecl.Doc),
		}

		extractResponses(route, funcDecl.Doc)
		extractSecurity(route, funcDecl.Doc)
		extractConsumes(route, funcDecl.Doc)
		extractProduces(route, funcDecl.Doc)
		extractIgnoredParameters(route, funcDecl.Doc)

		s.Routes[operationID] = route
		s.RouteSources[operationID] = filePath
	}
	return nil
}

// parseRouteDirective parses: METHOD /path tag1 tag2 operationID
func parseRouteDirective(value string) (string, string, []string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", nil, ""
	}

	tokens := tokenizeWithQuotes(value)
	if len(tokens) < 3 {
		return "", "", nil, ""
	}

	method := strings.ToUpper(tokens[0])
	path := tokens[1]
	operationID := tokens[len(tokens)-1]

	// Validate method
	validMethods := map[string]bool{
		MethodGet: true, MethodPost: true, MethodPut: true,
		MethodDelete: true, MethodPatch: true, MethodHead: true, MethodOptions: true,
	}
	if !validMethods[method] {
		return "", "", nil, ""
	}

	// Validate path
	if !strings.HasPrefix(path, "/") {
		return "", "", nil, ""
	}

	// Extract tags (everything between path and operationID)
	var tags []string
	if len(tokens) > 3 {
		tags = tokens[2 : len(tokens)-1]
	}

	return method, path, tags, operationID
}

// tokenizeWithQuotes splits a string by spaces but respects quoted strings.
func tokenizeWithQuotes(s string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := range len(s) {
		c := s[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
				tokens = append(tokens, current.String())
				current.Reset()
			} else {
				current.WriteByte(c)
			}
		} else {
			switch c {
			case '\'', '"':
				inQuote = true
				quoteChar = c
			case ' ', '\t':
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
			default:
				current.WriteByte(c)
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// extractRouteDescription extracts multiline description from route comments.
// Description starts with "description:" directive and continues until another directive.
func extractRouteDescription(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}

	comments := trimComments(doc)
	inDescription := false
	var descriptionLines []string

	// Directives that end the description section
	endDirectives := []string{
		SwaggerPrefix, SummaryFieldDirective, SecurityDirective,
		ResponsesDirective, ConsumesDirective, ProducesDirective,
		ParametersDirective, IgnoredParametersDirective, DeprecatedFieldDirective,
	}

	for _, comment := range comments {
		// Check if we're starting the description section
		if strings.HasPrefix(comment, DescriptionFieldDirective) {
			inDescription = true
			// Extract text after "description:" on the same line
			descText := strings.TrimSpace(strings.TrimPrefix(comment, DescriptionFieldDirective))
			if descText != "" {
				descriptionLines = append(descriptionLines, descText)
			}
			continue
		}

		// If we're not in description section, skip
		if !inDescription {
			continue
		}

		// Check if we've hit another known directive (end of description section)
		isEndDirective := false
		for _, directive := range endDirectives {
			if strings.HasPrefix(comment, directive) {
				isEndDirective = true
				break
			}
		}

		if isEndDirective {
			break
		}

		// Add line to description (including empty lines for formatting)
		descriptionLines = append(descriptionLines, comment)
	}

	// Join lines with newlines and trim trailing whitespace
	return strings.TrimSpace(strings.Join(descriptionLines, "\n"))
}

// extractResponses parses the Responses: section.
func extractResponses(route *RouteInfo, doc *ast.CommentGroup) {
	lines := extractSectionLines(doc, ResponsesDirective)
	for _, line := range lines {
		if !strings.HasPrefix(line, DashPrefix) {
			continue
		}
		line = strings.TrimPrefix(line, DashPrefix)
		line = strings.TrimSpace(line)

		resp := parseResponseLine(line)
		if resp != nil {
			route.Responses = append(route.Responses, resp)
		}
	}
}

// parseResponseLine parses a single response line.
// Format: STATUS: Type description:Description text
func parseResponseLine(line string) *ResponseInfo {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 1 {
		return nil
	}

	resp := &ResponseInfo{
		StatusCode: strings.TrimSpace(parts[0]),
	}

	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		return resp
	}

	rest := strings.TrimSpace(parts[1])

	// Check for description-only response
	if after, found := strings.CutPrefix(rest, DescriptionFieldDirective); found {
		resp.Description = strings.TrimSpace(after)
		return resp
	}

	// Split type from description
	typeParts := strings.SplitN(rest, " ", 2)
	typeName := typeParts[0]

	// Check for array type
	if strings.HasPrefix(typeName, "[]") {
		resp.IsArray = true
		typeName = typeName[2:]
	}

	// Check for map type
	if strings.HasPrefix(typeName, "map[") {
		resp.IsMap = true
		// Extract map[Key]Value
		if idx := strings.Index(typeName, "]"); idx > 4 {
			resp.MapKeyType = typeName[4:idx]
			typeName = typeName[idx+1:]
		}
	}

	resp.Type = typeName

	// Extract description
	if len(typeParts) > 1 {
		desc := typeParts[1]
		if after, ok := strings.CutPrefix(desc, DescriptionFieldDirective); ok {
			desc = after
		}
		resp.Description = strings.TrimSpace(desc)
	}

	return resp
}

// extractSecurity parses the Security: section.
func extractSecurity(route *RouteInfo, doc *ast.CommentGroup) {
	lines := extractSectionLines(doc, SecurityDirective)
	for _, line := range lines {
		if after, found := strings.CutPrefix(line, DashPrefix); found {
			scheme := strings.TrimSpace(after)
			if scheme != "" {
				route.Security = append(route.Security, scheme)
			}
		}
	}
}

// extractConsumes parses the Consumes: section.
func extractConsumes(route *RouteInfo, doc *ast.CommentGroup) {
	lines := extractSectionLines(doc, ConsumesDirective)
	for _, line := range lines {
		if after, found := strings.CutPrefix(line, DashPrefix); found {
			contentType := strings.TrimSpace(after)
			if contentType != "" {
				route.Consumes = append(route.Consumes, contentType)
			}
		}
	}
}

// extractProduces parses the Produces: section.
func extractProduces(route *RouteInfo, doc *ast.CommentGroup) {
	lines := extractSectionLines(doc, ProducesDirective)
	for _, line := range lines {
		if after, found := strings.CutPrefix(line, DashPrefix); found {
			contentType := strings.TrimSpace(after)
			if contentType != "" {
				route.Produces = append(route.Produces, contentType)
			}
		}
	}
}

// extractIgnoredParameters parses the IgnoredParameters: section.
func extractIgnoredParameters(route *RouteInfo, doc *ast.CommentGroup) {
	lines := extractSectionLines(doc, IgnoredParametersDirective)
	for _, line := range lines {
		if after, found := strings.CutPrefix(line, DashPrefix); found {
			param := strings.TrimSpace(after)
			if param != "" {
				route.IgnoredParameters = append(route.IgnoredParameters, param)
			}
		}
	}
}
