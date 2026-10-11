package scanner

import (
	"go/ast"
	"go/token"
	"reflect"
	"strings"
)

// processSchemas processes swagger:model, swagger:parameters, swagger:oneOf, and swagger:anyOf directives.
func (s *Scanner) processSchemas(filePath string, file *ast.File) error {
	imports := extractFileImports(file)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if genDecl.Doc == nil {
				continue
			}

			var name string
			var isParameter, isModel, isOneOfModel, isAnyOfModel bool

			if hasDirective(genDecl.Doc, ModelDirective) {
				name = extractDirectiveValue(genDecl.Doc, ModelDirective)
				isModel = true
			}

			if hasDirective(genDecl.Doc, ParameterDirective) {
				name = extractDirectiveValue(genDecl.Doc, ParameterDirective)
				isParameter = true
			}

			// Check for swagger:oneOf directive
			if hasDirective(genDecl.Doc, OneOfModelDirective) {
				customName := extractDirectiveValue(genDecl.Doc, OneOfModelDirective)
				if customName != "" {
					name = customName
				}
				isOneOfModel = true
				isModel = true // oneOf models are also models
			}

			// Check for swagger:anyOf directive
			if hasDirective(genDecl.Doc, AnyOfModelDirective) {
				customName := extractDirectiveValue(genDecl.Doc, AnyOfModelDirective)
				if customName != "" {
					name = customName
				}
				isAnyOfModel = true
				isModel = true // anyOf models are also models
			}

			if !isModel && !isParameter {
				continue
			}

			if name == "" {
				name = typeSpec.Name.Name
			}

			// List of directives to exclude from description
			descExclude := []string{
				SwaggerPrefix, OneOfDirective, AllOfDirective, AnyOfDirective,
				SpecDirective, DiscriminatorDirective,
			}

			structInfo := &StructInfo{
				Name:         name,
				Fields:       []*FieldInfo{},
				Description:  extractDescription(genDecl.Doc, descExclude),
				IsParameter:  isParameter,
				IsModel:      isModel,
				IsOneOfModel: isOneOfModel,
				IsAnyOfModel: isAnyOfModel,
				SourceFile:   filePath,
				OneOf:        extractCompositionSchemas(genDecl.Doc, OneOfDirective),
				AllOf:        extractCompositionSchemas(genDecl.Doc, AllOfDirective),
				AnyOf:        extractCompositionSchemas(genDecl.Doc, AnyOfDirective),
				Specs:        extractSpecs(genDecl.Doc),
			}

			// Extract discriminator if present
			if isOneOfModel || isAnyOfModel {
				structInfo.Discriminator = extractDiscriminator(genDecl.Doc)
			}

			// Process type based on its underlying kind
			switch t := typeSpec.Type.(type) {
			case *ast.StructType:
				structInfo.UnderlyingKind = KindStruct
				if isOneOfModel || isAnyOfModel {
					processOneOfAnyOfFields(structInfo, t, isOneOfModel)
				} else {
					processStructFields(structInfo, t, imports)
				}
			case *ast.ArrayType:
				structInfo.UnderlyingKind = KindArray
				structInfo.ElementType = extractTypeName(t.Elt)
			case *ast.MapType:
				structInfo.UnderlyingKind = KindMap
				structInfo.ElementType = extractTypeName(t.Value)
				if keyIdent, ok := t.Key.(*ast.Ident); ok {
					structInfo.MapKeyType = keyIdent.Name
				}
			case *ast.Ident:
				structInfo.UnderlyingKind = KindPrimitive
				structInfo.ElementType = t.Name
			case *ast.SelectorExpr:
				structInfo.UnderlyingKind = KindPrimitive
				structInfo.ElementType = extractTypeName(t)
			}

			s.Structs[name] = structInfo
			s.TypeToStruct[typeSpec.Name.Name] = name
			s.StructSources[name] = filePath
		}
	}
	return nil
}

