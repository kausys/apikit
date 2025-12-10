// Package extractors provides framework-specific parameter extraction for HTTP handlers
package extractors

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/kausys/apikit/parser"
	"github.com/kausys/apikit/types"
)

// Helper functions for code generation

// GetDefaultTag returns the default tag value from the field's struct tag
func GetDefaultTag(field *parser.Field) string {
	if field.StructTag == "" {
		return ""
	}
	tag := reflect.StructTag(field.StructTag)
	return tag.Get("default")
}

// GenerateExtractionCode generates code for extracting a value with optional default
// This is a generic helper to reduce code duplication
func GenerateExtractionCode(varName, fieldName, typeName string, field *parser.Field, parsingFunc func(string, string) string, imports []string) (string, []string) {
	defaultTag := GetDefaultTag(field)
	hasDefault := defaultTag != ""

	// For string types, no parsing needed
	if IsStringType(typeName) {
		if hasDefault {
			return fmt.Sprintf(`if val := %s; val != "" {
		payload.%s = val
	} else {
		%s
	}`, varName, fieldName, GenerateDefaultValue(fieldName, defaultTag, typeName)), imports
		}
		return fmt.Sprintf(`if val := %s; val != "" {
		payload.%s = val
	}`, varName, fieldName), imports
	}

	// For types that need parsing
	parsingCode := parsingFunc("val", fieldName)

	if hasDefault {
		return fmt.Sprintf(`if val := %s; val != "" {
		%s
	} else {
		%s
	}`, varName, parsingCode, GenerateDefaultValue(fieldName, defaultTag, typeName)), imports
	}

	return fmt.Sprintf(`if val := %s; val != "" {
		%s
	}`, varName, parsingCode), imports
}

// GenerateIntParsing generates code to parse an integer from a string
func GenerateIntParsing(varName, fieldName, typeName string) string {
	return fmt.Sprintf(`if i, err := strconv.ParseInt(%s, 10, 64); err == nil {
		payload.%s = %s(i)
	} else {
		return fmt.Errorf("invalid %s: %%w", err)
	}`, varName, fieldName, typeName, fieldName)
}

// GenerateUintParsing generates code to parse an unsigned integer from a string
func GenerateUintParsing(varName, fieldName, typeName string) string {
	return fmt.Sprintf(`if i, err := strconv.ParseUint(%s, 10, 64); err == nil {
		payload.%s = %s(i)
	} else {
		return fmt.Errorf("invalid %s: %%w", err)
	}`, varName, fieldName, typeName, fieldName)
}

// GenerateFloatParsing generates code to parse a float from a string
func GenerateFloatParsing(varName, fieldName, bitSize string) string {
	return fmt.Sprintf(`if f, err := strconv.ParseFloat(%s, %s); err == nil {
		payload.%s = float%s(f)
	} else {
		return fmt.Errorf("invalid %s: %%w", err)
	}`, varName, bitSize, fieldName, bitSize, fieldName)
}

// GenerateBoolParsing generates code to parse a boolean from a string
func GenerateBoolParsing(varName, fieldName string) string {
	return fmt.Sprintf(`if b, err := strconv.ParseBool(%s); err == nil {
		payload.%s = b
	} else {
		return fmt.Errorf("invalid %s: %%w", err)
	}`, varName, fieldName, fieldName)
}

// toCamelCase converts a PascalCase string to camelCase (first letter lowercase)
// Examples: "UserID" -> "userID", "FirstName" -> "firstName", "APIKey" -> "apiKey"
func toCamelCase(s string) string {
	if s == "" {
		return s
	}

	// Convert first character to lowercase
	runes := []rune(s)
	if runes[0] >= 'A' && runes[0] <= 'Z' {
		runes[0] = runes[0] + 32 // Convert to lowercase (A-Z to a-z)
	}
	return string(runes)
}

// GetParameterName returns the parameter name to use for extraction
// Priority: tag value > InCommentName > json tag > field name (converted to camelCase)
// This is a public helper that can be used by custom extractors
// Parameters:
//   - field: The field to get the parameter name for
//   - tagName: The name of the tag to look up (e.g., "query", "path", "header")
func GetParameterName(field *parser.Field, tagName string) string {
	// Priority 1: Use tag value if available
	if field.StructTag != "" {
		tag := reflect.StructTag(field.StructTag)
		if val, ok := tag.Lookup(tagName); ok {
			// If tag exists but is empty, fall through to use field name
			if val != "" {
				return val
			}
		}
	}

	// Priority 2: Use comment name if available
	if field.InCommentName != "" {
		return field.InCommentName
	}

	// Priority 3: Use json tag value as fallback (handles omitempty)
	if field.StructTag != "" {
		tag := reflect.StructTag(field.StructTag)
		if val, ok := tag.Lookup(parser.TagJSON); ok && val != "" {
			// Handle "fieldName,omitempty" format - extract just the field name
			if commaIdx := strings.Index(val, ","); commaIdx != -1 {
				val = val[:commaIdx]
			}
			if val != "" && val != "-" {
				return val
			}
		}
	}

	// Priority 4: Convert field name to camelCase as fallback
	return toCamelCase(field.Name)
}

