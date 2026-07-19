package parser

import (
	"path/filepath"
	"testing"
)

func TestAnnotateTextUnmarshalers(t *testing.T) {
	filename := filepath.Join("testdata", "textunmarshaler", "handler.go")

	p := New()
	result, err := p.ParseFile(filename)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(result.Handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(result.Handlers))
	}

	if err := AnnotateTextUnmarshalers(filename, result); err != nil {
		t.Fatalf("AnnotateTextUnmarshalers: %v", err)
	}

	s := result.Handlers[0].Struct
	if s == nil {
		t.Fatal("expected handler struct")
	}

	byName := map[string]Field{}
	for _, f := range s.Fields {
		byName[f.Name] = f
	}

	id, ok := byName["ID"]
	if !ok {
		t.Fatal("missing ID field")
	}
	if !id.ImplementsTextUnmarshaler {
		t.Errorf("ID (TypedID): expected ImplementsTextUnmarshaler=true")
	}

	opt, ok := byName["OptionalID"]
	if !ok {
		t.Fatal("missing OptionalID field")
	}
	if !opt.ImplementsTextUnmarshaler {
		t.Errorf("OptionalID (*TypedID): expected ImplementsTextUnmarshaler=true")
	}

	status, ok := byName["Filter"]
	if !ok {
		t.Fatal("missing Filter field")
	}
	if status.ImplementsTextUnmarshaler {
		t.Errorf("Filter (Status string enum): expected ImplementsTextUnmarshaler=false")
	}
}

// Package has a deliberate undefined symbol (broken_ref.go). Annotation must
// still succeed — same situation as regenerating while Router refs *_apikit.go.
func TestAnnotateTextUnmarshalers_DespitePackageErrors(t *testing.T) {
	filename := filepath.Join("testdata", "textunmarshaler", "handler.go")
	p := New()
	result, err := p.ParseFile(filename)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if err := AnnotateTextUnmarshalers(filename, result); err != nil {
		t.Fatalf("AnnotateTextUnmarshalers: %v", err)
	}
	for _, f := range result.Handlers[0].Struct.Fields {
		if f.Name == "ID" && !f.ImplementsTextUnmarshaler {
			t.Fatalf("ID should be annotated despite package type errors")
		}
	}
}
