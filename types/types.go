// Package types provides a type system for converting string values
// to Go types during request parsing.
//
// It includes built-in extractors for common types and allows
// registration of custom extractors for domain-specific types.
package types

import (
	"fmt"
	"maps"
	"sync"
)

// Extractor defines how to convert a string value to a specific Go type.
//
// Example:
//
//	&Extractor{
//	    TypeName: "decimal.Decimal",
//	    Import:   "github.com/shopspring/decimal",
//	    ParseFunc: func(varName, fieldName string, isPointer bool) string {
//	        return fmt.Sprintf(`
//	            if d, err := decimal.NewFromString(%s); err == nil {
//	                payload.%s = d
//	            } else {
//	                return fmt.Errorf("invalid %s: %%w", err)
//	            }
//	        `, varName, fieldName, fieldName)
//	    },
//	}
type Extractor struct {
	// TypeName is the full type name (e.g., "decimal.Decimal", "time.Time")
	TypeName string

	// Import is the import path needed for this type (e.g., "github.com/shopspring/decimal")
	// Leave empty for built-in types
	Import string

	// ParseFunc generates the code to parse a string value into this type.
	// Parameters:
	//   - varName: the variable containing the string value
	//   - fieldName: the struct field name to assign to
	//   - isPointer: whether the field is a pointer type
	// Returns: Go code as a string
	ParseFunc func(varName, fieldName string, isPointer bool) string

	// RequiresError indicates if the parsing can fail and needs error handling
	RequiresError bool
}

// Registry holds all registered type extractors
type Registry struct {
	mu         sync.RWMutex
	extractors map[string]*Extractor
}

// NewRegistry creates a new type registry with built-in types registered
func NewRegistry() *Registry {
	r := &Registry{
		extractors: make(map[string]*Extractor),
	}
	r.registerBuiltins()
	return r
}

// Register adds a custom type extractor to the registry
func (r *Registry) Register(e *Extractor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.extractors[e.TypeName] = e
}

// Get retrieves an extractor for the given type name
func (r *Registry) Get(typeName string) (*Extractor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.extractors[typeName]
	return e, ok
}

// All returns all registered extractors
func (r *Registry) All() map[string]*Extractor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*Extractor, len(r.extractors))
	maps.Copy(result, r.extractors)
	return result
}