// extractDiscriminator extracts discriminator configuration from comments.
// Only extracts the property name; mapping is built from field-level directives.
func extractDiscriminator(doc *ast.CommentGroup) *DiscriminatorInfo {
	if doc == nil {
		return nil
	}

	propertyName := extractDirectiveValue(doc, DiscriminatorDirective)
	if propertyName == "" {
		return nil
	}

	return &DiscriminatorInfo{
		PropertyName: propertyName,
		Mapping:      make(map[string]string),
	}
}

// processOneOfAnyOfFields processes embedded fields for oneOf/anyOf models.
func processOneOfAnyOfFields(structInfo *StructInfo, structType *ast.StructType, isOneOf bool) {
	if structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		// Only process embedded fields (no names)
		if len(field.Names) != 0 {
			continue
		}

		// Check if field has oneOfOption or anyOfOption directive
		var optionDirective string
		var discriminatorValue string

		// Check doc comments
		if field.Doc != nil {
			if isOneOf {
				optionDirective, discriminatorValue = extractOptionDirective(field.Doc, OneOfOptionDirective)
			} else {
				optionDirective, discriminatorValue = extractOptionDirective(field.Doc, AnyOfOptionDirective)
			}
		}

		// Also check inline comments if not found
		if optionDirective == "" && field.Comment != nil {
			if isOneOf {
				optionDirective, discriminatorValue = extractOptionDirective(field.Comment, OneOfOptionDirective)
			} else {
				optionDirective, discriminatorValue = extractOptionDirective(field.Comment, AnyOfOptionDirective)
			}
		}

		if optionDirective == "" {
			continue
		}

		// Extract the type name
		typeName := extractTypeName(field.Type)
		if typeName == "" {
			continue
		}

		if isOneOf {
			structInfo.OneOfOptions = append(structInfo.OneOfOptions, typeName)
		} else {
			structInfo.AnyOfOptions = append(structInfo.AnyOfOptions, typeName)
		}

		// Add discriminator mapping if present
		if discriminatorValue != "" && structInfo.Discriminator != nil {
			structInfo.Discriminator.Mapping[discriminatorValue] = typeName
		}
	}
}

// extractOptionDirective extracts the option directive and its discriminator value.
// Supports formats:
//   - swagger:oneOfOption
//   - swagger:oneOfOption discriminator=VALUE
func extractOptionDirective(doc *ast.CommentGroup, directive string) (string, string) {
	if doc == nil {
		return "", ""
	}

	for _, comment := range doc.List {
		text := strings.TrimPrefix(comment.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)

		remainder, ok := strings.CutPrefix(text, directive)
		if !ok {
			continue
		}

		// Found the directive
		remainder = strings.TrimSpace(remainder)

		// Check for discriminator=VALUE
		if value, ok := strings.CutPrefix(remainder, "discriminator="); ok {
			value = strings.TrimSpace(value)
			return directive, value
		}

		// Just the directive without discriminator value
		return directive, ""
	}

	return "", ""
}

// extractCompositionSchemas extracts schema names from composition directives.
func extractCompositionSchemas(doc *ast.CommentGroup, directive string) []string {
	if doc == nil {
		return nil
	}

	value := extractDirectiveValue(doc, directive)
	if value == "" {
		return nil
	}

	var schemas []string
	for s := range strings.SplitSeq(value, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			schemas = append(schemas, s)
		}
	}
	return schemas
}

// processStructFields processes all fields in a struct.
func processStructFields(structInfo *StructInfo, structType *ast.StructType, imports map[string]string) {
	if structType.Fields == nil {
		return
	}

	index := 0
	for _, field := range structType.Fields.List {
		// Check if this is an embedded field (no names)
		if len(field.Names) == 0 {
			embeddedType := extractTypeName(field.Type)
			if embeddedType != "" {
				structInfo.EmbeddedTypes = append(structInfo.EmbeddedTypes, embeddedType)
				structInfo.EmbeddedTypeInfos = append(structInfo.EmbeddedTypeInfos, &EmbeddedTypeInfo{
					Name:  embeddedType,
					Index: index,
				})
			}
			index++
			continue
		}

		// One declaration may name several fields (A, B string); each is a
		// field of its own. Only the first used to be kept.
		for _, fieldInfo := range parseFields(field, imports) {
			// Multiply by 1000 to leave room for embedded field sub-indices
			fieldInfo.Index = index * 1000
			structInfo.Fields = append(structInfo.Fields, fieldInfo)
			index++
		}
	}
}

