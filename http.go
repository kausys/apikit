package apikit

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync/atomic"
)

// HttpResponse represents an HTTP response with status code, body, headers, and content type
type HttpResponse struct {
	StatusCode  int               `json:"statusCode"`
	Body        any               `json:"body"`
	Headers     map[string]string `json:"headers"`
	ContentType string            `json:"contentType"`
	// Cookies are emitted as individual Set-Cookie headers. Unlike Headers
	// (a map, one value per key), this allows multiple Set-Cookie headers in a
	// single response — e.g. an access cookie plus a refresh cookie.
	Cookies []*http.Cookie `json:"-"`
}

// NewHttpResponse creates a new HttpResponse with the given status code and body
func NewHttpResponse(statusCode int, body any) *HttpResponse {
	return &HttpResponse{
		StatusCode:  statusCode,
		Body:        body,
		ContentType: "application/json", // default
	}
}

// WithHeaders adds custom headers to the response
func (r *HttpResponse) WithHeaders(headers map[string]string) *HttpResponse {
	r.Headers = headers
	return r
}

// WithHeader adds a single header to the response
func (r *HttpResponse) WithHeader(key, value string) *HttpResponse {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
	return r
}

// WithCookie appends a Set-Cookie to the response. Multiple calls emit multiple
// Set-Cookie headers.
func (r *HttpResponse) WithCookie(cookie *http.Cookie) *HttpResponse {
	if cookie != nil {
		r.Cookies = append(r.Cookies, cookie)
	}
	return r
}

// GetCookie returns the named cookie's value, or "" if absent (mirrors r.Header.Get).
func GetCookie(r *http.Request, name string) string {
	if c, err := r.Cookie(name); err == nil {
		return c.Value
	}
	return ""
}

// WithContentType sets a custom content type
func (r *HttpResponse) WithContentType(contentType string) *HttpResponse {
	r.ContentType = contentType
	return r
}

// statusCoder interface for errors that include their own status code
type statusCoder interface {
	StatusCode() int
}

// ErrorRenderer turns a handler error into an HTTP response. Consumers register
// their own (e.g. an RFC 7807 problem+json contract) via SetErrorRenderer
// without changing generated code; the default reproduces apikit's built-in
// error JSON.
type ErrorRenderer interface {
	RenderError(ctx context.Context, w http.ResponseWriter, err error)
}

// defaultErrorRenderer is the built-in renderer: the status an error in the
// chain declares through StatusCode, else 500, and apikit's Error JSON.
type defaultErrorRenderer struct{}

func (defaultErrorRenderer) RenderError(ctx context.Context, w http.ResponseWriter, err error) {
	writeError(ctx, w, err, statusOf(err))
}

// rendererBox lets an interface value live in an atomic.Pointer.
type rendererBox struct{ ErrorRenderer }

// globalErrorRenderer is the package-level error renderer; empty means the
// built-in one. Swapped atomically, like the logger.
var globalErrorRenderer atomic.Pointer[rendererBox]

// SetErrorRenderer overrides how errors are written for all generated
// handlers. A nil renderer restores the built-in one.
func SetErrorRenderer(r ErrorRenderer) {
	if r == nil {
		r = defaultErrorRenderer{}
	}
	globalErrorRenderer.Store(&rendererBox{r})
}

// GetErrorRenderer returns the current error renderer.
func GetErrorRenderer() ErrorRenderer {
	if b := globalErrorRenderer.Load(); b != nil {
		return b.ErrorRenderer
	}
	return defaultErrorRenderer{}
}