// registerBuiltins registers all built-in type extractors
func (r *Registry) registerBuiltins() {
	// String - no conversion needed
	r.Register(&Extractor{
		TypeName: "string",
		ParseFunc: func(varName, fieldName string, isPointer bool) string {
			if isPointer {
				return fmt.Sprintf("val := %s\npayload.%s = &val", varName, fieldName)
			}
			return fmt.Sprintf("payload.%s = %s", fieldName, varName)
		},
		RequiresError: false,
	})

	// Integer types
	for _, intType := range []string{"int", "int8", "int16", "int32", "int64"} {
		r.registerIntType(intType)
	}

	// Unsigned integer types
	for _, uintType := range []string{"uint", "uint8", "uint16", "uint32", "uint64"} {
		r.registerUintType(uintType)
	}

	// Float types
	r.registerFloatType("float32", 32)
	r.registerFloatType("float64", 64)

	// Bool
	r.Register(&Extractor{
		TypeName: "bool",
		ParseFunc: func(varName, fieldName string, isPointer bool) string {
			if isPointer {
				return fmt.Sprintf(`if b, err := strconv.ParseBool(%s); err == nil {
	val := b
	payload.%s = &val
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, fieldName, fieldName)
			}
			return fmt.Sprintf(`if b, err := strconv.ParseBool(%s); err == nil {
	payload.%s = b
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, fieldName, fieldName)
		},
		RequiresError: true,
	})

	// time.Time - supports multiple common formats using apikit.NewTimeFromString helper
	r.Register(&Extractor{
		TypeName: "time.Time",
		Import:   "github.com/kausys/apikit",
		ParseFunc: func(varName, fieldName string, isPointer bool) string {
			if isPointer {
				return fmt.Sprintf(`if t, err := apikit.NewTimeFromString(%s); err == nil {
	payload.%s = &t
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, fieldName, fieldName)
			}
			return fmt.Sprintf(`if t, err := apikit.NewTimeFromString(%s); err == nil {
	payload.%s = t
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, fieldName, fieldName)
		},
		RequiresError: true,
	})

	// uuid.UUID - parses the canonical string form via uuid.Parse. A direct
	// uuid.UUID(val) cast is invalid (the underlying type is [16]byte, not a
	// string), so a dedicated extractor is required for header/query/path UUID
	// params.
	r.Register(&Extractor{
		TypeName: "uuid.UUID",
		Import:   "github.com/google/uuid",
		ParseFunc: func(varName, fieldName string, isPointer bool) string {
			if isPointer {
				return fmt.Sprintf(`if u, err := uuid.Parse(%s); err == nil {
	payload.%s = &u
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, fieldName, fieldName)
			}
			return fmt.Sprintf(`if u, err := uuid.Parse(%s); err == nil {
	payload.%s = u
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, fieldName, fieldName)
		},
		RequiresError: true,
	})
}

// intBitSize returns the strconv bitSize argument that matches a Go integer
// type, so strconv.ParseInt/ParseUint range-checks the value before the
// generated code casts it. "0" maps to the platform-sized int/uint.
func intBitSize(typeName string) string {
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

func (r *Registry) registerIntType(typeName string) {
	bitSize := intBitSize(typeName)
	r.Register(&Extractor{
		TypeName: typeName,
		ParseFunc: func(varName, fieldName string, isPointer bool) string {
			if isPointer {
				return fmt.Sprintf(`if i, err := strconv.ParseInt(%s, 10, %s); err == nil {
	val := %s(i)
	payload.%s = &val
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, bitSize, typeName, fieldName, fieldName)
			}
			return fmt.Sprintf(`if i, err := strconv.ParseInt(%s, 10, %s); err == nil {
	payload.%s = %s(i)
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, bitSize, fieldName, typeName, fieldName)
		},
		RequiresError: true,
	})
}

func (r *Registry) registerUintType(typeName string) {
	bitSize := intBitSize(typeName)
	r.Register(&Extractor{
		TypeName: typeName,
		ParseFunc: func(varName, fieldName string, isPointer bool) string {
			if isPointer {
				return fmt.Sprintf(`if u, err := strconv.ParseUint(%s, 10, %s); err == nil {
	val := %s(u)
	payload.%s = &val
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, bitSize, typeName, fieldName, fieldName)
			}
			return fmt.Sprintf(`if u, err := strconv.ParseUint(%s, 10, %s); err == nil {
	payload.%s = %s(u)
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, bitSize, fieldName, typeName, fieldName)
		},
		RequiresError: true,
	})
}

func (r *Registry) registerFloatType(typeName string, bits int) {
	r.Register(&Extractor{
		TypeName: typeName,
		ParseFunc: func(varName, fieldName string, isPointer bool) string {
			if isPointer {
				return fmt.Sprintf(`if f, err := strconv.ParseFloat(%s, %d); err == nil {
	val := %s(f)
	payload.%s = &val
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, bits, typeName, fieldName, fieldName)
			}
			return fmt.Sprintf(`if f, err := strconv.ParseFloat(%s, %d); err == nil {
	payload.%s = %s(f)
} else {
	return fmt.Errorf("invalid %s: %%w", err)
}`, varName, bits, fieldName, typeName, fieldName)
		},
		RequiresError: true,
	})
}

// DefaultRegistry is the global registry instance
var DefaultRegistry = NewRegistry()

// Register adds a custom type extractor to the default registry
func Register(e *Extractor) {
	DefaultRegistry.Register(e)
}

// Get retrieves an extractor from the default registry
func Get(typeName string) (*Extractor, bool) {
	return DefaultRegistry.Get(typeName)
}