// extractFileImports extracts import alias → full path mappings from a Go file.
func extractFileImports(file *ast.File) map[string]string {
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			// Use last segment of path as default alias
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}
		imports[alias] = path
	}
	return imports
}

// extractTypeName extracts the type name from an AST expression.
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return extractTypeName(t.X)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return ""
}

// parseFields parses the fields one declaration names.
func parseFields(field *ast.Field, imports map[string]string) []*FieldInfo {
	var fields []*FieldInfo
	for _, name := range field.Names {
		if f := parseField(field, name.Name, imports); f != nil {
			fields = append(fields, f)
		}
	}
	return fields
}

// parseField parses a single struct field.
func parseField(field *ast.Field, name string, imports map[string]string) *FieldInfo {
	// Skip unexported fields
	if !ast.IsExported(name) {
		return nil
	}

	// Check if field has swagger:ignore directive BEFORE creating FieldInfo
	if hasFieldIgnoreDirective(field) {
		return nil
	}

	fieldInfo := &FieldInfo{
		Name:        name,
		Tags:        make(map[string]string),
		Validations: make(map[string]string),
	}

	// Determine field type
	determineFieldType(fieldInfo, field.Type, imports)

	// Parse struct tags
	if field.Tag != nil {
		if !parseFieldTags(fieldInfo, field.Tag.Value) {
			return nil
		}
	}

	// Parse field doc comments, then the trailing comment: a directive in
	// either applies, and the trailing comment describes the field only when
	// the doc comment does not.
	if field.Doc != nil {
		parseFieldDoc(fieldInfo, field.Doc)
	}
	if field.Comment != nil {
		parseFieldDoc(fieldInfo, field.Comment)
	}

	return fieldInfo
}

// hasFieldIgnoreDirective checks if a field has the swagger:ignore directive.
func hasFieldIgnoreDirective(field *ast.Field) bool {
	// Check field doc comments
	if field.Doc != nil && hasDirective(field.Doc, IgnoreDirective) {
		return true
	}
	// Check inline comments
	if field.Comment != nil && hasDirective(field.Comment, IgnoreDirective) {
		return true
	}
	return false
}

// determineFieldType determines the type information for a field.
func determineFieldType(fieldInfo *FieldInfo, expr ast.Expr, imports map[string]string) {
	switch t := expr.(type) {
	case *ast.Ident:
		fieldInfo.Type = t.Name
	case *ast.StarExpr:
		fieldInfo.IsPointer = true
		determineFieldType(fieldInfo, t.X, imports)
	case *ast.ArrayType:
		fieldInfo.IsArray = true
		determineFieldType(fieldInfo, t.Elt, imports)
	case *ast.MapType:
		fieldInfo.IsMap = true
		if keyIdent, ok := t.Key.(*ast.Ident); ok {
			fieldInfo.MapKeyType = keyIdent.Name
		}
		determineFieldType(fieldInfo, t.Value, imports)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			fieldInfo.Type = x.Name + "." + t.Sel.Name
			// Resolve fully-qualified type using import map
			if importPath, found := imports[x.Name]; found {
				fieldInfo.QualifiedType = importPath + "." + t.Sel.Name
			}
		}
	case *ast.StructType:
		// Inline struct type
		fieldInfo.Type = "object"
		fieldInfo.IsInlineStruct = true
		fieldInfo.InlineStruct = processInlineStruct(fieldInfo.Name, t, imports)
	}
}

