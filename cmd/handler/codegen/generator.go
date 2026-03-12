// Package codegen generates Go code from parsed handler information using extractors
package codegen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"log"
	"reflect"
	"slices"
	"strings"
	"text/template"

	_ "embed"

	"github.com/kausys/apikit/cmd/handler/extractors"
	"github.com/kausys/apikit/cmd/handler/parser"

	"golang.org/x/tools/imports"
)

//go:embed templates/http_handler.tmpl
var httpHandlerTemplate string

// Generator generates wrapper code for handlers using http.Handler
type Generator struct {
	tmpl      *template.Template
	extractor extractors.FrameworkExtractor
	framework string
}

// New creates a new code generator for http.Handler (default)
func New() (*Generator, error) {
	return NewWithFramework("http")
}

// NewWithFramework creates a new code generator for the specified framework
// Supported frameworks: http (default), fiber, gin, echo
func NewWithFramework(framework string) (*Generator, error) {
	tmpl, err := template.New("handler").Funcs(templateFuncs()).Parse(httpHandlerTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	return &Generator{
		tmpl:      tmpl,
		extractor: extractors.NewHTTPExtractor(),
		framework: framework,
	}, nil
}

// TemplateData holds data for template execution
type TemplateData struct {
	PackageName string
	Imports     []string
	Handlers    []HandlerData
	Framework   string
}

// HandlerData holds data for a single handler
type HandlerData struct {
	Name              string
	WrapperName       string
	ParseFuncName     string
	ParamType         string
	ReturnType        string
	HasExtractionCode bool
	ExtractionCode    string
	HasBody           bool
	BodyFieldName     string
	HasRawBody        bool
	RawBodyFieldName  string
	HasValidation     bool
	HasResponseWriter bool
	HasRequest        bool
	HasMultipartForm  bool
	MaxMemory         int64 // Max memory for multipart form parsing (default 32MB)
}

// Generate creates wrapper code for the given handlers
func (g *Generator) Generate(result *parser.ParseResult) ([]byte, error) {
	if len(result.Handlers) == 0 {
		return nil, errors.New("no handlers found")
	}

	// Prepare template data using extractors
	data := g.prepareTemplateData(result)

	// Execute template
	var buf bytes.Buffer
	if err := g.tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Format with goimports (handles imports and formatting)
	formatted, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		// Fallback to basic formatting
		formatted, err = format.Source(buf.Bytes())
		if err != nil {
			// Return unformatted code as fallback so callers can still inspect it
			log.Printf("warning: failed to format generated code: %v", err)
			return buf.Bytes(), nil
		}
	}

	return formatted, nil
}

func (g *Generator) prepareTemplateData(result *parser.ParseResult) *TemplateData {
	data := &TemplateData{
		PackageName: result.Source.Package,
		Imports:     []string{},
		Handlers:    []HandlerData{},
		Framework:   g.framework,
	}

	importsMap := make(map[string]bool)

	// Always add apikit import since we use it for error handling
	importsMap["github.com/kausys/apikit"] = true

	for _, handler := range result.Handlers {
		hd := g.prepareHandlerData(&handler, importsMap)
		data.Handlers = append(data.Handlers, hd)
	}

	// Convert imports map to slice and sort alphabetically for deterministic output
	for imp := range importsMap {
		data.Imports = append(data.Imports, imp)
	}
	slices.Sort(data.Imports)

	return data
}

