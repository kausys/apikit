// Package extractors provides framework-specific parameter extraction for HTTP handlers
package extractors

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/kausys/apikit/cmd/handler/parser"
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
// This is a generic helper to reduce code duplication.
//
// parsingFunc turns the string in "val" into an assignment to the field; nil
// assigns the string as it is. A default tag is the input used when the
// request leaves the value out, so it goes through the same parsing as a value
// that was sent: a default of the wrong form fails the request that relies on
// it instead of being pasted into the generated code as Go.
func GenerateExtractionCode(varName, fieldName, typeName string, field *parser.Field, parsingFunc func(string, string) string, imports []string) (string, []string) {
	source := varName
	if def := GetDefaultTag(field); def != "" {
		source = fmt.Sprintf("cmp.Or(%s, %s)", varName, strconv.Quote(def))
		imports = append(imports, "cmp")
	}

	assign := assignCode(fieldName, "val", field.IsPointer)
	if parsingFunc != nil {
		assign = parsingFunc("val", fieldName)
	}

	return fmt.Sprintf(`if val := %s; val != "" {
		%s
	}`, source, assign), imports
}

// assignCode assigns the value of a variable of the field's base type to the
// field, through a pointer when the field is one.
func assignCode(fieldName, value string, isPointer bool) string {
	if isPointer {
		return fmt.Sprintf(`parsed := %s
		payload.%s = &parsed`, value, fieldName)
	}
	return fmt.Sprintf(`payload.%s = %s`, fieldName, value)
}

// IntBitSize returns the strconv bitSize argument that matches a Go integer
// type. Using the matching bitSize makes strconv.ParseInt/ParseUint range-check
// the value so the subsequent cast to the target type cannot overflow on any
// platform. For platform-sized "int"/"uint" we return "0", which strconv
// interprets as the size of int/uint on the current platform.
func IntBitSize(typeName string) string {
	switch typeName {
	case "int8", "uint8":
		return "8"
	case "int16", "uint16":
		return "16"
	case "int32", "uint32":
		return "32"
	case "int64", "uint64":
		return "64"
	default: // "int", "uint"
		return "0"
	}
}

// GenerateIntParsing generates code to parse an integer from a string
func GenerateIntParsing(varName, fieldName, typeName string) string {
	return generateIntParsing(varName, fieldName, typeName, false)
}

func generateIntParsing(varName, fieldName, typeName string, isPointer bool) string {
	return fmt.Sprintf(`i, err := strconv.ParseInt(%s, 10, %s)
		if err != nil {
			return fmt.Errorf("invalid %s: %%w", err)
		}
		%s`, varName, IntBitSize(typeName), fieldName, assignCode(fieldName, typeName+"(i)", isPointer))
}

// GenerateUintParsing generates code to parse an unsigned integer from a string
func GenerateUintParsing(varName, fieldName, typeName string) string {
	return generateUintParsing(varName, fieldName, typeName, false)
}

func generateUintParsing(varName, fieldName, typeName string, isPointer bool) string {
	return fmt.Sprintf(`u, err := strconv.ParseUint(%s, 10, %s)
		if err != nil {
			return fmt.Errorf("invalid %s: %%w", err)
		}
		%s`, varName, IntBitSize(typeName), fieldName, assignCode(fieldName, typeName+"(u)", isPointer))
}

// GenerateFloatParsing generates code to parse a float from a string
func GenerateFloatParsing(varName, fieldName, bitSize string) string {
	return generateFloatParsing(varName, fieldName, bitSize, false)
}

func generateFloatParsing(varName, fieldName, bitSize string, isPointer bool) string {
	return fmt.Sprintf(`f, err := strconv.ParseFloat(%s, %s)
		if err != nil {
			return fmt.Errorf("invalid %s: %%w", err)
		}
		%s`, varName, bitSize, fieldName, assignCode(fieldName, "float"+bitSize+"(f)", isPointer))
}

// GenerateBoolParsing generates code to parse a boolean from a string
func GenerateBoolParsing(varName, fieldName string) string {
	return generateBoolParsing(varName, fieldName, false)
}

