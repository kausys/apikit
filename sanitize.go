package apikit

import (
	"maps"
	"reflect"
	"strings"
)

const maxSanitizeDepth = 10

// Sanitize processes a value for safe logging by respecting struct field tags:
//   - `log:"-"` omits the field entirely
//   - `log:"sensitive"` replaces the value with "[REDACTED]"
//   - No tag passes the value through unchanged
//
// For non-struct types, the value is returned as-is.
// Map keys use the json tag name (falling back to the field name).
func Sanitize(v any) any {
	if v == nil {
		return nil
	}
	return sanitize(reflect.ValueOf(v), 0)
}

func sanitize(v reflect.Value, depth int) any {
	if depth > maxSanitizeDepth {
		return v.Interface()
	}

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		return sanitizeStruct(v, depth)
	case reflect.Slice, reflect.Array:
		return sanitizeSlice(v, depth)
	default:
		return v.Interface()
	}
}

func sanitizeStruct(v reflect.Value, depth int) map[string]any {
	t := v.Type()
	result := make(map[string]any, t.NumField())

	for i := range t.NumField() {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check log tag
		logTag := field.Tag.Get("log")
		if logTag == "-" {
			continue
		}

		key := fieldKey(field)

		if logTag == "sensitive" {
			result[key] = "[REDACTED]"
			continue
		}

		// Handle embedded structs (anonymous fields)
		if field.Anonymous && v.Field(i).Kind() == reflect.Struct {
			embedded := sanitizeStruct(v.Field(i), depth+1)
			maps.Copy(result, embedded)
			continue
		}

		result[key] = sanitize(v.Field(i), depth+1)
	}

	return result
}

func sanitizeSlice(v reflect.Value, depth int) any {
	if v.IsNil() {
		return nil
	}

	result := make([]any, v.Len())
	for i := range v.Len() {
		result[i] = sanitize(v.Index(i), depth+1)
	}
	return result
}

func fieldKey(f reflect.StructField) string {
	if jsonTag := f.Tag.Get("json"); jsonTag != "" {
		name, _, _ := strings.Cut(jsonTag, ",")
		if name != "" && name != "-" {
			return name
		}
	}
	return f.Name
}
