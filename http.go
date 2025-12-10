package apikit

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// HttpResponse represents an HTTP response with status code, body, headers, and content type
type HttpResponse struct {
	StatusCode  int               `json:"statusCode"`
	Body        any               `json:"body"`
	Headers     map[string]string `json:"headers"`
	ContentType string            `json:"contentType"`
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

// WithContentType sets a custom content type
func (r *HttpResponse) WithContentType(contentType string) *HttpResponse {
	r.ContentType = contentType
	return r
}

// statusCoder interface for errors that include their own status code
type statusCoder interface {
	StatusCode() int
}

// WriteJSON writes a JSON response with default 200 OK status
func WriteJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// writeJSONWithStatus writes a JSON response with a specific status code
func writeJSONWithStatus(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Status already written, can't change it
		return
	}
}

// writeError writes an error response with the given status code
func writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Check if it's the custom Error type
	if apiErr, ok := err.(*Error); ok {
		json.NewEncoder(w).Encode(apiErr)
		return
	}

	// Default error format
	json.NewEncoder(w).Encode(map[string]any{
		"error": err.Error(),
	})
}

// HandleError handles errors with custom status codes
func HandleError(w http.ResponseWriter, err error) {
	if sc, ok := err.(statusCoder); ok {
		writeError(w, err, sc.StatusCode())
		return
	}

	// Default to 500 Internal Server Error
	writeError(w, err, http.StatusInternalServerError)
}

// HandleResponse handles both the response and error from a handler
// This is the main function used by generated code
func HandleResponse(w http.ResponseWriter, response any, err error) {
	// Handle error first
	if err != nil {
		HandleError(w, err)
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
					w.Write([]byte(v))
				case []byte:
					w.Write(v)
				default:
					// Fallback to fmt.Fprint for other types
					fmt.Fprint(w, v)
				}
			}
		}
	} else {
		// Default: write JSON with 200 OK
		writeJSONWithStatus(w, http.StatusOK, response)
	}
}