// processInlineStruct processes an inline struct and returns a StructInfo.
func processInlineStruct(name string, structType *ast.StructType, imports map[string]string) *StructInfo {
	structInfo := &StructInfo{
		Name:   name,
		Fields: []*FieldInfo{},
	}

	for _, field := range structType.Fields.List {
		// Skip embedded fields
		if len(field.Names) == 0 {
			continue
		}

		structInfo.Fields = append(structInfo.Fields, parseFields(field, imports)...)
	}

	return structInfo
}

// bindingTags are the tags a generated handler binds a request field from,
// in the order it looks for them. The spec reads the same ones, so a
// documented parameter is the one the handler reads.
var bindingTags = []string{"path", "query", "header", "cookie", "form"}

// parseFieldTags parses struct tags (json, yaml, validate, binding tags). It
// reports false for a field the wire never carries: json:"-" with no binding.
func parseFieldTags(fieldInfo *FieldInfo, tagValue string) bool {
	tag := reflect.StructTag(strings.Trim(tagValue, "`"))

	// A binding tag says where a request field comes from and under what name.
	for _, source := range bindingTags {
		if value, ok := tag.Lookup(source); ok {
			name, _, _ := strings.Cut(value, ",")
			fieldInfo.In = source
			if name != "" {
				fieldInfo.Tags[source] = name
			}
			// A form tag names one part of a multipart body. (The "in: form"
			// directive, by contrast, marks a struct as the whole body.)
			fieldInfo.FormPart = source == "form"
			break
		}
	}

	// Parse json tag
	if jsonTag, ok := tag.Lookup("json"); ok {
		name, opts, _ := strings.Cut(jsonTag, ",")
		if name == "-" && opts == "" && fieldInfo.In == "" {
			return false
		}
		if name != "" && name != "-" {
			fieldInfo.Tags["json"] = name
		}
		for opt := range strings.SplitSeq(opts, ",") {
			if opt == "omitempty" || opt == "omitzero" {
				fieldInfo.HasOmitempty = true
			}
		}
		// The field a generated handler decodes the body into.
		if name == "body" && fieldInfo.In == "" {
			fieldInfo.In = "body"
			fieldInfo.IsRequestBody = true
		}
	}

	// Parse yaml tag
	if yamlTag := tag.Get("yaml"); yamlTag != "" {
		name, _, _ := strings.Cut(yamlTag, ",")
		if name != "" && name != "-" {
			fieldInfo.Tags["yaml"] = name
		}
	}

	// Parse example tag
	if example := tag.Get("example"); example != "" {
		fieldInfo.Example = example
	}

	// Parse default tag
	if def := tag.Get("default"); def != "" {
		fieldInfo.Default = def
	}

	// Parse validate tag for validations
	if validate := tag.Get("validate"); validate != "" {
		parseValidateTag(fieldInfo, validate)
	}
	return true
}

// parseValidateTag records the rules of a validate tag. Which keyword a rule
// becomes depends on the schema it lands on — min is a minimum for a number,
// a minLength for a string, a minItems for an array — so the generator maps
// them once the type is known.
func parseValidateTag(fieldInfo *FieldInfo, validate string) {
	for rule := range strings.SplitSeq(validate, ",") {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		fieldInfo.Rules = append(fieldInfo.Rules, rule)
		if rule == "required" {
			fieldInfo.Required = true
			fieldInfo.ExplicitRequired = true
		}
	}
}

// knownFieldDirectives lists all known field-level directives
var knownFieldDirectives = []string{
	SwaggerPrefix,
	ExampleDirective,
	DefaultDirective,
	RequiredDirective,
	NullableDirective,
	FormatDirective,
	InDirective,
	MinimumDirective,
	MaximumDirective,
	MinLengthDirective,
	MaxLengthDirective,
	PatternDirective,
	MinItemsDirective,
	MaxItemsDirective,
	UniqueItemsDirective,
	ReadOnlyDirective,
	WriteOnlyDirective,
	IgnoreDirective,
	OneOfDirective,
	AllOfDirective,
	AnyOfDirective,
}

