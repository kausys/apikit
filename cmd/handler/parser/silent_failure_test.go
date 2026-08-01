package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parseSource(t *testing.T, content string) *ParseResult {
	t.Helper()
	file := filepath.Join(t.TempDir(), "handler.go")
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	result, err := New().ParseFile(file)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return result
}

// A payload type declared outside the handler's file used to produce a wrapper
// whose parse function was `return nil` — no extraction, no body unmarshal, no
// validation — which COMPILED and served requests with a zero payload. The
// lookup is file-scoped and cannot be made to work here, so the only safe
// outcome is refusing to generate.
func TestPayloadDeclaredElsewhereIsAnError(t *testing.T) {
	result := parseSource(t, `package test

import "context"

// apikit:handler
func (h *handler) createUser(ctx context.Context, req CreateUserRequest) (any, error) {
	return nil, nil
}
`)

	if len(result.Handlers) != 0 {
		t.Errorf("handler with an unresolvable payload must not be generated, got %d", len(result.Handlers))
	}
	if len(result.Errors) != 1 {
		t.Fatalf("want 1 error, got %d: %v", len(result.Errors), result.Errors)
	}
	if !strings.Contains(result.Errors[0], "CreateUserRequest") {
		t.Errorf("error should name the unresolved type, got %q", result.Errors[0])
	}
}

// A qualified type reduces to its bare name during lookup (`dto.Foo` -> `Foo`),
// so it is never found either — same silent wrapper, same refusal.
func TestQualifiedPayloadTypeIsAnError(t *testing.T) {
	result := parseSource(t, `package test

import (
	"context"

	"example.com/dto"
)

// apikit:handler
func (h *handler) createUser(ctx context.Context, req dto.CreateUserRequest) (any, error) {
	return nil, nil
}
`)

	if len(result.Errors) != 1 {
		t.Fatalf("want 1 error, got %d: %v", len(result.Errors), result.Errors)
	}
}

// An annotated method with a bad signature used to be dropped with a warning
// that only printed under --verbose, so the failure surfaced as an
// undefined-symbol error at the call site in another file.
func TestInvalidSignatureIsAnError(t *testing.T) {
	result := parseSource(t, `package test

// GetUserRequest is the payload
type GetUserRequest struct {
	ID string `+"`"+`path:"id"`+"`"+`
}

// apikit:handler
func (h *handler) getUser(req GetUserRequest) (any, error) {
	return nil, nil
}
`)

	if len(result.Handlers) != 0 {
		t.Errorf("handler with an invalid signature must not be generated, got %d", len(result.Handlers))
	}
	if len(result.Errors) != 1 {
		t.Fatalf("want 1 error, got %d: %v", len(result.Errors), result.Errors)
	}
	if !strings.Contains(result.Errors[0], "getUser") {
		t.Errorf("error should name the method, got %q", result.Errors[0])
	}
}

// An empty payload struct is legitimate — endpoints that take nothing still
// declare one — and must not be mistaken for an unresolved type.
func TestEmptyPayloadStructIsValid(t *testing.T) {
	result := parseSource(t, `package test

import "context"

// statsPayload takes nothing.
type statsPayload struct{}

// apikit:handler
func (h *handler) stats(ctx context.Context, p statsPayload) (any, error) {
	return nil, nil
}
`)

	if len(result.Errors) != 0 {
		t.Fatalf("empty payload struct must be accepted, got: %v", result.Errors)
	}
	if len(result.Handlers) != 1 {
		t.Fatalf("want 1 handler, got %d", len(result.Handlers))
	}
	if result.Handlers[0].Struct == nil {
		t.Error("empty struct should still resolve, not be nil")
	}
}

// A method without the annotation is not a handler, whatever its shape.
func TestUnannotatedFunctionIsIgnored(t *testing.T) {
	result := parseSource(t, `package test

func helper(x int) (any, error) { return nil, nil }
`)

	if len(result.Errors) != 0 || len(result.Handlers) != 0 {
		t.Errorf("unannotated function should be invisible, got errors=%v handlers=%d",
			result.Errors, len(result.Handlers))
	}
}
