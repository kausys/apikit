package apikit

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewError(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		message string
	}{
		{"basic error", 400, "bad request"},
		{"server error", 500, "internal error"},
		{"empty message", 404, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.code, tt.message)
			if err.Code != tt.code {
				t.Errorf("expected code %d, got %d", tt.code, err.Code)
			}
			if err.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, err.Message)
			}
			if err.cause != nil {
				t.Errorf("expected nil cause, got %v", err.cause)
			}
		})
	}
}

func TestNewErrorf(t *testing.T) {
	err := NewErrorf(400, "invalid value: %d", 42)
	expected := "invalid value: 42"
	if err.Message != expected {
		t.Errorf("expected message %q, got %q", expected, err.Message)
	}
	if err.Code != 400 {
		t.Errorf("expected code 400, got %d", err.Code)
	}
}

func TestNewErrorWithDetails(t *testing.T) {
	details := map[string]string{"field": "email", "reason": "invalid format"}
	err := NewErrorWithDetails(422, "validation failed", details)

	if err.Code != 422 {
		t.Errorf("expected code 422, got %d", err.Code)
	}
	if err.Message != "validation failed" {
		t.Errorf("expected message 'validation failed', got %q", err.Message)
	}
	if err.Details == nil {
		t.Fatal("expected details to be set")
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	err := WrapError(500, "wrapped error", originalErr)

	if err.Code != 500 {
		t.Errorf("expected code 500, got %d", err.Code)
	}
	if err.Message != "wrapped error" {
		t.Errorf("expected message 'wrapped error', got %q", err.Message)
	}
	if err.cause != originalErr {
		t.Errorf("expected cause to be original error")
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "without cause",
			err:      NewError(400, "bad request"),
			expected: "bad request",
		},
		{
			name:     "with cause",
			err:      WrapError(500, "server error", errors.New("database connection failed")),
			expected: "server error: database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestError_StatusCode(t *testing.T) {
	err := NewError(404, "not found")
	if err.StatusCode() != 404 {
		t.Errorf("expected status code 404, got %d", err.StatusCode())
	}
}

func TestError_Unwrap(t *testing.T) {
	originalErr := errors.New("original")
	err := WrapError(500, "wrapped", originalErr)

	unwrapped := err.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("expected unwrapped error to be original error")
	}

	// Test error without cause
	errNoCause := NewError(400, "no cause")
	if errNoCause.Unwrap() != nil {
		t.Errorf("expected nil unwrap for error without cause")
	}
}

func TestError_WithDetails(t *testing.T) {
	err := NewError(400, "bad request")
	details := map[string]int{"count": 5}

	result := err.WithDetails(details)

	if result.Details == nil {
		t.Fatal("expected details to be set")
	}
	if result != err {
		t.Error("expected WithDetails to return same error instance")
	}
}

func TestError_WithRequestID(t *testing.T) {
	err := NewError(500, "server error")
	requestID := "req-123-456"

	result := err.WithRequestID(requestID)

	if result.RequestID != requestID {
		t.Errorf("expected request ID %q, got %q", requestID, result.RequestID)
	}
	if result != err {
		t.Error("expected WithRequestID to return same error instance")
	}
}

func TestError_WithCause(t *testing.T) {
	err := NewError(500, "server error")
	cause := errors.New("database error")

	result := err.WithCause(cause)

	if result.cause != cause {
		t.Errorf("expected cause to be set")
	}
	if result != err {
		t.Error("expected WithCause to return same error instance")
	}
}

func TestError_Chaining(t *testing.T) {
	// Test method chaining
	details := map[string]string{"key": "value"}
	err := NewError(400, "bad request").
		WithDetails(details).
		WithRequestID("req-123").
		WithCause(errors.New("underlying error"))

	if err.Details == nil {
		t.Error("expected details to be set")
	}
	if err.RequestID != "req-123" {
		t.Errorf("expected request ID 'req-123', got %q", err.RequestID)
	}
	if err.cause == nil {
		t.Error("expected cause to be set")
	}
}

func TestError_ErrorCodeField(t *testing.T) {
	err := &Error{
		Code:      400,
		ErrorCode: "BAD_REQUEST",
		Message:   "invalid input",
	}

	if err.ErrorCode != "BAD_REQUEST" {
		t.Errorf("expected error code 'BAD_REQUEST', got %q", err.ErrorCode)
	}
}

func TestError_ErrorsIs(t *testing.T) {
	// Test that errors.Is works with wrapped errors
	originalErr := errors.New("original")
	wrappedErr := WrapError(500, "wrapped", originalErr)

	if !errors.Is(wrappedErr, originalErr) {
		t.Error("expected errors.Is to work with wrapped errors")
	}
}

func TestError_ErrorsAs(t *testing.T) {
	// Test that errors.As works with Error type
	err := NewError(404, "not found")
	var apiErr *Error

	if !errors.As(err, &apiErr) {
		t.Error("expected errors.As to work with Error type")
	}
	if apiErr.Code != 404 {
		t.Errorf("expected code 404, got %d", apiErr.Code)
	}
}

func TestError_NilCause(t *testing.T) {
	// Test that WithCause with nil doesn't panic
	err := NewError(500, "error")
	result := err.WithCause(nil)

	if result.cause != nil {
		t.Error("expected cause to be nil")
	}
}

func TestError_FormattedError(t *testing.T) {
	// Test error formatting with fmt
	err := NewError(400, "bad request")
	formatted := fmt.Sprintf("%v", err)
	if formatted != "bad request" {
		t.Errorf("expected 'bad request', got %q", formatted)
	}

	// Test with cause
	wrappedErr := WrapError(500, "server error", errors.New("db error"))
	formatted = fmt.Sprintf("%v", wrappedErr)
	expected := "server error: db error"
	if formatted != expected {
		t.Errorf("expected %q, got %q", expected, formatted)
	}
}
