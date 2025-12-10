package apikit

import (
	"fmt"
)

// Error represents an API error with an HTTP status code
type Error struct {
	// HTTP status code
	Code int `json:"code"`

	// Semantic error code for client handling
	ErrorCode string `json:"errorCode,omitempty"`

	// Human-readable error message
	Message string `json:"message"`

	// Additional error details
	Details any `json:"details,omitempty"`

	// Request ID for correlation
	RequestID string `json:"requestId,omitempty"`

	// Original error (not serialized)
	cause error `json:"-"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

// StatusCode returns the HTTP status code for this error
func (e *Error) StatusCode() int {
	return e.Code
}

// Unwrap returns the underlying error for error chain support
func (e *Error) Unwrap() error {
	return e.cause
}

// WithDetails adds details to the error
func (e *Error) WithDetails(details any) *Error {
	e.Details = details
	return e
}

// WithRequestID adds request ID to the error
func (e *Error) WithRequestID(requestID string) *Error {
	e.RequestID = requestID
	return e
}

// WithCause wraps an underlying error
func (e *Error) WithCause(cause error) *Error {
	e.cause = cause
	return e
}

// NewError creates a new API error with the given status code and message
func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewErrorf creates a new API error with a formatted message
func NewErrorf(code int, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// NewErrorWithDetails creates a new API error with additional details
func NewErrorWithDetails(code int, message string, details any) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WrapError wraps an existing error with an API error
func WrapError(code int, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		cause:   cause,
	}
}
