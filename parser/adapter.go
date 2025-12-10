package parser

import (
	"fmt"
	"go/ast"
	"strings"

	coreast "github.com/kausys/apikit/ast"
)

// ExtractFromGeneric extracts APIKit-specific information from generic parse result
// This adapter filters for apikit:handler and apikit:dto directives
func ExtractFromGeneric(generic *coreast.ParseResult) (*ParseResult, error) {
	result := &ParseResult{
		Handlers: []Handler{},
		Structs:  make(map[string]*Struct),
		Source: Source{
			Filename: generic.Filename,
			Package:  generic.Package,
		},
		Warnings: []string{},
	}

	// Convert all structs first (needed for handler lookup)
	for name, genericStruct := range generic.Structs {
		result.Structs[name] = convertStruct(genericStruct)
	}

	// Extract handlers (only functions with apikit:handler)
	for _, fn := range generic.Functions {
		if hasDirective(fn.Doc, "apikit:handler") {
			handler := extractHandler(fn, result, generic)
			if handler != nil {
				result.Handlers = append(result.Handlers, *handler)
			}
		}
	}

	return result, nil
}

// convertStruct converts a generic struct to APIKit struct
func convertStruct(generic *coreast.Struct) *Struct {
	s := &Struct{
		Name:   generic.Name,
		Fields: []Field{},
		IsDTO:  hasDirective(generic.Doc, "apikit:dto"),
	}

	for _, genericField := range generic.Fields {
		field := convertField(genericField)
		s.Fields = append(s.Fields, field)
	}

	return s
}

// convertField converts a generic field to APIKit field
func convertField(generic *coreast.Field) Field {
	f := Field{
		Name:       generic.Name,
		Type:       generic.Type,
		StructTag:  generic.Tag,
		IsPointer:  generic.IsPointer,
		IsSlice:    generic.IsSlice,
		SliceType:  generic.SliceType,
		IsEmbedded: generic.IsEmbedded,
	}

	// Extract "// in:xxx" and "// default:xxx" comments
	if generic.Comment != nil {
		for _, comment := range generic.Comment.List {
			if source, name := extractInComment(comment.Text); source != "" {
				f.InComment = source
				f.InCommentName = name
				if source == "body" {
					f.IsBody = true
				}
			}
		}
	}
	if generic.Doc != nil {
		for _, comment := range generic.Doc.List {
			// Only extract if not found in Comment
			if f.InComment == "" {
				if source, name := extractInComment(comment.Text); source != "" {
					f.InComment = source
					f.InCommentName = name
					if source == "body" {
						f.IsBody = true
					}
				}
			}
		}
	}

	// Check for special field types
	f.IsRawBody = generic.Type == "[]byte" && (generic.Name == "RawBody" || generic.Name == "Raw")

	// http.ResponseWriter aliases
	f.IsResponseWriter = (generic.Name == "ResponseWriter" ||
		generic.Name == "Response" ||
		generic.Name == "Writer" ||
		generic.Name == "Res" ||
		generic.Name == "W") &&
		generic.Type == "http.ResponseWriter"

	// *http.Request aliases
	f.IsRequest = (generic.Name == "Request" ||
		generic.Name == "Req" ||
		generic.Name == "R") &&
		generic.Type == "*http.Request"

	return f
}

// extractHandler extracts handler information from a generic function
func extractHandler(fn *coreast.Function, result *ParseResult, generic *coreast.ParseResult) *Handler {
	// Validate handler signature
	if !isValidHandlerSignature(fn) {
		warning := fmt.Sprintf("%s: function %s has apikit:handler comment but invalid signature",
			fn.Pos, fn.Name)
		result.Warnings = append(result.Warnings, warning)
		return nil
	}

	h := &Handler{
		Name:    fn.Name,
		Package: generic.Package,
		Pos:     fn.Pos,
	}

	// Handle receiver for methods
	if fn.Receiver != "" {
		h.Receiver = fn.Receiver
	}

	// Get parameter type (second parameter)
	if len(fn.Params) < 2 {
		warning := fmt.Sprintf("%s: function %s has insufficient parameters", fn.Pos, fn.Name)
		result.Warnings = append(result.Warnings, warning)
		return nil
	}
	h.ParamType = fn.Params[1].Type

	// Check for optional http.ResponseWriter and *http.Request parameters
	if len(fn.Params) > 2 {
		for i := 2; i < len(fn.Params); i++ {
			if fn.Params[i].Type == "http.ResponseWriter" {
				h.HasResponseWriter = true
			} else if fn.Params[i].Type == "*http.Request" {
				h.HasRequest = true
			}
		}
	}

	// Look up struct info
	structName := getTypeName(fn.Params[1].Type)
	if s, ok := result.Structs[structName]; ok {
		h.Struct = s
	}

	// Get return type (first return value)
	if len(fn.Results) < 1 {
		warning := fmt.Sprintf("%s: function %s has no return values", fn.Pos, fn.Name)
		result.Warnings = append(result.Warnings, warning)
		return nil
	}
	h.ReturnType = fn.Results[0].Type

	return h
}

// hasDirective checks if comments contain a specific directive
func hasDirective(comments *ast.CommentGroup, directive string) bool {
	if comments == nil {
		return false
	}
	for _, comment := range comments.List {
		if strings.Contains(comment.Text, directive) {
			return true
		}
	}
	return false
}

// isValidHandlerSignature checks if function has the correct signature:
// func(context.Context, T) (R, error)
// func(context.Context, T, http.ResponseWriter) (R, error)
// func(context.Context, T, *http.Request) (R, error)
// func(context.Context, T, http.ResponseWriter, *http.Request) (R, error)
func isValidHandlerSignature(fn *coreast.Function) bool {
	// Check parameters: minimum (context.Context, T)
	if len(fn.Params) < 2 || len(fn.Params) > 4 {
		return false
	}

	// First param must be context.Context
	if fn.Params[0].Type != "context.Context" {
		return false
	}

	// Optional third and fourth params can be http.ResponseWriter or *http.Request
	if len(fn.Params) > 2 {
		for i := 2; i < len(fn.Params); i++ {
			paramType := fn.Params[i].Type
			if paramType != "http.ResponseWriter" && paramType != "*http.Request" {
				return false
			}
		}
	}

	// Check results: (T, error)
	if len(fn.Results) != 2 {
		return false
	}

	// Second result must be error
	if fn.Results[1].Type != "error" {
		return false
	}

	return true
}

// getTypeName extracts just the type name without package prefix or pointer
func getTypeName(typeStr string) string {
	// Remove pointer
	typeStr = strings.TrimPrefix(typeStr, "*")

	// Remove package prefix
	if idx := strings.LastIndex(typeStr, "."); idx != -1 {
		return typeStr[idx+1:]
	}

	return typeStr
}
