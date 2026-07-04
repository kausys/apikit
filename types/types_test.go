package types

import (
	"strings"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}

	// Should have built-in types registered
	all := r.All()
	if len(all) == 0 {
		t.Error("expected built-in types to be registered")
	}

	// Check for some expected built-in types
	expectedTypes := []string{
		"string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "bool", "time.Time", "uuid.UUID",
	}

	for _, typeName := range expectedTypes {
		if _, ok := r.Get(typeName); !ok {
			t.Errorf("expected built-in type %q to be registered", typeName)
		}
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	customExtractor := &Extractor{
		TypeName: "custom.Type",
		Import:   "example.com/custom",
		ParseFunc: func(varName, fieldName string, isPointer bool) string {
			return "custom code"
		},
		RequiresError: true,
	}

	r.Register(customExtractor)

	retrieved, ok := r.Get("custom.Type")
	if !ok {
		t.Fatal("expected custom type to be registered")
	}

	if retrieved.TypeName != "custom.Type" {
		t.Errorf("expected TypeName %q, got %q", "custom.Type", retrieved.TypeName)
	}
	if retrieved.Import != "example.com/custom" {
		t.Errorf("expected Import %q, got %q", "example.com/custom", retrieved.Import)
	}
	if !retrieved.RequiresError {
		t.Error("expected RequiresError to be true")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	// Get existing type
	extractor, ok := r.Get("string")
	if !ok {
		t.Fatal("expected to find string type")
	}
	if extractor.TypeName != "string" {
		t.Errorf("expected TypeName %q, got %q", "string", extractor.TypeName)
	}

	// Get non-existent type
	_, ok = r.Get("nonexistent.Type")
	if ok {
		t.Error("expected not to find non-existent type")
	}
}

func TestRegistry_All(t *testing.T) {
	r := NewRegistry()

	all := r.All()
	if len(all) == 0 {
		t.Error("expected All() to return extractors")
	}

	// Verify it's a copy (modifying it shouldn't affect the registry)
	all["test.Type"] = &Extractor{TypeName: "test.Type"}

	_, ok := r.Get("test.Type")
	if ok {
		t.Error("modifying All() result should not affect registry")
	}
}

func TestStringExtractor(t *testing.T) {
	r := NewRegistry()
	extractor, ok := r.Get("string")
	if !ok {
		t.Fatal("expected string extractor")
	}

	// Test non-pointer
	code := extractor.ParseFunc("value", "Name", false)
	if !strings.Contains(code, "payload.Name = value") {
		t.Errorf("expected simple assignment, got: %s", code)
	}

	// Test pointer
	code = extractor.ParseFunc("value", "Name", true)
	if !strings.Contains(code, "&val") {
		t.Errorf("expected pointer assignment, got: %s", code)
	}

	if extractor.RequiresError {
		t.Error("string extractor should not require error handling")
	}
}

func TestIntExtractor(t *testing.T) {
	r := NewRegistry()

	// bitSize must match the target type so strconv.ParseInt range-checks
	// the value before the cast (CodeQL go/incorrect-integer-conversion).
	intTypes := map[string]string{
		"int":   "0",
		"int8":  "8",
		"int16": "16",
		"int32": "32",
		"int64": "64",
	}
	for typeName, bitSize := range intTypes {
		t.Run(typeName, func(t *testing.T) {
			extractor, ok := r.Get(typeName)
			if !ok {
				t.Fatalf("expected %s extractor", typeName)
			}

			wantParse := "strconv.ParseInt(value, 10, " + bitSize + ")"

			// Test non-pointer
			code := extractor.ParseFunc("value", "Age", false)
			if !strings.Contains(code, wantParse) {
				t.Errorf("expected %q, got: %s", wantParse, code)
			}
			if !strings.Contains(code, "payload.Age") {
				t.Errorf("expected field assignment, got: %s", code)
			}

			// Test pointer
			code = extractor.ParseFunc("value", "Age", true)
			if !strings.Contains(code, wantParse) {
				t.Errorf("expected %q, got: %s", wantParse, code)
			}
			if !strings.Contains(code, "&val") {
				t.Errorf("expected pointer assignment, got: %s", code)
			}

			if !extractor.RequiresError {
				t.Error("int extractor should require error handling")
			}
		})
	}
}

func TestUintExtractor(t *testing.T) {
	r := NewRegistry()

	// bitSize must match the target type so strconv.ParseUint range-checks
	// the value before the cast (CodeQL go/incorrect-integer-conversion).
	uintTypes := map[string]string{
		"uint":   "0",
		"uint8":  "8",
		"uint16": "16",
		"uint32": "32",
		"uint64": "64",
	}
	for typeName, bitSize := range uintTypes {
		t.Run(typeName, func(t *testing.T) {
			extractor, ok := r.Get(typeName)
			if !ok {
				t.Fatalf("expected %s extractor", typeName)
			}

			wantParse := "strconv.ParseUint(value, 10, " + bitSize + ")"

			// Test non-pointer
			code := extractor.ParseFunc("value", "Count", false)
			if !strings.Contains(code, wantParse) {
				t.Errorf("expected %q, got: %s", wantParse, code)
			}

			// Test pointer
			code = extractor.ParseFunc("value", "Count", true)
			if !strings.Contains(code, wantParse) {
				t.Errorf("expected %q, got: %s", wantParse, code)
			}
			if !strings.Contains(code, "&val") {
				t.Errorf("expected pointer assignment, got: %s", code)
			}

			if !extractor.RequiresError {
				t.Error("uint extractor should require error handling")
			}
		})
	}
}

func TestFloatExtractor(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		typeName string
		bits     string
	}{
		{"float32", "32"},
		{"float64", "64"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			extractor, ok := r.Get(tt.typeName)
			if !ok {
				t.Fatalf("expected %s extractor", tt.typeName)
			}

			// Test non-pointer
			code := extractor.ParseFunc("value", "Price", false)
			if !strings.Contains(code, "strconv.ParseFloat") {
				t.Errorf("expected ParseFloat call, got: %s", code)
			}
			if !strings.Contains(code, tt.bits) {
				t.Errorf("expected bits %s, got: %s", tt.bits, code)
			}

			// Test pointer
			code = extractor.ParseFunc("value", "Price", true)
			if !strings.Contains(code, "&val") {
				t.Errorf("expected pointer assignment, got: %s", code)
			}

			if !extractor.RequiresError {
				t.Error("float extractor should require error handling")
			}
		})
	}
}

