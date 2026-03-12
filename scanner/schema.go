package scanner

import (
	"go/ast"
	"go/token"
	"strings"
)

// processSchemas processes swagger:model, swagger:parameters, swagger:oneOf, and swagger:anyOf directives.
func (s *Scanner) processSchemas(filePath string, file *ast.File) error {
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
					processStructFields(structInfo, t)
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
func processStructFields(structInfo *StructInfo, structType *ast.StructType) {
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

		fieldInfo := parseField(field)
		if fieldInfo != nil {
			// Multiply by 1000 to leave room for embedded field sub-indices
			fieldInfo.Index = index * 1000
			structInfo.Fields = append(structInfo.Fields, fieldInfo)
		}
		index++
	}
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

// parseField parses a single struct field.
func parseField(field *ast.Field) *FieldInfo {
	// Skip fields without names (embedded)
	if len(field.Names) == 0 {
		return nil
	}

	// Skip unexported fields
	if !ast.IsExported(field.Names[0].Name) {
		return nil
	}

	// Check if field has swagger:ignore directive BEFORE creating FieldInfo
	if hasFieldIgnoreDirective(field) {
		return nil
	}

	fieldInfo := &FieldInfo{
		Name:        field.Names[0].Name,
		Tags:        make(map[string]string),
		Validations: make(map[string]string),
	}

	// Determine field type
	determineFieldType(fieldInfo, field.Type)

	// Parse struct tags
	if field.Tag != nil {
		parseFieldTags(fieldInfo, field.Tag.Value)
	}

	// Parse field doc comments
	if field.Doc != nil {
		parseFieldDoc(fieldInfo, field.Doc)
	}

	// Also check inline comments
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
func determineFieldType(fieldInfo *FieldInfo, expr ast.Expr) {
	switch t := expr.(type) {
	case *ast.Ident:
		fieldInfo.Type = t.Name
	case *ast.StarExpr:
		fieldInfo.IsPointer = true
		determineFieldType(fieldInfo, t.X)
	case *ast.ArrayType:
		fieldInfo.IsArray = true
		determineFieldType(fieldInfo, t.Elt)
	case *ast.MapType:
		fieldInfo.IsMap = true
		if keyIdent, ok := t.Key.(*ast.Ident); ok {
			fieldInfo.MapKeyType = keyIdent.Name
		}
		determineFieldType(fieldInfo, t.Value)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			fieldInfo.Type = x.Name + "." + t.Sel.Name
		}
	case *ast.StructType:
		// Inline struct type
		fieldInfo.Type = "object"
		fieldInfo.IsInlineStruct = true
		fieldInfo.InlineStruct = processInlineStruct(fieldInfo.Name, t)
	}
}

// processInlineStruct processes an inline struct and returns a StructInfo.
func processInlineStruct(name string, structType *ast.StructType) *StructInfo {
	structInfo := &StructInfo{
		Name:   name,
		Fields: []*FieldInfo{},
	}

	for _, field := range structType.Fields.List {
		// Skip embedded fields
		if len(field.Names) == 0 {
			continue
		}

		fieldInfo := parseField(field)
		if fieldInfo != nil {
			structInfo.Fields = append(structInfo.Fields, fieldInfo)
		}
	}

	return structInfo
}

// parseFieldTags parses struct tags (json, yaml, validate, etc.).
func parseFieldTags(fieldInfo *FieldInfo, tagValue string) {
	// Remove backticks
	tagValue = strings.Trim(tagValue, "`")

	// Parse json tag
	if jsonTag := getTagValue(tagValue, "json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 && parts[0] != "-" {
			fieldInfo.Tags["json"] = parts[0]
		}
		for _, opt := range parts[1:] {
			if opt == "omitempty" || opt == "omitzero" {
				fieldInfo.HasOmitempty = true
			}
		}
	}

	// Parse yaml tag
	if yamlTag := getTagValue(tagValue, "yaml"); yamlTag != "" {
		parts := strings.Split(yamlTag, ",")
		if len(parts) > 0 && parts[0] != "-" {
			fieldInfo.Tags["yaml"] = parts[0]
		}
	}

	// Parse example tag
	if example := getTagValue(tagValue, "example"); example != "" {
		fieldInfo.Example = example
	}

	// Parse default tag
	if def := getTagValue(tagValue, "default"); def != "" {
		fieldInfo.Default = def
	}

	// Parse validate tag for validations
	if validate := getTagValue(tagValue, "validate"); validate != "" {
		parseValidateTag(fieldInfo, validate)
	}
}

// getTagValue extracts the value for a specific tag key.
func getTagValue(tagValue, key string) string {
	// Look for key:"value" pattern
	keyPattern := key + `:"([^"]*)"`
	re := getCompiledRegex(keyPattern)
	matches := re.FindStringSubmatch(tagValue)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// parseValidateTag parses validation rules from the validate tag.
func parseValidateTag(fieldInfo *FieldInfo, validate string) {
	rules := strings.SplitSeq(validate, ",")
	for rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "required" {
			fieldInfo.Required = true
			fieldInfo.ExplicitRequired = true
		} else if v, ok := strings.CutPrefix(rule, "min="); ok {
			fieldInfo.Validations["min"] = v
		} else if v, ok := strings.CutPrefix(rule, "max="); ok {
			fieldInfo.Validations["max"] = v
		} else if v, ok := strings.CutPrefix(rule, "len="); ok {
			fieldInfo.Validations["len"] = v
		} else {
			switch rule {
			case "email":
				fieldInfo.Validations["format"] = "email"
			case "uuid":
				fieldInfo.Validations["format"] = "uuid"
			case "url":
				fieldInfo.Validations["format"] = "uri"
			}
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

	// Extract single-line directive values
	fieldInfo.Example = extractDirectiveValue(doc, ExampleDirective)
	fieldInfo.Default = extractDirectiveValue(doc, DefaultDirective)

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

	// Extract description (non-directive lines)
	fieldInfo.Description = extractFieldDescription(comments, knownFieldDirectives)
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
