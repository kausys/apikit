package extractors

import (
	"strings"
	"testing"

	"github.com/kausys/apikit/cmd/handler/parser"
)

func TestIntBitSize(t *testing.T) {
	cases := map[string]string{
		"int":    "0",
		"int8":   "8",
		"int16":  "16",
		"int32":  "32",
		"int64":  "64",
		"uint":   "0",
		"uint8":  "8",
		"uint16": "16",
		"uint32": "32",
		"uint64": "64",
	}
	for typeName, want := range cases {
		t.Run(typeName, func(t *testing.T) {
			if got := IntBitSize(typeName); got != want {
				t.Errorf("IntBitSize(%q) = %q, want %q", typeName, got, want)
			}
		})
	}
}

func TestGenerateIntParsing_BitSizeMatchesType(t *testing.T) {
	cases := map[string]string{
		"int":   "0",
		"int8":  "8",
		"int16": "16",
		"int32": "32",
		"int64": "64",
	}
	for typeName, bitSize := range cases {
		t.Run(typeName, func(t *testing.T) {
			code := GenerateIntParsing("val", "Field", typeName)
			want := "strconv.ParseInt(val, 10, " + bitSize + ")"
			if !strings.Contains(code, want) {
				t.Errorf("expected %q in generated code, got: %s", want, code)
			}
		})
	}
}

func TestGenerateUintParsing_BitSizeMatchesType(t *testing.T) {
	cases := map[string]string{
		"uint":   "0",
		"uint8":  "8",
		"uint16": "16",
		"uint32": "32",
		"uint64": "64",
	}
	for typeName, bitSize := range cases {
		t.Run(typeName, func(t *testing.T) {
			code := GenerateUintParsing("val", "Field", typeName)
			want := "strconv.ParseUint(val, 10, " + bitSize + ")"
			if !strings.Contains(code, want) {
				t.Errorf("expected %q in generated code, got: %s", want, code)
			}
		})
	}
}

func TestGenerateSliceCodeByType_IntBitSizeMatchesType(t *testing.T) {
	cases := map[string]string{
		"int":   "0",
		"int8":  "8",
		"int16": "16",
		"int32": "32",
		"int64": "64",
	}
	for elementType, bitSize := range cases {
		t.Run(elementType, func(t *testing.T) {
			code, _ := GenerateSliceCodeByType("vals", "Field", elementType, &parser.Field{})
			want := "strconv.ParseInt(val, 10, " + bitSize + ")"
			if !strings.Contains(code, want) {
				t.Errorf("expected %q in generated code, got: %s", want, code)
			}
		})
	}
}

func TestGenerateSliceCodeByType_UintBitSizeMatchesType(t *testing.T) {
	cases := map[string]string{
		"uint":   "0",
		"uint8":  "8",
		"uint16": "16",
		"uint32": "32",
		"uint64": "64",
	}
	for elementType, bitSize := range cases {
		t.Run(elementType, func(t *testing.T) {
			code, _ := GenerateSliceCodeByType("vals", "Field", elementType, &parser.Field{})
			want := "strconv.ParseUint(val, 10, " + bitSize + ")"
			if !strings.Contains(code, want) {
				t.Errorf("expected %q in generated code, got: %s", want, code)
			}
		})
	}
}

func TestGenerateCodeByType_TextUnmarshaler(t *testing.T) {
	field := &parser.Field{
		Name:                      "ID",
		Type:                      "TypedID",
		ImplementsTextUnmarshaler: true,
	}
	code, _ := GenerateCodeByType(`r.PathValue("id")`, "ID", "TypedID", field)
	if !strings.Contains(code, "UnmarshalText([]byte(val))") {
		t.Errorf("expected UnmarshalText in generated code, got: %s", code)
	}
	if strings.Contains(code, "TypedID(val)") {
		t.Errorf("did not expect string cast for TextUnmarshaler type, got: %s", code)
	}
}

func TestGenerateCodeByType_StringEnumCast(t *testing.T) {
	field := &parser.Field{
		Name: "Filter",
		Type: "Status",
	}
	code, _ := GenerateCodeByType(`r.URL.Query().Get("filter")`, "Filter", "Status", field)
	if !strings.Contains(code, "payload.Filter = Status(val)") {
		t.Errorf("expected string cast for enum, got: %s", code)
	}
	if strings.Contains(code, "UnmarshalText") {
		t.Errorf("did not expect UnmarshalText for string enum, got: %s", code)
	}
}

func TestGenerateCodeByType_TextUnmarshalerPointer(t *testing.T) {
	field := &parser.Field{
		Name:                      "OptionalID",
		Type:                      "*TypedID",
		IsPointer:                 true,
		ImplementsTextUnmarshaler: true,
	}
	code, _ := GenerateCodeByType(`r.URL.Query().Get("optionalId")`, "OptionalID", "TypedID", field)
	if !strings.Contains(code, "payload.OptionalID = &parsed") {
		t.Errorf("expected pointer assignment, got: %s", code)
	}
}

func TestGenerateSliceCodeByType_TextUnmarshaler(t *testing.T) {
	field := &parser.Field{
		Name:                      "IDs",
		Type:                      "[]TypedID",
		IsSlice:                   true,
		SliceType:                 "TypedID",
		ImplementsTextUnmarshaler: true,
	}
	code, _ := GenerateSliceCodeByType(`r.URL.Query()["ids"]`, "IDs", "TypedID", field)
	if !strings.Contains(code, "UnmarshalText([]byte(val))") {
		t.Errorf("expected UnmarshalText in slice code, got: %s", code)
	}
}