func generateBoolParsing(varName, fieldName string, isPointer bool) string {
	return fmt.Sprintf(`b, err := strconv.ParseBool(%s)
		if err != nil {
			return fmt.Errorf("invalid %s: %%w", err)
		}
		%s`, varName, fieldName, assignCode(fieldName, "b", isPointer))
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
		runes[0] += 32 // Convert to lowercase (A-Z to a-z)
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
	isPointer := field.IsPointer

	switch {
	case IsStringType(typeName):
		return GenerateExtractionCode(varName, fieldName, typeName, field, nil, imports)

	case IsIntType(typeName):
		imports = append(imports, "strconv")
		parsingFunc := func(v, f string) string { return generateIntParsing(v, f, typeName, isPointer) }
		return GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)

	case IsUintType(typeName):
		imports = append(imports, "strconv")
		parsingFunc := func(v, f string) string { return generateUintParsing(v, f, typeName, isPointer) }
		return GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)

	case IsFloatType(typeName):
		imports = append(imports, "strconv")
		bitSize := "64"
		if typeName == "float32" {
			bitSize = "32"
		}
		parsingFunc := func(v, f string) string { return generateFloatParsing(v, f, bitSize, isPointer) }
		return GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)

	case IsBoolType(typeName):
		imports = append(imports, "strconv")
		parsingFunc := func(v, f string) string { return generateBoolParsing(v, f, isPointer) }
		return GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)
	}

	// Check if there's a custom type extractor registered in the Type Registry
	if typeExtractor, ok := types.Get(typeName); ok {
		parsingFunc := func(v, f string) string {
			return typeExtractor.ParseFunc(v, f, isPointer)
		}
		if typeExtractor.Import != "" {
			imports = append(imports, typeExtractor.Import)
		}
		return GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)
	}

	if field.ImplementsTextUnmarshaler {
		// Types that implement encoding.TextUnmarshaler (e.g. typed IDs)
		parsingFunc := func(v, f string) string {
			return GenerateTextUnmarshalerParsing(v, f, typeName, isPointer)
		}
		return GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)
	}

	if field.IsEmbedded {
		// Embedded struct without extractor - skip (should have been expanded)
		return "", imports
	}

	// Fallback: for unknown custom types (e.g., enums), cast the string value
	// This handles types like model.AgentStatus, model.UserRole, etc.
	parsingFunc := func(v, f string) string {
		return assignCode(f, fmt.Sprintf("%s(%s)", typeName, v), isPointer)
	}
	return GenerateExtractionCode(varName, fieldName, typeName, field, parsingFunc, imports)
}

// GenerateSliceCodeByType generates code to parse a slice of values
// This handles the standard HTTP pattern: ?tags=go&tags=api&tags=http
// Returns: (code, imports)
func GenerateSliceCodeByType(varName, fieldName, elementType string, field *parser.Field) (string, []string) {
	var imports []string
	var parse string

	switch {
	case IsStringType(elementType):
		// For []string, direct assignment
		return fmt.Sprintf(`if vals := %s; len(vals) > 0 {
		payload.%s = vals
	}`, varName, fieldName), imports

	case IsIntType(elementType):
		imports = append(imports, "strconv")
		parse = fmt.Sprintf(`n, err := strconv.ParseInt(val, 10, %s)
			if err != nil {
				return fmt.Errorf("invalid %s[%%d]: %%w", i, err)
			}
			parsed := %s(n)`, IntBitSize(elementType), fieldName, elementType)

	case IsUintType(elementType):
		imports = append(imports, "strconv")
		parse = fmt.Sprintf(`n, err := strconv.ParseUint(val, 10, %s)
			if err != nil {
				return fmt.Errorf("invalid %s[%%d]: %%w", i, err)
			}
			parsed := %s(n)`, IntBitSize(elementType), fieldName, elementType)

	case IsFloatType(elementType):
		imports = append(imports, "strconv")
		bitSize := "64"
		if elementType == "float32" {
			bitSize = "32"
		}
		parse = fmt.Sprintf(`f, err := strconv.ParseFloat(val, %s)
			if err != nil {
				return fmt.Errorf("invalid %s[%%d]: %%w", i, err)
			}
			parsed := %s(f)`, bitSize, fieldName, elementType)

	case IsBoolType(elementType):
		imports = append(imports, "strconv")
		parse = fmt.Sprintf(`parsed, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid %s[%%d]: %%w", i, err)
			}`, fieldName)

	default:
		if typeExtractor, ok := types.Get(elementType); ok {
			// A registered ParseFunc assigns to payload.<field>. Each element
			// is parsed into a one-field struct that stands in for the payload
			// inside its own block, so the code it emits lands in "parsed".
			if typeExtractor.Import != "" {
				imports = append(imports, typeExtractor.Import)
			}
			parse = fmt.Sprintf(`_ = i
			var elem struct{ Value %s }
			{
				payload := &elem
				%s
			}
			parsed := elem.Value`, elementType, typeExtractor.ParseFunc("val", "Value", false))
		} else if field.ImplementsTextUnmarshaler {
			parse = fmt.Sprintf(`var parsed %s
			if err := parsed.UnmarshalText([]byte(val)); err != nil {
				return fmt.Errorf("invalid %s[%%d]: %%w", i, err)
			}`, elementType, fieldName)
		} else {
			// Fallback: for unknown named types (e.g., enums), cast each value.
			parse = fmt.Sprintf(`_ = i
			parsed := %s(val)`, elementType)
		}
	}

	return fmt.Sprintf(`if vals := %s; len(vals) > 0 {
		payload.%s = make([]%s, 0, len(vals))
		for i, val := range vals {
			%s
			payload.%s = append(payload.%s, parsed)
		}
	}`, varName, fieldName, elementType, parse, fieldName, fieldName), imports
}

// GenerateTextUnmarshalerParsing emits UnmarshalText assignment for path/query/header/form.
func GenerateTextUnmarshalerParsing(varName, fieldName, typeName string, isPointer bool) string {
	if isPointer {
		return fmt.Sprintf(`var parsed %s
	if err := parsed.UnmarshalText([]byte(%s)); err != nil {
		return fmt.Errorf("invalid %s: %%w", err)
	}
	payload.%s = &parsed`, typeName, varName, fieldName, fieldName)
	}
	return fmt.Sprintf(`var parsed %s
	if err := parsed.UnmarshalText([]byte(%s)); err != nil {
		return fmt.Errorf("invalid %s: %%w", err)
	}
	payload.%s = parsed`, typeName, varName, fieldName, fieldName)
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