func (g *Generator) prepareHandlerData(handler *parser.Handler, importsMap map[string]bool) HandlerData {
	hd := HandlerData{
		Name:              handler.Name,
		WrapperName:       toCamelCasePrivate(handler.Name) + "APIKit",
		ParseFuncName:     "parse" + capitalize(handler.Name) + "Request",
		ParamType:         handler.ParamType,
		ReturnType:        handler.ReturnType,
		HasResponseWriter: handler.HasResponseWriter,
		HasRequest:        handler.HasRequest,
	}

	if handler.Struct == nil {
		return hd
	}

	// Use extractors to generate code for each field
	extractionCode := g.generateExtractionCode(handler.Struct, importsMap)

	hd.HasExtractionCode = extractionCode != ""
	hd.ExtractionCode = extractionCode

	// Check if we need body parsing and find the body field name
	hd.HasBody = g.hasBodyFields(handler.Struct)
	if hd.HasBody {
		bodyField := g.findBodyField(handler.Struct)
		if bodyField != "" {
			hd.BodyFieldName = bodyField
		}
	}

	// Check if there's a RawBody field
	rawBodyField := g.findRawBodyField(handler.Struct)
	if rawBodyField != "" {
		hd.HasRawBody = true
		hd.RawBodyFieldName = rawBodyField
	}

	// Check if validation is needed
	hd.HasValidation = g.hasValidationTags(handler.Struct)
	if hd.HasValidation {
		// Add validator import
		importsMap["github.com/kausys/apikit/validator"] = true
	}

	// Check if multipart form parsing is needed
	hd.HasMultipartForm = g.hasMultipartFormFields(handler.Struct)
	if hd.HasMultipartForm {
		hd.MaxMemory = 32 << 20 // 32MB default
	}

	return hd
}

func (g *Generator) generateExtractionCode(s *parser.Struct, importsMap map[string]bool) string {
	var lines []string

	// Process each field using the framework extractor
	for _, field := range s.Fields {
		// Handle embedded structs - expand their fields
		if field.IsEmbedded {
			if field.NestedStruct != nil {
				nestedCode := g.generateExtractionCode(field.NestedStruct, importsMap)
				if nestedCode != "" {
					lines = append(lines, nestedCode)
				}
			}
			continue
		}

		// Skip RawBody field (handled separately in template)
		if field.IsRawBody {
			continue
		}

		// Skip special fields
		if field.IsRequest || field.IsResponseWriter {
			code, imports := g.extractSpecialField(&field)
			if code != "" {
				for _, imp := range imports {
					importsMap[imp] = true
				}
				lines = append(lines, code)
			}
			continue
		}

		// Generate extraction code based on field source
		code, imports := g.extractField(&field)
		if code != "" {
			for _, imp := range imports {
				importsMap[imp] = true
			}
			lines = append(lines, code)
		}
	}

	return strings.Join(lines, "\n\t")
}

// extractField generates extraction code for a regular field
func (g *Generator) extractField(field *parser.Field) (string, []string) {
	return g.extractFieldWithPath(field, field.Name)
}

// extractFieldWithPath generates extraction code with a custom field path (for nested fields)
func (g *Generator) extractFieldWithPath(field *parser.Field, fieldPath string) (string, []string) {
	paramName := extractors.GetParameterName(field, g.getTagForSource(field))

	switch {
	case field.InComment == parser.SourcePath || g.hasTag(field, parser.TagPath):
		return g.extractor.ExtractPath(field, paramName, fieldPath)

	case field.InComment == parser.SourceQuery || g.hasTag(field, parser.TagQuery):
		if field.IsSlice {
			return g.extractor.ExtractQuerySlice(field, paramName, fieldPath)
		}
		return g.extractor.ExtractQuery(field, paramName, fieldPath)

	case field.InComment == parser.SourceHeader || g.hasTag(field, parser.TagHeader):
		if field.IsSlice {
			return g.extractor.ExtractHeaderSlice(field, paramName, fieldPath)
		}
		return g.extractor.ExtractHeader(field, paramName, fieldPath)

	case field.InComment == parser.SourceCookie || g.hasTag(field, parser.TagCookie):
		return g.extractor.ExtractCookie(field, paramName, fieldPath)

	case field.InComment == parser.SourceForm || g.hasTag(field, parser.TagForm):
		// Handle nested struct with in:form - extract its fields
		if field.NestedStruct != nil {
			return g.extractNestedFormFields(field.NestedStruct, fieldPath)
		}
		if field.IsFile {
			if field.IsSlice {
				return g.extractor.ExtractFormFiles(field, paramName, fieldPath)
			}
			return g.extractor.ExtractFormFile(field, paramName, fieldPath)
		}
		if field.IsSlice {
			return g.extractor.ExtractFormSlice(field, paramName, fieldPath)
		}
		return g.extractor.ExtractForm(field, paramName, fieldPath)
	}

	// Body fields are handled in the template, not here
	return "", nil
}

