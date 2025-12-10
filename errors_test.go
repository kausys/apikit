package apikit

import (
	"net/http"
	"testing"
)

func TestBadRequest(t *testing.T) {
	err := BadRequest("invalid input")

	if err.Code != http.StatusBadRequest {
		t.Errorf("expected code %d, got %d", http.StatusBadRequest, err.Code)
	}
	if err.Message != "invalid input" {
		t.Errorf("expected message 'invalid input', got %q", err.Message)
	}
	if err.ErrorCode != http.StatusText(http.StatusBadRequest) {
		t.Errorf("expected error code %q, got %q", http.StatusText(http.StatusBadRequest), err.ErrorCode)
	}
}

func TestUnauthorized(t *testing.T) {
	err := Unauthorized("authentication required")

	if err.Code != http.StatusUnauthorized {
		t.Errorf("expected code %d, got %d", http.StatusUnauthorized, err.Code)
	}
	if err.Message != "authentication required" {
		t.Errorf("expected message 'authentication required', got %q", err.Message)
	}
}

func TestForbidden(t *testing.T) {
	err := Forbidden("access denied")

	if err.Code != http.StatusForbidden {
		t.Errorf("expected code %d, got %d", http.StatusForbidden, err.Code)
	}
	if err.Message != "access denied" {
		t.Errorf("expected message 'access denied', got %q", err.Message)
	}
}

func TestNotFound(t *testing.T) {
	err := NotFound("user")

	if err.Code != http.StatusNotFound {
		t.Errorf("expected code %d, got %d", http.StatusNotFound, err.Code)
	}
	expected := "user not found"
	if err.Message != expected {
		t.Errorf("expected message %q, got %q", expected, err.Message)
	}
}

func TestConflict(t *testing.T) {
	err := Conflict("resource already exists")

	if err.Code != http.StatusConflict {
		t.Errorf("expected code %d, got %d", http.StatusConflict, err.Code)
	}
	if err.Message != "resource already exists" {
		t.Errorf("expected message 'resource already exists', got %q", err.Message)
	}
}

func TestNotAcceptable(t *testing.T) {
	err := NotAcceptable("unsupported format")

	if err.Code != http.StatusNotAcceptable {
		t.Errorf("expected code %d, got %d", http.StatusNotAcceptable, err.Code)
	}
	if err.Message != "unsupported format" {
		t.Errorf("expected message 'unsupported format', got %q", err.Message)
	}
}

func TestUnprocessableEntity(t *testing.T) {
	err := UnprocessableEntity("validation failed")

	if err.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected code %d, got %d", http.StatusUnprocessableEntity, err.Code)
	}
	if err.Message != "validation failed" {
		t.Errorf("expected message 'validation failed', got %q", err.Message)
	}
}

func TestInternalError(t *testing.T) {
	err := InternalError("database connection failed")

	if err.Code != http.StatusInternalServerError {
		t.Errorf("expected code %d, got %d", http.StatusInternalServerError, err.Code)
	}
	if err.Message != "database connection failed" {
		t.Errorf("expected message 'database connection failed', got %q", err.Message)
	}
}

func TestNotImplemented(t *testing.T) {
	err := NotImplemented("feature not available")

	if err.Code != http.StatusNotImplemented {
		t.Errorf("expected code %d, got %d", http.StatusNotImplemented, err.Code)
	}
	if err.Message != "feature not available" {
		t.Errorf("expected message 'feature not available', got %q", err.Message)
	}
}

func TestServiceUnavailable(t *testing.T) {
	err := ServiceUnavailable("service temporarily down")

	if err.Code != http.StatusServiceUnavailable {
		t.Errorf("expected code %d, got %d", http.StatusServiceUnavailable, err.Code)
	}
	if err.Message != "service temporarily down" {
		t.Errorf("expected message 'service temporarily down', got %q", err.Message)
	}
}

func TestGatewayTimeout(t *testing.T) {
	err := GatewayTimeout("upstream timeout")

	if err.Code != http.StatusGatewayTimeout {
		t.Errorf("expected code %d, got %d", http.StatusGatewayTimeout, err.Code)
	}
	if err.Message != "upstream timeout" {
		t.Errorf("expected message 'upstream timeout', got %q", err.Message)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		code     int
		contains string
	}{
		{"ErrInternalServer", ErrInternalServer, http.StatusInternalServerError, "internal server error"},
		{"ErrNotImplemented", ErrNotImplemented, http.StatusNotImplemented, "not implemented"},
		{"ErrUnauthorized", ErrUnauthorized, http.StatusUnauthorized, "authentication required"},
		{"ErrForbidden", ErrForbidden, http.StatusForbidden, "access denied"},
		{"ErrInvalidRequest", ErrInvalidRequest, http.StatusBadRequest, "invalid request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("expected code %d, got %d", tt.code, tt.err.Code)
			}
			if tt.err.Message != tt.contains {
				t.Errorf("expected message %q, got %q", tt.contains, tt.err.Message)
			}
		})
	}
}
