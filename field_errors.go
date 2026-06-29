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
	var apiErr *Error
	if errors.As(err, &apiErr) {
		if fe, ok := apiErr.Details.([]validator.FieldError); ok {
			return fe, true
		}
	}
	return nil, false
}
