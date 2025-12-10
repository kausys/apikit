package validator

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	entranslations "github.com/go-playground/validator/v10/translations/en"
)

var (
	validate   *validator.Validate
	translator ut.Translator
	once       sync.Once
)

func init() {
	once.Do(initValidator)
}

func initValidator() {
	validate = validator.New()

	// Use JSON tag names in error messages instead of struct field names
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	// Initialize universal translator
	en := en.New()
	uni := ut.New(en, en)
	translator, _ = uni.GetTranslator("en")

	// Register default translations
	if err := entranslations.RegisterDefaultTranslations(validate, translator); err != nil {
		panic("failed to register default translations: " + err.Error())
	}
}

// Validate returns the validator instance
func Validate() *validator.Validate {
	return validate
}

// Translator returns the universal translator instance
func Translator() ut.Translator {
	return translator
}

// RegisterValidation registers a custom validation with translation
func RegisterValidation(f func(v *validator.Validate, translator ut.Translator)) {
	f(validate, translator)
}

// Struct validates a struct without context
func Struct(s any) error {
	if err := validate.Struct(s); err != nil {
		return FormatError(err)
	}
	return nil
}

// StructCtx validates a struct with context
func StructCtx(ctx context.Context, s any) error {
	if err := validate.StructCtx(ctx, s); err != nil {
		return FormatError(err)
	}
	return nil
}

// StructExceptCtx validates a struct with context, omitting specified fields
func StructExceptCtx(ctx context.Context, s any, omitField ...string) error {
	if err := validate.StructExceptCtx(ctx, s, omitField...); err != nil {
		return FormatError(err)
	}
	return nil
}

// FieldError represents a single field validation error
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError represents validation errors
type ValidationError struct {
	Message     string       `json:"message"`
	FieldErrors []FieldError `json:"fieldErrors"`
}

func (v ValidationError) Error() string {
	if len(v.FieldErrors) == 0 {
		return "validation error"
	}

	var messages []string
	for _, err := range v.FieldErrors {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(messages, "; "))
}

// FormatError formats validator errors using the universal translator
func FormatError(err error) error {
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	var fieldErrors []FieldError
	for _, e := range validationErrors {
		fieldErrors = append(fieldErrors, FieldError{
			Field:   e.Field(),
			Message: e.Translate(translator),
		})
	}

	return ValidationError{
		Message:     "Validation failed",
		FieldErrors: fieldErrors,
	}
}
