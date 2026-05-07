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