// GenerateCodeByType generates extraction code based on the field type
// This is a public helper that handles all type-specific parsing logic
// Returns: (code, imports)
func GenerateCodeByType(varName, fieldName, typeName string, field *parser.Field) (string, []string) {
	var imports []string
	var code string

	switch {
	case IsStringType(typeName):
		code, imports = GenerateExtractionCode(varName, fieldName, typeName, field, nil, imports)

	case IsIntType(typeName):
		imports = append(imports, "strconv")
		parsingFunc := func(v, f string) string { return GenerateIntParsing(v, f, typeName) }
		code, imports = GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)

	case IsUintType(typeName):
		imports = append(imports, "strconv")
		parsingFunc := func(v, f string) string { return GenerateUintParsing(v, f, typeName) }
		code, imports = GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)

	case IsFloatType(typeName):
		imports = append(imports, "strconv")
		bitSize := "64"
		if typeName == "float32" {
			bitSize = "32"
		}
		parsingFunc := func(v, f string) string { return GenerateFloatParsing(v, f, bitSize) }
		code, imports = GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)

	case IsBoolType(typeName):
		imports = append(imports, "strconv")
		parsingFunc := func(v, f string) string { return GenerateBoolParsing(v, f) }
		code, imports = GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)

	default:
		// Check if there's a custom type extractor registered in the Type Registry
		if typeExtractor, ok := types.Get(typeName); ok {
			// Use the registered ParseFunc to generate parsing code
			parsingFunc := func(v, f string) string {
				return typeExtractor.ParseFunc(v, f, field.IsPointer)
			}

			// Add import if specified
			if typeExtractor.Import != "" {
				imports = append(imports, typeExtractor.Import)
			}

			// Generate extraction code with the custom parser
			code, imports = GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)
		} else if !field.IsEmbedded {
			// Fallback: for unknown custom types (e.g., enums), cast the string value
			// This handles types like model.AgentStatus, model.UserRole, etc.
			// BUT: Skip embedded structs - they should have been expanded by the parser
			parsingFunc := func(v, f string) string {
				return fmt.Sprintf(`payload.%s = %s(%s)`, f, typeName, v)
			}
			code, imports = GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)
		}
		// else: Embedded struct without extractor - skip (should have been expanded)
	}

	return code, imports
}