// WriteJSON writes a JSON response with default 200 OK status
func WriteJSON(w http.ResponseWriter, data any) {
	body, err := marshalJSON(data)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

// writeJSONWithStatus writes a JSON response with a specific status code.
//
// The body is encoded before anything is written, so a value that cannot be
// encoded answers 500 instead of the intended status over a truncated body.
func writeJSONWithStatus(ctx context.Context, w http.ResponseWriter, status int, data any) {
	body, err := marshalJSON(data)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("encoding response: %w", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if bodyAllowed(status) {
		_, _ = w.Write(body)
	}
}

// internalMessage is what a client is told about a failure that is not its
// own. What actually failed — a driver error, a path, a stack of wrapping — is
// for the log, not for whoever sent the request.
const internalMessage = "internal server error"

// writeError writes an error response with the given status code (built-in format).
//
// An *Error in the chain is written as it is: its message was written for the
// client. Any other error is written in the same shape. Below 500 it keeps its
// message, which a type declaring its own status chose to show; from 500 up it
// says only that the server failed, and the error itself goes to the logger.
func writeError(ctx context.Context, w http.ResponseWriter, err error, status int) {
	payload, ok := errors.AsType[*Error](err)
	if !ok {
		message := err.Error()
		if status >= http.StatusInternalServerError {
			GetLogger().Error(ctx, "request failed", "status", status, "error", err)
			message = internalMessage
		}
		payload = &Error{Code: status, ErrorCode: http.StatusText(status), Message: message}
	}

	body, encErr := marshalJSON(payload)
	if encErr != nil {
		// Details the caller attached cannot be encoded: say what the status
		// says rather than nothing.
		GetLogger().Error(ctx, "failed to encode error response", "error", encErr)
		body, _ = marshalJSON(&Error{Code: status, ErrorCode: http.StatusText(status), Message: payload.Message})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if bodyAllowed(status) {
		_, _ = w.Write(body)
	}
}

// HandleError handles errors with custom status codes (no request context).
// Kept for backward compatibility; prefer HandleErrorCtx.
func HandleError(w http.ResponseWriter, err error) {
	HandleErrorCtx(context.Background(), w, err)
}

// HandleErrorCtx renders an error through the registered ErrorRenderer, passing
// the request context (so renderers can read request_id, trace, etc.).
func HandleErrorCtx(ctx context.Context, w http.ResponseWriter, err error) {
	GetErrorRenderer().RenderError(ctx, w, err)
}

// HandleResponse handles both the response and error from a handler (no request
// context). Kept for backward compatibility; prefer HandleResponseCtx.
func HandleResponse(w http.ResponseWriter, response any, err error) {
	HandleResponseCtx(context.Background(), w, response, err)
}

// HandleResponseCtx is the main entry point used by generated code: it renders
// the handler's error (via the registered ErrorRenderer) or its success
// response, with the request context available throughout.
func HandleResponseCtx(ctx context.Context, w http.ResponseWriter, response any, err error) {
	if err != nil {
		HandleErrorCtx(ctx, w, err)
		return
	}

	// Support both *HttpResponse (pointer) and HttpResponse (value)
	var httpResp *HttpResponse
	switch v := response.(type) {
	case *HttpResponse:
		httpResp = v
	case HttpResponse:
		httpResp = &v
	}
	if httpResp == nil {
		writeJSONWithStatus(ctx, w, http.StatusOK, response)
		return
	}

	writeHTTPResponse(ctx, w, httpResp)
}

// writeHTTPResponse writes a response a handler described with HttpResponse.
func writeHTTPResponse(ctx context.Context, w http.ResponseWriter, resp *HttpResponse) {
	status := resp.StatusCode
	if status == 0 {
		// A literal HttpResponse{Body: x} says nothing about the status;
		// http.ResponseWriter would panic on 0.
		status = http.StatusOK
	}

	contentType := resp.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	var body []byte
	if resp.Body != nil && bodyAllowed(status) {
		var err error
		if body, err = encodeBody(resp.Body, contentType); err != nil {
			writeError(ctx, w, fmt.Errorf("encoding response: %w", err), http.StatusInternalServerError)
			return
		}
	}

	for key, value := range resp.Headers {
		w.Header().Set(key, value)
	}
	for _, cookie := range resp.Cookies {
		if cookie != nil {
			http.SetCookie(w, cookie)
		}
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	if body != nil {
		_, _ = w.Write(body)
	}
}

// encodeBody encodes a response body for its content type: JSON for any JSON
// media type (application/json with parameters, application/problem+json), the
// bytes as they are for strings, byte slices and readers, and the value's
// default format otherwise.
func encodeBody(body any, contentType string) ([]byte, error) {
	if isJSON(contentType) {
		return marshalJSON(body)
	}
	switch v := body.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case io.Reader:
		return io.ReadAll(v)
	default:
		return fmt.Append(nil, v), nil
	}
}

// isJSON reports whether a content type is JSON: application/json, or any
// type with the +json structured suffix.
func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

// marshalJSON encodes a response with encoding/json/v2: a nil slice or map is
// [] or {} rather than null, `omitempty` omits empty JSON values (use
// `omitzero` for a zero number or false), strings are not HTML-escaped, and
// invalid UTF-8 is an error rather than silently replaced. Map keys are sorted,
// so the same value always encodes to the same bytes. The trailing newline is
// the one json.Encoder wrote, kept so the bytes on the wire stay as they were.
func marshalJSON(v any) ([]byte, error) {
	body, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

// bodyAllowed reports whether a response with this status may carry a body.
func bodyAllowed(status int) bool {
	switch {
	case status >= 100 && status < 200, status == http.StatusNoContent, status == http.StatusNotModified:
		return false
	default:
		return true
	}
}
