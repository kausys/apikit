package apikit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
// error JSON, so an unset renderer behaves exactly as before.
type ErrorRenderer interface {
	RenderError(ctx context.Context, w http.ResponseWriter, err error)
}

// defaultErrorRenderer is the built-in renderer (status from statusCoder, else
// 500) producing apikit's classic Error JSON.
type defaultErrorRenderer struct{}

func (defaultErrorRenderer) RenderError(ctx context.Context, w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if sc, ok := err.(statusCoder); ok {
		status = sc.StatusCode()
	}
	writeError(ctx, w, err, status)
}

// globalErrorRenderer is the package-level error renderer, defaults to built-in.
var globalErrorRenderer ErrorRenderer = defaultErrorRenderer{}

// SetErrorRenderer overrides how errors are written for all generated handlers.
func SetErrorRenderer(r ErrorRenderer) {
	if r == nil {
		r = defaultErrorRenderer{}
	}
	globalErrorRenderer = r
}

// GetErrorRenderer returns the current error renderer.
func GetErrorRenderer() ErrorRenderer { return globalErrorRenderer }

// WriteJSON writes a JSON response with default 200 OK status
func WriteJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// writeJSONWithStatus writes a JSON response with a specific status code
func writeJSONWithStatus(ctx context.Context, w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Status already written, can't change it; log for observability
		globalLogger.Error(ctx, "failed to encode JSON response", "error", err)
	}
}

// writeError writes an error response with the given status code (built-in format).
func writeError(ctx context.Context, w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Check if it's the custom Error type
	apiErr := &Error{}
	if errors.As(err, &apiErr) {
		if encErr := json.NewEncoder(w).Encode(apiErr); encErr != nil {
			globalLogger.Error(ctx, "failed to encode error response", "error", encErr)
		}
		return
	}

	// Default error format
	if encErr := json.NewEncoder(w).Encode(map[string]any{
		"error": err.Error(),
	}); encErr != nil {
		globalLogger.Error(ctx, "failed to encode error response", "error", encErr)
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
	globalErrorRenderer.RenderError(ctx, w, err)
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
	// Handle error first
	if err != nil {
		HandleErrorCtx(ctx, w, err)
		return
	}

	// Handle successful response
	// Support both *HttpResponse (pointer) and HttpResponse (value)
	var httpResp *HttpResponse
	if ptr, ok := response.(*HttpResponse); ok {
		httpResp = ptr
	} else if val, ok := response.(HttpResponse); ok {
		httpResp = &val
	}

	if httpResp != nil {
		// Set custom headers
		for key, value := range httpResp.Headers {
			w.Header().Set(key, value)
		}

		// Emit cookies as individual Set-Cookie headers (multiple allowed).
		for _, cookie := range httpResp.Cookies {
			if cookie != nil {
				http.SetCookie(w, cookie)
			}
		}

		// Set content type
		contentType := httpResp.ContentType
		if contentType == "" {
			contentType = "application/json"
		}
		w.Header().Set("Content-Type", contentType)

		// Write status code
		w.WriteHeader(httpResp.StatusCode)

		// Write body if present
		if httpResp.Body != nil {
			if contentType == "application/json" {
				if err := json.NewEncoder(w).Encode(httpResp.Body); err != nil {
					// Status already written, can't change it
					return
				}
			} else {
				// For non-JSON, write as string or bytes
				switch v := httpResp.Body.(type) {
				case string:
					_, _ = w.Write([]byte(v))
				case []byte:
					_, _ = w.Write(v)
				default:
					// Fallback to fmt.Fprint for other types
					_, _ = fmt.Fprint(w, v)
				}
			}
		}
	} else {
		// Default: write JSON with 200 OK
		writeJSONWithStatus(ctx, w, http.StatusOK, response)
	}
}