// extractNestedFormFields extracts fields from a nested struct marked with in:form
func (g *Generator) extractNestedFormFields(nested *parser.Struct, parentPath string) (string, []string) {
	var lines []string
	var allImports []string

	for _, nestedField := range nested.Fields {
		// Skip special fields
		if nestedField.IsRequest || nestedField.IsResponseWriter || nestedField.IsRawBody {
			continue
		}

		// Check if nested field should be extracted as form field
		hasFormTag := g.hasTag(&nestedField, parser.TagForm)

		if hasFormTag || nestedField.InComment == parser.SourceForm {
			fieldPath := parentPath + "." + nestedField.Name
			paramName := extractors.GetParameterName(&nestedField, parser.TagForm)

			var code string
			var imports []string

			switch {
			case nestedField.IsFile && nestedField.IsSlice:
				code, imports = g.extractor.ExtractFormFiles(&nestedField, paramName, fieldPath)
			case nestedField.IsFile:
				code, imports = g.extractor.ExtractFormFile(&nestedField, paramName, fieldPath)
			case nestedField.IsSlice:
				code, imports = g.extractor.ExtractFormSlice(&nestedField, paramName, fieldPath)
			default:
				code, imports = g.extractor.ExtractForm(&nestedField, paramName, fieldPath)
			}

			if code != "" {
				lines = append(lines, code)
				allImports = append(allImports, imports...)
			}
		}
	}

	return strings.Join(lines, "\n\t"), allImports
}

// extractSpecialField handles *http.Request and http.ResponseWriter fields
func (g *Generator) extractSpecialField(field *parser.Field) (string, []string) {
	if field.IsRequest {
		return g.extractor.ExtractRequest(field)
	}
	if field.IsResponseWriter {
		return g.extractor.ExtractResponse(field)
	}
	return "", nil
}

// getTagForSource returns the tag name for a given field source
func (g *Generator) getTagForSource(field *parser.Field) string {
	switch field.InComment {
	case parser.SourcePath:
		return parser.TagPath
	case parser.SourceQuery:
		return parser.TagQuery
	case parser.SourceHeader:
		return parser.TagHeader
	case parser.SourceCookie:
		return parser.TagCookie
	case parser.SourceForm:
		return parser.TagForm
	default:
		// Check tags
		if g.hasTag(field, parser.TagPath) {
			return parser.TagPath
		}
		if g.hasTag(field, parser.TagQuery) {
			return parser.TagQuery
		}
		if g.hasTag(field, parser.TagHeader) {
			return parser.TagHeader
		}
		if g.hasTag(field, parser.TagCookie) {
			return parser.TagCookie
		}
		if g.hasTag(field, parser.TagForm) {
			return parser.TagForm
		}
		return ""
	}
}

// hasTag checks if a field has a specific struct tag
func (g *Generator) hasTag(field *parser.Field, tagName string) bool {
	if field.StructTag == "" {
		return false
	}
	tag := reflect.StructTag(field.StructTag)
	_, ok := tag.Lookup(tagName)
	return ok
}

func (g *Generator) hasBodyFields(s *parser.Struct) bool {
	for _, field := range s.Fields {
		// Check embedded structs recursively
		if field.IsEmbedded && field.NestedStruct != nil {
			if g.hasBodyFields(field.NestedStruct) {
				return true
			}
		}

		// Field is a body field if:
		// 1. It has IsBody = true (from "in: body" comment), OR
		// 2. It has json:"body" tag
		if field.IsBody {
			return true
		}

		if field.StructTag != "" {
			tag := reflect.StructTag(field.StructTag)
			if jsonTag, ok := tag.Lookup("json"); ok && jsonTag == "body" {
				return true
			}
		}
	}
	return false
}

