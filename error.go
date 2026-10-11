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

	// shared marks an error every request returns, like ErrUnauthorized. A
	// With* call on one changes a copy: changing it in place would hand one
	// request's details to every other, and race while doing it.
	shared bool

	// origin is the shared error a copy was made from, so errors.Is still
	// matches it.
	origin *Error
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

// Is reports whether e was made from target by a With* call on a shared error.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && e.origin != nil && e.origin == t
}

// WithDetails adds details to the error
func (e *Error) WithDetails(details any) *Error {
	e = e.writable()
	e.Details = details
	return e
}

// WithRequestID adds request ID to the error
func (e *Error) WithRequestID(requestID string) *Error {
	e = e.writable()
	e.RequestID = requestID
	return e
}

// WithCause wraps an underlying error
func (e *Error) WithCause(cause error) *Error {
	e = e.writable()
	e.cause = cause
	return e
}

// writable returns e itself, or a copy of it when e is shared.
func (e *Error) writable() *Error {
	if !e.shared {
		return e
	}
	c := *e
	c.shared = false
	c.origin = e
	return &c
}

// share marks e as shared by every request (see Error.shared).
func share(e *Error) *Error {
	e.shared = true
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
