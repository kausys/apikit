package apikit

import (
	"fmt"
	"net/http"
)

// ============================================================================
// 4xx Client Errors
// ============================================================================

// BadRequest creates a 400 error
func BadRequest(message string) *Error {
	return &Error{
		Code:      http.StatusBadRequest,
		ErrorCode: http.StatusText(http.StatusBadRequest),
		Message:   message,
	}
}

// Unauthorized creates a 401 error
func Unauthorized(message string) *Error {
	return &Error{
		Code:      http.StatusUnauthorized,
		ErrorCode: http.StatusText(http.StatusUnauthorized),
		Message:   message,
	}
}

// Forbidden creates a 403 error
func Forbidden(message string) *Error {
	return &Error{
		Code:      http.StatusForbidden,
		ErrorCode: http.StatusText(http.StatusForbidden),
		Message:   message,
	}
}

// NotFound creates a 404 error
func NotFound(resource string) *Error {
	return &Error{
		Code:      http.StatusNotFound,
		ErrorCode: http.StatusText(http.StatusNotFound),
		Message:   fmt.Sprintf("%s not found", resource),
	}
}

// TooManyRequests creates a 429 error
func TooManyRequests(message string) *Error {
	return &Error{
		Code:      http.StatusTooManyRequests,
		ErrorCode: http.StatusText(http.StatusTooManyRequests),
		Message:   message,
	}
}

// Conflict creates a 409 error
func Conflict(message string) *Error {
	return &Error{
		Code:      http.StatusConflict,
		ErrorCode: http.StatusText(http.StatusConflict),
		Message:   message,
	}
}

// NotAcceptable creates a 406 error
func NotAcceptable(message string) *Error {
	return &Error{
		Code:      http.StatusNotAcceptable,
		ErrorCode: http.StatusText(http.StatusNotAcceptable),
		Message:   message,
	}
}

// PayloadTooLarge creates a 413 error
func PayloadTooLarge(message string) *Error {
	return &Error{
		Code:      http.StatusRequestEntityTooLarge,
		ErrorCode: http.StatusText(http.StatusRequestEntityTooLarge),
		Message:   message,
	}
}

// UnprocessableEntity creates a 422 error
func UnprocessableEntity(message string) *Error {
	return &Error{
		Code:      http.StatusUnprocessableEntity,
		ErrorCode: http.StatusText(http.StatusUnprocessableEntity),
		Message:   message,
	}
}

// ============================================================================
// 5xx Server Errors
// ============================================================================

// InternalError creates a 500 error
func InternalError(message string) *Error {
	return &Error{
		Code:      http.StatusInternalServerError,
		ErrorCode: http.StatusText(http.StatusInternalServerError),
		Message:   message,
	}
}

// NotImplemented creates a 501 error
func NotImplemented(message string) *Error {
	return &Error{
		Code:      http.StatusNotImplemented,
		ErrorCode: http.StatusText(http.StatusNotImplemented),
		Message:   message,
	}
}

// ServiceUnavailable creates a 503 error
func ServiceUnavailable(message string) *Error {
	return &Error{
		Code:      http.StatusServiceUnavailable,
		ErrorCode: http.StatusText(http.StatusServiceUnavailable),
		Message:   message,
	}
}

// GatewayTimeout creates a 504 error
func GatewayTimeout(message string) *Error {
	return &Error{
		Code:      http.StatusGatewayTimeout,
		ErrorCode: http.StatusText(http.StatusGatewayTimeout),
		Message:   message,
	}
}

// ============================================================================
// Predefined Errors
// ============================================================================

// The predefined errors are shared by every request that returns one. A With*
// call on one of them changes a copy, and errors.Is still matches the copy
// against the original.
var (
	// ErrInternalServer is a generic internal server error
	ErrInternalServer = share(InternalError("internal server error"))

	// ErrSomethingWentWrong is a generic internal server error
	ErrSomethingWentWrong = share(InternalError("something went wrong"))

	// ErrNotImplemented indicates the feature is not implemented
	ErrNotImplemented = share(NotImplemented("not implemented"))

	// ErrUnauthorized indicates authentication is required
	ErrUnauthorized = share(Unauthorized("authentication required"))

	// ErrForbidden indicates access is denied
	ErrForbidden = share(Forbidden("access denied"))

	// ErrInvalidRequest indicates the request is invalid
	ErrInvalidRequest = share(BadRequest("invalid request"))

	// ErrTooManyRequests indicates the client has exceeded the rate limit
	ErrTooManyRequests = share(TooManyRequests("too many requests"))
)
