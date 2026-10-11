package apikit

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An *Error wrapped by the caller still answers its own status. The status
// used to come from a type assertion on the outer error, so the response said
// 500 while its body said 404.
func TestHandleError_WrappedErrorKeepsItsStatus(t *testing.T) {
	is := assert.New(t)
	w := httptest.NewRecorder()

	HandleError(w, fmt.Errorf("loading user: %w", NotFound("user")))

	is.Equal(http.StatusNotFound, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	is.InDelta(404, body["code"], 0)
	is.Equal("user not found", body["message"])
}

// A failure that is not an API error answers 500 with a generic message and
// goes to the logger: the driver's text, a file path or a query is not the
// client's business.
func TestHandleError_InternalErrorIsNotShown(t *testing.T) {
	is := assert.New(t)
	logs := &recordingLogger{}
	SetLogger(logs)
	t.Cleanup(func() { SetLogger(nil) })
	w := httptest.NewRecorder()

	HandleError(w, errors.New(`pq: relation "users" does not exist`))

	is.Equal(http.StatusInternalServerError, w.Code)
	is.NotContains(w.Body.String(), "relation")
	is.Contains(w.Body.String(), internalMessage)
	is.Equal([]string{"error"}, logs.levels())
}

// A With* call on a predefined error must not change it for every other
// request. It used to set the field on the shared value, so one request's
// request id or details showed up in the next one's response.
func TestSharedErrors_WithReturnsACopy(t *testing.T) {
	is := assert.New(t)

	got := ErrUnauthorized.WithRequestID("req-1").WithDetails("only this request")

	is.Empty(ErrUnauthorized.RequestID)
	is.Nil(ErrUnauthorized.Details)
	is.Equal("req-1", got.RequestID)
	require.ErrorIs(t, got, ErrUnauthorized, "a copy still matches the error it came from")
	require.NotErrorIs(t, BadRequest("x"), ErrInvalidRequest)
}

// The same, under the race detector: requests use the shared errors at once.
func TestSharedErrors_ConcurrentWithIsRaceFree(t *testing.T) {
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			_ = ErrForbidden.WithRequestID(strconv.Itoa(i)).WithCause(errors.New("x"))
		})
	}
	wg.Wait()
	assert.Empty(t, ErrForbidden.RequestID)
}

// An error built by the caller keeps the documented chaining semantics.
func TestError_WithOnOwnErrorChangesIt(t *testing.T) {
	err := BadRequest("x")
	_ = err.WithDetails("d")
	assert.Equal(t, "d", err.Details)
}

// A literal HttpResponse with no status means 200. net/http panics on
// WriteHeader(0), so it used to take the request down.
func TestHandleResponse_ZeroStatusIsOK(t *testing.T) {
	w := httptest.NewRecorder()

	require.NotPanics(t, func() {
		HandleResponse(w, &HttpResponse{Body: map[string]string{"ok": "yes"}}, nil)
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"ok":"yes"}`, w.Body.String())
}

// Any JSON media type is encoded as JSON. Only the exact "application/json"
// used to be, so a charset parameter or problem+json printed the Go value.
func TestHandleResponse_JSONMediaTypes(t *testing.T) {
	for _, ct := range []string{
		"application/json; charset=utf-8",
		"application/problem+json",
		"application/vnd.api+json",
	} {
		t.Run(ct, func(t *testing.T) {
			w := httptest.NewRecorder()

			HandleResponse(w, NewHttpResponse(http.StatusOK, map[string]int{"n": 1}).WithContentType(ct), nil)

			assert.JSONEq(t, `{"n":1}`, w.Body.String())
			assert.Equal(t, ct, w.Header().Get("Content-Type"))
		})
	}
}

// A body that cannot be encoded answers 500, not the intended status over a
// half-written body.
func TestHandleResponse_UnencodableBodyIs500(t *testing.T) {
	w := httptest.NewRecorder()

	HandleResponse(w, NewHttpResponse(http.StatusCreated, make(chan int)), nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), internalMessage)
}

// 204 carries no body, whatever the handler put in it.
func TestHandleResponse_NoContentHasNoBody(t *testing.T) {
	w := httptest.NewRecorder()

	HandleResponse(w, NewHttpResponse(http.StatusNoContent, map[string]string{"x": "y"}), nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestSetters_NilRestoresDefaults(t *testing.T) {
	SetLogger(nil)
	SetErrorRenderer(nil)

	assert.False(t, LoggerEnabled())
	assert.NotPanics(t, func() { HandleError(httptest.NewRecorder(), errors.New("x")) })
}

// A client's mistake is not the process failing: it goes to debug when the
// logger has it, nowhere otherwise, and the payload is never in the record.
func TestLogFailure_LevelFollowsWhoseFaultItWas(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/users", nil)
	cases := []struct {
		name string
		err  error
		want []string
	}{
		{"client error", BadRequest("bad"), []string{"debug"}},
		{"wrapped client error", fmt.Errorf("x: %w", UnprocessableEntity("bad")), []string{"debug"}},
		{"server error", InternalError("boom"), []string{"error"}},
		{"plain error", errors.New("boom"), []string{"error"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logs := &recordingLogger{}
			SetLogger(logs)
			t.Cleanup(func() { SetLogger(nil) })

			LogFailure(r, "CreateUser", "handler error", tc.err)

			assert.Equal(t, tc.want, logs.levels())
			assert.Contains(t, logs.lastArgs(), "CreateUser")
		})
	}
}

func TestLogFailure_ErrorOnlyLoggerSkipsClientErrors(t *testing.T) {
	logs := &errorOnlyLogger{}
	SetLogger(logs)
	t.Cleanup(func() { SetLogger(nil) })
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	LogFailure(r, "H", "failed to parse request", BadRequest("bad"))

	assert.Zero(t, logs.n)
}

func TestNewSlogLogger(t *testing.T) {
	var out strings.Builder
	l := NewSlogLogger(slog.New(slog.NewTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug})))

	l.Debug(context.Background(), "refused", "k", "v")
	l.Error(context.Background(), "failed")

	assert.Contains(t, out.String(), "level=DEBUG msg=refused k=v")
	assert.Contains(t, out.String(), "level=ERROR msg=failed")
}

type recordingLogger struct {
	mu      sync.Mutex
	records []string
	args    [][]any
}

func (l *recordingLogger) Error(_ context.Context, _ string, args ...any) { l.add("error", args) }
func (l *recordingLogger) Debug(_ context.Context, _ string, args ...any) { l.add("debug", args) }

func (l *recordingLogger) add(level string, args []any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, level)
	l.args = append(l.args, args)
}

func (l *recordingLogger) levels() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.records
}

func (l *recordingLogger) lastArgs() []any {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.args) == 0 {
		return nil
	}
	return l.args[len(l.args)-1]
}

type errorOnlyLogger struct{ n int }

func (l *errorOnlyLogger) Error(context.Context, string, ...any) { l.n++ }