func TestBoolExtractor(t *testing.T) {
	r := NewRegistry()
	extractor, ok := r.Get("bool")
	if !ok {
		t.Fatal("expected bool extractor")
	}

	// Test non-pointer
	code := extractor.ParseFunc("value", "Active", false)
	if !strings.Contains(code, "strconv.ParseBool") {
		t.Errorf("expected ParseBool call, got: %s", code)
	}
	if !strings.Contains(code, "payload.Active") {
		t.Errorf("expected field assignment, got: %s", code)
	}

	// Test pointer
	code = extractor.ParseFunc("value", "Active", true)
	if !strings.Contains(code, "&val") {
		t.Errorf("expected pointer assignment, got: %s", code)
	}

	if !extractor.RequiresError {
		t.Error("bool extractor should require error handling")
	}
}

func TestTimeExtractor(t *testing.T) {
	r := NewRegistry()
	extractor, ok := r.Get("time.Time")
	if !ok {
		t.Fatal("expected time.Time extractor")
	}

	// Should have import
	if extractor.Import == "" {
		t.Error("expected time.Time to have import")
	}

	// Test non-pointer
	code := extractor.ParseFunc("value", "CreatedAt", false)
	if !strings.Contains(code, "apikit.NewTimeFromString") {
		t.Errorf("expected NewTimeFromString call, got: %s", code)
	}
	if !strings.Contains(code, "payload.CreatedAt") {
		t.Errorf("expected field assignment, got: %s", code)
	}

	// Test pointer
	code = extractor.ParseFunc("value", "CreatedAt", true)
	if !strings.Contains(code, "&t") {
		t.Errorf("expected pointer assignment, got: %s", code)
	}

	if !extractor.RequiresError {
		t.Error("time.Time extractor should require error handling")
	}
}

func TestUUIDExtractor(t *testing.T) {
	r := NewRegistry()
	extractor, ok := r.Get("uuid.UUID")
	if !ok {
		t.Fatal("expected uuid.UUID extractor")
	}

	// Should import google/uuid
	if extractor.Import != "github.com/google/uuid" {
		t.Errorf("expected github.com/google/uuid import, got: %q", extractor.Import)
	}

	// Non-pointer: parses via uuid.Parse into the field directly
	code := extractor.ParseFunc("value", "MemberID", false)
	if !strings.Contains(code, "uuid.Parse(value)") {
		t.Errorf("expected uuid.Parse call, got: %s", code)
	}
	if !strings.Contains(code, "payload.MemberID = ") {
		t.Errorf("expected field assignment, got: %s", code)
	}
	// Must NOT emit the invalid direct cast uuid.UUID(value).
	if strings.Contains(code, "uuid.UUID(value)") {
		t.Errorf("must not cast string to uuid.UUID, got: %s", code)
	}

	// Pointer: assigns the address of the parsed value
	code = extractor.ParseFunc("value", "MemberID", true)
	if !strings.Contains(code, "&") {
		t.Errorf("expected pointer assignment, got: %s", code)
	}

	if !extractor.RequiresError {
		t.Error("uuid.UUID extractor should require error handling")
	}
}

func TestDefaultRegistry(t *testing.T) {
	// Test that DefaultRegistry is initialized
	if DefaultRegistry == nil {
		t.Fatal("expected DefaultRegistry to be initialized")
	}

	// Should have built-in types
	_, ok := DefaultRegistry.Get("string")
	if !ok {
		t.Error("expected DefaultRegistry to have built-in types")
	}
}

func TestGlobalRegister(t *testing.T) {
	// Test global Register function
	customExtractor := &Extractor{
		TypeName: "test.GlobalType",
		ParseFunc: func(varName, fieldName string, isPointer bool) string {
			return "test"
		},
	}

	Register(customExtractor)

	retrieved, ok := Get("test.GlobalType")
	if !ok {
		t.Fatal("expected custom type to be registered globally")
	}

	if retrieved.TypeName != "test.GlobalType" {
		t.Errorf("expected TypeName %q, got %q", "test.GlobalType", retrieved.TypeName)
	}
}