// GenerateSliceCodeByType generates code to parse a slice of values
// This handles the standard HTTP pattern: ?tags=go&tags=api&tags=http
// Returns: (code, imports)
func GenerateSliceCodeByType(varName, fieldName, elementType string, field *parser.Field) (string, []string) {
	var imports []string
	var code string

	switch {
	case IsStringType(elementType):
		// For []string, direct assignment
		code = fmt.Sprintf(`if vals := %s; len(vals) > 0 {
		payload.%s = vals
	}`, varName, fieldName)

	case IsIntType(elementType):
		// For []int, []int64, etc. - parse each element
		imports = append(imports, "strconv")
		code = fmt.Sprintf(`if vals := %s; len(vals) > 0 {
		payload.%s = make([]%s, 0, len(vals))
		for i, val := range vals {
			if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
				payload.%s = append(payload.%s, %s(parsed))
			} else {
				return fmt.Errorf("invalid %s[%%d]: %%w", i, err)
			}
		}
	}`, varName, fieldName, elementType, fieldName, fieldName, elementType, fieldName)

	case IsUintType(elementType):
		// For []uint, []uint64, etc. - parse each element
		imports = append(imports, "strconv")
		code = fmt.Sprintf(`if vals := %s; len(vals) > 0 {
		payload.%s = make([]%s, 0, len(vals))
		for i, val := range vals {
			if parsed, err := strconv.ParseUint(val, 10, 64); err == nil {
				payload.%s = append(payload.%s, %s(parsed))
			} else {
				return fmt.Errorf("invalid %s[%%d]: %%w", i, err)
			}
		}
	}`, varName, fieldName, elementType, fieldName, fieldName, elementType, fieldName)

	case IsFloatType(elementType):
		// For []float32, []float64 - parse each element
		imports = append(imports, "strconv")
		bitSize := "64"
		if elementType == "float32" {
			bitSize = "32"
		}
		code = fmt.Sprintf(`if vals := %s; len(vals) > 0 {
		payload.%s = make([]%s, 0, len(vals))
		for i, val := range vals {
			if parsed, err := strconv.ParseFloat(val, %s); err == nil {
				payload.%s = append(payload.%s, %s(parsed))
			} else {
				return fmt.Errorf("invalid %s[%%d]: %%w", i, err)
			}
		}
	}`, varName, fieldName, elementType, bitSize, fieldName, fieldName, elementType, fieldName)

	case IsBoolType(elementType):
		// For []bool - parse each element
		imports = append(imports, "strconv")
		code = fmt.Sprintf(`if vals := %s; len(vals) > 0 {
		payload.%s = make([]bool, 0, len(vals))
		for i, val := range vals {
			if parsed, err := strconv.ParseBool(val); err == nil {
				payload.%s = append(payload.%s, parsed)
			} else {
				return fmt.Errorf("invalid %s[%%d]: %%w", i, err)
			}
		}
	}`, varName, fieldName, fieldName, fieldName, fieldName)

	default:
		// Check if there's a custom type extractor for the element type
		if typeExtractor, ok := types.Get(elementType); ok {
			// Add import if specified
			if typeExtractor.Import != "" {
				imports = append(imports, typeExtractor.Import)
			}

			// Generate code to parse each element in the slice using the custom parser
			// The ParseFunc may include error handling with return statements
			code = fmt.Sprintf(`if vals := %s; len(vals) > 0 {
		payload.%s = make([]%s, 0, len(vals))
		for _, val := range vals {
			var parsed %s
			%s
			payload.%s = append(payload.%s, parsed)
		}
	}`, varName, fieldName, elementType, elementType,
				// Generate inline parsing code - the ParseFunc will assign to "parsed"
				// and may include error handling with return statements
				typeExtractor.ParseFunc("val", "parsed", false),
				fieldName, fieldName)
		} else {
			// Fallback: for unknown types, assign the string slice as-is
			// Users will need to handle conversion themselves in their handler
			code = fmt.Sprintf(`if vals := %s; len(vals) > 0 {
		payload.%s = vals
	}`, varName, fieldName)
		}
	}

	return code, imports
}

// GenerateDefaultValue generates code to set a default value
func GenerateDefaultValue(fieldName, defaultValue, typeName string) string {
	switch {
	case strings.HasPrefix(typeName, "int") || strings.HasPrefix(typeName, "uint"):
		return fmt.Sprintf(`payload.%s = %s`, fieldName, defaultValue)
	case strings.HasPrefix(typeName, "float"):
		return fmt.Sprintf(`payload.%s = %s`, fieldName, defaultValue)
	case typeName == "bool":
		return fmt.Sprintf(`payload.%s = %s`, fieldName, defaultValue)
	case typeName == "string":
		// Use strconv.Quote to properly escape the string value
		// This prevents code injection if the default value contains quotes or special characters
		escapedValue := strconv.Quote(defaultValue)
		return fmt.Sprintf(`payload.%s = %s`, fieldName, escapedValue)
	default:
		return fmt.Sprintf(`payload.%s = %s`, fieldName, defaultValue)
	}
}

// GetBaseType returns the base type without pointer or slice
func GetBaseType(field *parser.Field) string {
	typeName := field.Type
	if field.IsPointer {
		typeName = strings.TrimPrefix(typeName, "*")
	}
	if field.IsSlice {
		typeName = field.SliceType
	}
	return typeName
}

// IsIntType checks if the type is an integer type
func IsIntType(typeName string) bool {
	return typeName == "int" || typeName == "int8" || typeName == "int16" ||
		typeName == "int32" || typeName == "int64"
}

// IsUintType checks if the type is an unsigned integer type
func IsUintType(typeName string) bool {
	return typeName == "uint" || typeName == "uint8" || typeName == "uint16" ||
		typeName == "uint32" || typeName == "uint64"
}

// IsFloatType checks if the type is a float type
func IsFloatType(typeName string) bool {
	return typeName == "float32" || typeName == "float64"
}

// IsBoolType checks if the type is a boolean type
func IsBoolType(typeName string) bool {
	return typeName == "bool"
}

// IsStringType checks if the type is a string type
func IsStringType(typeName string) bool {
	return typeName == "string"
}
