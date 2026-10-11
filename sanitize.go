package apikit

import (
	"encoding"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"strings"
)

const maxSanitizeDepth = 10

// redacted replaces a value marked sensitive.
const redacted = "[REDACTED]"

// truncated replaces whatever lies deeper than maxSanitizeDepth. Returning the
// value itself there would log a nested secret in full.
const truncated = "[TRUNCATED]"

// Sanitize processes a value for safe logging by respecting struct field tags:
//   - `log:"-"` omits the field entirely
//   - `log:"sensitive"` replaces the value with "[REDACTED]"
//   - `json:"-"` omits the field too: a field kept off the wire is usually kept
//     off it for the same reason it should stay out of a log
//   - No tag passes the value through unchanged
//
// Structs are walked wherever they are: behind pointers and interfaces, in
// slices, arrays and map values. A value that renders itself — time.Time, a
// JSON marshaler, a fmt.Stringer, a slog.LogValuer — is kept as it is. For
// other non-struct types, the value is returned as-is.
// Map keys use the json tag name (falling back to the field name).
func Sanitize(v any) any {
	if v == nil {
		return nil
	}
	return sanitize(reflect.ValueOf(v), 0)
}

var (
	jsonMarshalerType   = reflect.TypeFor[json.Marshaler]()
	jsonMarshalerToType = reflect.TypeFor[json.MarshalerTo]()
	textMarshalerType   = reflect.TypeFor[encoding.TextMarshaler]()
	stringerType        = reflect.TypeFor[fmt.Stringer]()
	logValuerType       = reflect.TypeFor[slog.LogValuer]()
)

func sanitize(v reflect.Value, depth int) any {
	if depth > maxSanitizeDepth {
		return truncated
	}

	// Dereference pointers and interfaces
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if rendersItself(v.Type()) {
		return v.Interface()
	}

	switch v.Kind() {
	case reflect.Struct:
		return sanitizeStruct(v, depth)
	case reflect.Slice:
		if v.IsNil() {
			return nil
		}
		return sanitizeList(v, depth)
	case reflect.Array:
		return sanitizeList(v, depth)
	case reflect.Map:
		return sanitizeMap(v, depth)
	default:
		return v.Interface()
	}
}

// rendersItself reports whether a type decides its own representation. A
// struct like time.Time has only unexported fields, so walking it would log {}.
func rendersItself(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	for _, iface := range []reflect.Type{jsonMarshalerType, jsonMarshalerToType, textMarshalerType, stringerType, logValuerType} {
		if t.Implements(iface) || reflect.PointerTo(t).Implements(iface) {
			return true
		}
	}
	return false
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
		if logTag == "-" || field.Tag.Get("json") == "-" {
			continue
		}

		key := fieldKey(field)

		if logTag == "sensitive" {
			result[key] = redacted
			continue
		}

		// Handle embedded structs (anonymous fields), by value or by pointer,
		// unless a json name says they are a field of their own.
		if field.Anonymous && !hasJSONName(field) {
			if embedded, ok := embeddedStruct(v.Field(i)); ok {
				maps.Copy(result, sanitizeStruct(embedded, depth+1))
				continue
			}
		}

		result[key] = sanitize(v.Field(i), depth+1)
	}

	return result
}

// embeddedStruct returns the struct an anonymous field embeds, by value or
// through a non-nil pointer.
func embeddedStruct(v reflect.Value) (reflect.Value, bool) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
	}
	return v, v.Kind() == reflect.Struct && !rendersItself(v.Type())
}

func sanitizeList(v reflect.Value, depth int) any {
	result := make([]any, v.Len())
	for i := range v.Len() {
		result[i] = sanitize(v.Index(i), depth+1)
	}
	return result
}

func sanitizeMap(v reflect.Value, depth int) any {
	if v.IsNil() {
		return nil
	}
	result := make(map[string]any, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		result[fmt.Sprint(iter.Key().Interface())] = sanitize(iter.Value(), depth+1)
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

func hasJSONName(f reflect.StructField) bool {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	return name != ""
}
