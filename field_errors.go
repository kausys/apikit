package apikit

import (
	"errors"

	"github.com/kausys/apikit/validator"
)

// FieldErrorsOf extracts per-field validation errors from an error chain, if the
// error is (or wraps) an apikit.Error whose Details were set to a
// []validator.FieldError (as the generated validation path does). Consumer error
// renderers use it to populate their own errors[] without depending on the
// internal detail type.
func FieldErrorsOf(err error) ([]validator.FieldError, bool) {
	if apiErr, ok := errors.AsType[*Error](err); ok {
		if fe, ok := apiErr.Details.([]validator.FieldError); ok {
			return fe, true
		}
	}
	return nil, false
}

// ValidationFailed turns a failed validation into the error the client gets:
// 422, with one entry per field when the validator reported them.
func ValidationFailed(err error) error {
	if valErr, ok := errors.AsType[validator.ValidationError](err); ok {
		return UnprocessableEntity(valErr.Message).WithDetails(valErr.FieldErrors)
	}
	return UnprocessableEntity("validation failed").WithCause(err)
}