// findBodyField searches for a body field in the struct
// Returns the field name if found, empty string otherwise
func (g *Generator) findBodyField(s *parser.Struct) string {
	for _, field := range s.Fields {
		// Check embedded structs recursively
		if field.IsEmbedded && field.NestedStruct != nil {
			if bodyField := g.findBodyField(field.NestedStruct); bodyField != "" {
				return bodyField
			}
		}

		// Check if this is a body field
		if field.IsBody {
			return field.Name
		}

		// Check if field has json:"body" tag
		if field.StructTag != "" {
			tag := reflect.StructTag(field.StructTag)
			if jsonTag, ok := tag.Lookup("json"); ok && jsonTag == "body" {
				return field.Name
			}
		}
	}
	return ""
}

// findRawBodyField searches for a RawBody field ([]byte) in the struct
// Returns the field name if found, empty string otherwise
func (g *Generator) findRawBodyField(s *parser.Struct) string {
	for _, field := range s.Fields {
		// Check embedded structs recursively
		if field.IsEmbedded && field.NestedStruct != nil {
			if rawBodyField := g.findRawBodyField(field.NestedStruct); rawBodyField != "" {
				return rawBodyField
			}
		}

		// Check if this is a RawBody field
		// More flexible detection: any field with type []byte that contains "body" (case-insensitive)
		if field.IsRawBody {
			return field.Name
		}
	}
	return ""
}

// hasValidationTags checks if the struct has any validation tags
// Returns true if any field has a validate tag
func (g *Generator) hasValidationTags(s *parser.Struct) bool {
	for _, field := range s.Fields {
		// Check embedded structs recursively
		if field.IsEmbedded && field.NestedStruct != nil {
			if g.hasValidationTags(field.NestedStruct) {
				return true
			}
		}

		// Check if this field has a validate tag
		if field.StructTag != "" {
			tag := reflect.StructTag(field.StructTag)
			if _, ok := tag.Lookup("validate"); ok {
				return true
			}
		}
	}
	return false
}

// hasMultipartFormFields checks if the struct has any multipart form fields
// Returns true if any field has a form tag or is a file upload field
func (g *Generator) hasMultipartFormFields(s *parser.Struct) bool {
	for _, field := range s.Fields {
		// Check embedded structs recursively
		if field.IsEmbedded && field.NestedStruct != nil {
			if g.hasMultipartFormFields(field.NestedStruct) {
				return true
			}
		}

		// Check if this is a file field
		if field.IsFile {
			return true
		}

		// Check if this field has a form tag
		if field.StructTag != "" {
			tag := reflect.StructTag(field.StructTag)
			if _, ok := tag.Lookup("form"); ok {
				return true
			}
		}

		// Check if field is marked with // in:form comment
		if field.InComment == "form" {
			return true
		}
	}
	return false
}

// Template helper functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"toLower": strings.ToLower,
		"toUpper": strings.ToUpper,
	}
}

// toCamelCasePrivate converts a string to camelCase with first letter lowercase
// Example: "GetUser" -> "getUser", "SearchUsers" -> "searchUsers"
func toCamelCasePrivate(s string) string {
	if s == "" {
		return s
	}
	// Convert first character to lowercase
	runes := []rune(s)
	runes[0] = []rune(strings.ToLower(string(runes[0])))[0]
	return string(runes)
}

// capitalize converts the first letter to uppercase (PascalCase)
// Example: "listTransactions" -> "ListTransactions", "getUser" -> "GetUser"
func capitalize(s string) string {
	if s == "" {
		return s
	}
	// Convert first character to uppercase
	runes := []rune(s)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}
