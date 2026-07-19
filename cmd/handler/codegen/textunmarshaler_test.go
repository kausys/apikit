package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kausys/apikit/cmd/handler/parser"
)

func TestGenerate_TextUnmarshalerPathParam(t *testing.T) {
	filename := filepath.Join("..", "parser", "testdata", "textunmarshaler", "handler.go")

	p := parser.New()
	result, err := p.ParseFile(filename)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if err := parser.AnnotateTextUnmarshalers(filename, result); err != nil {
		t.Fatalf("AnnotateTextUnmarshalers: %v", err)
	}

	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	generated, err := g.Generate(result)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	code := string(generated)

	if !strings.Contains(code, "UnmarshalText([]byte(val))") {
		t.Errorf("expected UnmarshalText for TypedID path/query params, got:\n%s", code)
	}
	if strings.Contains(code, "TypedID(val)") {
		t.Errorf("did not expect TypedID(val) cast, got:\n%s", code)
	}
	// String enums still use cast
	if !strings.Contains(code, "payload.Filter = Status(val)") {
		t.Errorf("expected Status(val) cast for string enum, got:\n%s", code)
	}
}
