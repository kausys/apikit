package validator

import (
	"fmt"
	"strings"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// ValidEnum is implemented by enum types that can validate their value.
type ValidEnum interface {
	IsValid() bool
}

// EnumValuer is optionally implemented by enums that can list their valid values.
// When implemented, validation error messages will include the allowed values.
type EnumValuer interface {
	Values() []string
}

func registerEnumValidation(v *validator.Validate, tr ut.Translator) {
	_ = v.RegisterValidation("validEnum", func(fl validator.FieldLevel) bool {
		if e, ok := fl.Field().Interface().(ValidEnum); ok {
			return e.IsValid()
		}
		return false
	})

	_ = v.RegisterTranslation("validEnum", tr, func(ut ut.Translator) error {
		return ut.Add("validEnum", "'{0}' must be a valid value", true)
		// overload: field + allowed values
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("validEnum", fe.Field())
		if e, ok := fe.Value().(EnumValuer); ok {
			return fmt.Sprintf("%s, allowed values [%s]", t, strings.Join(e.Values(), ", "))
		}
		return t
	})
}
