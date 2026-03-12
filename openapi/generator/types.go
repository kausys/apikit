package generator

import (
	"reflect"
	"sync"
)

// TypeInfo contains OpenAPI schema information for a custom type.
type TypeInfo struct {
	Type        string            // OpenAPI type (string, integer, number, boolean, object, array)
	Format      string            // OpenAPI format (uuid, date-time, decimal, etc.)
	Example     any               // Example value
	Default     any               // Default value
	Validations map[string]string // Additional validations (pattern, minLength, etc.)
}

// TypeHandler is a function that configures TypeInfo for a custom type.
type TypeHandler func(info *TypeInfo)

var (
	customTypes   = make(map[string]*TypeInfo)
	customTypesMu sync.RWMutex
)

// FieldType registers a custom type handler for OpenAPI schema generation.
// Pass a zero value of the type to register.
//
// Example:
//
//	generator.FieldType(decimal.Decimal{}, func(info *generator.TypeInfo) {
//	    info.Type = "string"
//	    info.Format = "decimal"
//	    info.Example = "100.00000000"
//	})
//
//	generator.FieldType(uuid.UUID{}, func(info *generator.TypeInfo) {
//	    info.Type = "string"
//	    info.Format = "uuid"
//	})
func FieldType(t any, handler TypeHandler) {
	typeName := getTypeName(t)
	RegisterType(typeName, handler)
}

// getTypeName returns the package-qualified type name (e.g., "decimal.Decimal").
func getTypeName(t any) string {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	pkgPath := rt.PkgPath()
	if pkgPath == "" {
		return rt.Name()
	}

	// Extract package name from path (e.g., "github.com/shopspring/decimal" -> "decimal")
	pkgName := rt.PkgPath()
	for i := len(pkgName) - 1; i >= 0; i-- {
		if pkgName[i] == '/' {
			pkgName = pkgName[i+1:]
			break
		}
	}

	return pkgName + "." + rt.Name()
}

// RegisterType registers a custom type handler by type name string.
// Prefer using FieldType with the actual type for type safety.
func RegisterType(typeName string, handler TypeHandler) {
	customTypesMu.Lock()
	defer customTypesMu.Unlock()

	info := &TypeInfo{
		Validations: make(map[string]string),
	}
	handler(info)
	customTypes[typeName] = info
}

// RegisterTypeInfo registers a TypeInfo directly for a custom type.
func RegisterTypeInfo(typeName string, info *TypeInfo) {
	customTypesMu.Lock()
	defer customTypesMu.Unlock()

	if info.Validations == nil {
		info.Validations = make(map[string]string)
	}
	customTypes[typeName] = info
}

// GetCustomType returns the TypeInfo for a registered custom type.
// Returns nil if the type is not registered.
func GetCustomType(typeName string) *TypeInfo {
	customTypesMu.RLock()
	defer customTypesMu.RUnlock()

	return customTypes[typeName]
}

// ClearCustomTypes removes all registered custom types.
// Useful for testing.
func ClearCustomTypes() {
	customTypesMu.Lock()
	defer customTypesMu.Unlock()

	customTypes = make(map[string]*TypeInfo)
}

// ResetToDefaults clears all custom types and re-registers the defaults.
// Useful for testing.
func ResetToDefaults() {
	ClearCustomTypes()
	registerDefaults()
}

// registerDefaults registers the default custom types.
func registerDefaults() {
	// uuid.UUID
	RegisterType("uuid.UUID", func(info *TypeInfo) {
		info.Type = "string"
		info.Format = "uuid"
	})

	// time.Time
	RegisterType("time.Time", func(info *TypeInfo) {
		info.Type = "string"
		info.Format = "date-time"
	})
}

//nolint:gochecknoinits // registers default type mappings at package init
func init() {
	registerDefaults()
}
