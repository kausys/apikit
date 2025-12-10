package apikit

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected string
	}{
		{
			name:     "simple object",
			data:     map[string]string{"message": "hello"},
			expected: `{"message":"hello"}`,
		},
		{
			name:     "array",
			data:     []int{1, 2, 3},
			expected: `[1,2,3]`,
		},
		{
			name:     "nil",
			data:     nil,
			expected: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSON(w, tt.data)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", contentType)
			}

			body := w.Body.String()
			// Normalize JSON for comparison
			var expected, actual any
			json.Unmarshal([]byte(tt.expected), &expected)
			json.Unmarshal([]byte(body), &actual)

			expectedJSON, _ := json.Marshal(expected)
			actualJSON, _ := json.Marshal(actual)

			if string(expectedJSON) != string(actualJSON) {
				t.Errorf("expected body %s, got %s", expectedJSON, actualJSON)
			}
		})
	}
}

func TestWriteJSONWithStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		data   any
	}{
		{"created", http.StatusCreated, map[string]string{"id": "123"}},
		{"accepted", http.StatusAccepted, map[string]string{"status": "pending"}},
		{"no content", http.StatusNoContent, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSONWithStatus(w, tt.status, tt.data)

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", contentType)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		status         int
		expectAPIError bool
	}{
		{
			name:           "API error",
			err:            NewError(400, "bad request"),
			status:         400,
			expectAPIError: true,
		},
		{
			name:           "standard error",
			err:            errors.New("something went wrong"),
			status:         500,
			expectAPIError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeError(w, tt.err, tt.status)

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", contentType)
			}

			var response map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if tt.expectAPIError {
				if _, ok := response["code"]; !ok {
					t.Error("expected 'code' field in API error response")
				}
				if _, ok := response["message"]; !ok {
					t.Error("expected 'message' field in API error response")
				}
			} else {
				if _, ok := response["error"]; !ok {
					t.Error("expected 'error' field in standard error response")
				}
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "API error with status code",
			err:            NewError(404, "not found"),
			expectedStatus: 404,
		},
		{
			name:           "standard error defaults to 500",
			err:            errors.New("generic error"),
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			HandleError(w, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandleResponse(t *testing.T) {
	tests := []struct {
		name           string
		response       any
		err            error
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "successful response",
			response:       map[string]string{"status": "ok"},
			err:            nil,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "error response",
			response:       nil,
			err:            NewError(400, "bad request"),
			expectedStatus: 400,
			expectError:    true,
		},
		{
			name:           "error takes precedence",
			response:       map[string]string{"data": "ignored"},
			err:            NewError(500, "server error"),
			expectedStatus: 500,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			HandleResponse(w, tt.response, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if tt.expectError {
				// Should have error fields
				if _, ok := response["code"]; !ok {
					t.Error("expected 'code' field in error response")
				}
			} else {
				// Should have success response
				if _, ok := response["status"]; !ok {
					t.Error("expected 'status' field in success response")
				}
			}
		})
	}
}

func TestStatusCoder(t *testing.T) {
	// Test that Error implements statusCoder interface
	var _ statusCoder = (*Error)(nil)

	err := NewError(418, "I'm a teapot")
	if err.StatusCode() != 418 {
		t.Errorf("expected status code 418, got %d", err.StatusCode())
	}
}

func TestWriteJSON_InvalidJSON(t *testing.T) {
	// Test with a type that can't be marshaled to JSON
	w := httptest.NewRecorder()
	invalidData := make(chan int) // channels can't be marshaled to JSON

	WriteJSON(w, invalidData)

	// Should write an error response
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 for invalid JSON, got %d", w.Code)
	}
}

func TestWriteJSONWithStatus_EncodingError(t *testing.T) {
	// Test that encoding errors are handled gracefully
	w := httptest.NewRecorder()
	invalidData := make(chan int)

	writeJSONWithStatus(w, http.StatusCreated, invalidData)

	// Status should already be written
	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestHttpResponse_NoBody(t *testing.T) {
	w := httptest.NewRecorder()
	resp := NewHttpResponse(http.StatusNoContent, nil)

	HandleResponse(w, resp, nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	if w.Body.Len() > 0 {
		t.Errorf("Expected empty body, got %s", w.Body.String())
	}
}

func TestHttpResponse_WithJSONBody(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "success"}
	resp := NewHttpResponse(http.StatusOK, data)

	HandleResponse(w, resp, nil)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("Expected message 'success', got '%s'", result["message"])
	}
}

func TestHttpResponse_WithHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	resp := NewHttpResponse(http.StatusCreated, map[string]string{"id": "123"}).
		WithHeader("Location", "/api/resource/123").
		WithHeader("X-Request-ID", "req-456")

	HandleResponse(w, resp, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	location := w.Header().Get("Location")
	if location != "/api/resource/123" {
		t.Errorf("Expected Location header '/api/resource/123', got '%s'", location)
	}

	requestID := w.Header().Get("X-Request-ID")
	if requestID != "req-456" {
		t.Errorf("Expected X-Request-ID header 'req-456', got '%s'", requestID)
	}
}

func TestHttpResponse_WithCustomContentType(t *testing.T) {
	w := httptest.NewRecorder()
	resp := NewHttpResponse(http.StatusOK, "Hello, World!").
		WithContentType("text/plain")

	HandleResponse(w, resp, nil)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}

	body := w.Body.String()
	if body != "Hello, World!" {
		t.Errorf("Expected body 'Hello, World!', got '%s'", body)
	}
}

func TestHttpResponse_WithBinaryData(t *testing.T) {
	w := httptest.NewRecorder()
	data := []byte{0x89, 0x50, 0x4E, 0x47}
	resp := NewHttpResponse(http.StatusOK, data).
		WithContentType("image/png")

	HandleResponse(w, resp, nil)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "image/png" {
		t.Errorf("Expected Content-Type image/png, got %s", contentType)
	}

	if len(w.Body.Bytes()) != len(data) {
		t.Errorf("Expected body length %d, got %d", len(data), len(w.Body.Bytes()))
	}
}

func TestHttpResponse_WithMultipleHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	headers := map[string]string{
		"X-Total-Count": "100",
		"X-Page":        "1",
	}
	resp := NewHttpResponse(http.StatusOK, []string{"item1", "item2"}).
		WithHeaders(headers)

	HandleResponse(w, resp, nil)

	if w.Header().Get("X-Total-Count") != "100" {
		t.Errorf("Expected X-Total-Count header '100', got '%s'", w.Header().Get("X-Total-Count"))
	}

	if w.Header().Get("X-Page") != "1" {
		t.Errorf("Expected X-Page header '1', got '%s'", w.Header().Get("X-Page"))
	}
}