// parseFieldDoc parses documentation comments for a field.
func parseFieldDoc(fieldInfo *FieldInfo, doc *ast.CommentGroup) {
	if doc == nil {
		return
	}

	// Check for ignore directive first
	if hasDirective(doc, IgnoreDirective) {
		return
	}

	comments := trimComments(doc)

	// Extract single-line directive values. A directive overrides the tag;
	// its absence leaves the tag's value alone.
	if example := extractDirectiveValue(doc, ExampleDirective); example != "" {
		fieldInfo.Example = example
	}
	if def := extractDirectiveValue(doc, DefaultDirective); def != "" {
		fieldInfo.Default = def
	}

	// Handle required directive
	if requiredValue := extractDirectiveValue(doc, RequiredDirective); requiredValue != "" {
		switch requiredValue {
		case "true":
			fieldInfo.Required = true
			fieldInfo.ExplicitRequired = true
		case "false":
			fieldInfo.Required = false
			fieldInfo.ExplicitOptional = true
		}
	}

	// Extract nullable directive
	if nullableVal := extractDirectiveValue(doc, NullableDirective); nullableVal == "true" {
		fieldInfo.Nullable = true
	}

	// Extract format directive
	if format := extractDirectiveValue(doc, FormatDirective); format != "" {
		fieldInfo.Validations["format"] = format
	}

	// Extract in directive (for parameter location)
	// Handle format like "in:header 'Origin'" - only take the first word (header)
	if inValue := extractDirectiveValue(doc, InDirective); inValue != "" {
		// Take only the first word (e.g., "header" from "header 'Origin'")
		inValue = strings.Fields(inValue)[0]
		fieldInfo.In = inValue
		// Both "body" and "form" indicate request body
		// "form" is more semantic for multipart/form-data
		if inValue == "body" || inValue == "form" {
			fieldInfo.IsRequestBody = true
		}
	}

	// Extract numeric constraints
	if minVal := extractDirectiveValue(doc, MinimumDirective); minVal != "" {
		fieldInfo.Validations["min"] = minVal
	}
	if maxVal := extractDirectiveValue(doc, MaximumDirective); maxVal != "" {
		fieldInfo.Validations["max"] = maxVal
	}

	// Extract string length constraints
	if minLenVal := extractDirectiveValue(doc, MinLengthDirective); minLenVal != "" {
		fieldInfo.Validations["minLength"] = minLenVal
	}
	if maxLenVal := extractDirectiveValue(doc, MaxLengthDirective); maxLenVal != "" {
		fieldInfo.Validations["maxLength"] = maxLenVal
	}

	// Extract pattern
	if pattern := extractDirectiveValue(doc, PatternDirective); pattern != "" {
		fieldInfo.Validations["pattern"] = pattern
	}

	// Extract array constraints
	if minItemsVal := extractDirectiveValue(doc, MinItemsDirective); minItemsVal != "" {
		fieldInfo.Validations["minItems"] = minItemsVal
	}
	if maxItemsVal := extractDirectiveValue(doc, MaxItemsDirective); maxItemsVal != "" {
		fieldInfo.Validations["maxItems"] = maxItemsVal
	}
	if hasDirective(doc, UniqueItemsDirective) {
		fieldInfo.Validations["uniqueItems"] = "true"
	}

	// Extract read/write constraints
	if hasDirective(doc, ReadOnlyDirective) {
		fieldInfo.Validations["readOnly"] = "true"
	}
	if hasDirective(doc, WriteOnlyDirective) {
		fieldInfo.Validations["writeOnly"] = "true"
	}

	// Extract description (non-directive lines), unless an earlier comment
	// already gave one.
	if fieldInfo.Description == "" {
		fieldInfo.Description = extractFieldDescription(comments, knownFieldDirectives)
	}
}

// extractFieldDescription extracts description lines, excluding directives.
func extractFieldDescription(comments, knownDirectives []string) string {
	var lines []string

	for _, comment := range comments {
		// Skip directives
		isDirective := false
		for _, directive := range knownDirectives {
			if strings.HasPrefix(comment, directive) {
				isDirective = true
				break
			}
		}

		if !isDirective && comment != "" {
			lines = append(lines, comment)
		}
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n")
}
