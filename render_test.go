package apikit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kausys/apikit/validator"
)

type errorRendererFunc func(ctx context.Context, w http.ResponseWriter, err error)

func (f errorRendererFunc) RenderError(ctx context.Context, w http.ResponseWriter, err error) {
	f(ctx, w, err)
}

// With no renderer set, the default output must match the classic apikit shape
// (legacy backward-compat).
func TestHandleError_DefaultUnchanged(t *testing.T) {
	w := httptest.NewRecorder()
	HandleError(w, BadRequest("bad input"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["code"] != float64(400) || body["message"] != "bad input" {
		t.Fatalf("unexpected default error body: %v", body)
	}
}

// A registered renderer fully controls the error output and receives the ctx.
func TestSetErrorRenderer_OverridesAndReceivesCtx(t *testing.T) {
	t.Cleanup(func() { SetErrorRenderer(nil) })

	type ctxKey struct{}
	var gotReqID any
	SetErrorRenderer(errorRendererFunc(func(ctx context.Context, w http.ResponseWriter, _ error) {
		gotReqID = ctx.Value(ctxKey{}) // proves the renderer receives the request ctx
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"type":"about:blank"}`))
	}))

	w := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), ctxKey{}, "req-123")
	HandleErrorCtx(ctx, w, BadRequest("ignored"))

	if w.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418 (renderer should control it)", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", ct)
	}
	if gotReqID != "req-123" {
		t.Fatalf("renderer did not receive the request ctx (got %v)", gotReqID)
	}
}

// SetErrorRenderer(nil) restores the default behavior.
func TestSetErrorRenderer_NilRestoresDefault(t *testing.T) {
	SetErrorRenderer(errorRendererFunc(func(_ context.Context, w http.ResponseWriter, _ error) {
		w.WriteHeader(http.StatusTeapot)
	}))
	SetErrorRenderer(nil)

	w := httptest.NewRecorder()
	HandleError(w, NotFound("thing"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 after nil reset", w.Code)
	}
}

// HandleResponse (legacy, no ctx) still works for success.
func TestHandleResponse_LegacySuccess(t *testing.T) {
	w := httptest.NewRecorder()
	HandleResponse(w, map[string]string{"status": "ok"}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestFieldErrorsOf(t *testing.T) {
	fe := []validator.FieldError{{Field: "email", Message: "required"}}
	err := UnprocessableEntity("validation failed").WithDetails(fe)

	got, ok := FieldErrorsOf(err)
	if !ok || len(got) != 1 || got[0].Field != "email" {
		t.Fatalf("FieldErrorsOf = %v, %v", got, ok)
	}

	if _, ok := FieldErrorsOf(NotFound("x")); ok {
		t.Fatalf("FieldErrorsOf should be false for a non-validation error")
	}
}