func TestHttpResponse_ValueNotPointer(t *testing.T) {
	w := httptest.NewRecorder()

	// Test with HttpResponse value (not pointer)
	resp := HttpResponse{
		StatusCode:  http.StatusCreated,
		Body:        map[string]string{"id": "123"},
		ContentType: "application/json",
	}

	HandleResponse(w, resp, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("Expected id '123', got '%s'", result["id"])
	}
}

func TestHttpResponse_ValueWithHeaders(t *testing.T) {
	w := httptest.NewRecorder()

	// Test with HttpResponse value with headers
	resp := HttpResponse{
		StatusCode: http.StatusOK,
		Body:       map[string]string{"status": "ok"},
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
		ContentType: "application/json",
	}

	HandleResponse(w, resp, nil)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	customHeader := w.Header().Get("X-Custom-Header")
	if customHeader != "custom-value" {
		t.Errorf("Expected X-Custom-Header 'custom-value', got '%s'", customHeader)
	}
}

func TestHttpResponse_ValueNoBody(t *testing.T) {
	w := httptest.NewRecorder()

	// Test with HttpResponse value and no body
	resp := HttpResponse{
		StatusCode: http.StatusNoContent,
		Body:       nil,
	}

	HandleResponse(w, resp, nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	if w.Body.Len() > 0 {
		t.Errorf("Expected empty body, got %s", w.Body.String())
	}
}
