package validator

import (
	"context"
	"strings"
	"testing"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

type validationTestStruct struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0,lte=120"`
}

type validationTestStructWithJSON struct {
	UserName string `json:"userName" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type nestedAddress struct {
	Address1 string `json:"address1" validate:"required"`
	City     string `json:"city" validate:"required"`
}

type validationTestStructNested struct {
	Name        string        `json:"name" validate:"required"`
	Address     nestedAddress `json:"address" validate:"required"`
	BankAddress nestedAddress `json:"bankAddress" validate:"required"`
}

func TestValidate(t *testing.T) {
	v := Validate()
	if v == nil {
		t.Fatal("expected non-nil validator")
	}

	// Verify it's a validator.Validate instance
	if _, ok := any(v).(*validator.Validate); !ok {
		t.Error("expected *validator.Validate type")
	}
}

func TestTranslator(t *testing.T) {
	tr := Translator()
	if tr == nil {
		t.Fatal("expected non-nil translator")
	}
}

func TestStruct(t *testing.T) {
	testCases := []struct {
		name      string
		input     any
		shouldErr bool
	}{
		{
			name: "valid struct",
			input: validationTestStruct{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			shouldErr: false,
		},
		{
			name: "missing required field",
			input: validationTestStruct{
				Email: "john@example.com",
				Age:   30,
			},
			shouldErr: true,
		},
		{
			name: "invalid email",
			input: validationTestStruct{
				Name:  "John Doe",
				Email: "not-an-email",
				Age:   30,
			},
			shouldErr: true,
		},
		{
			name: "age out of range",
			input: validationTestStruct{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   150,
			},
			shouldErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := Struct(tt.input)

			if tt.shouldErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStructCtx(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     any
		shouldErr bool
	}{
		{
			name: "valid struct with context",
			input: validationTestStruct{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				Age:   25,
			},
			shouldErr: false,
		},
		{
			name: "invalid struct with context",
			input: validationTestStruct{
				Name: "Jane Doe",
				Age:  25,
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := StructCtx(ctx, tt.input)

			if tt.shouldErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStructExceptCtx(t *testing.T) {
	ctx := context.Background()

	// Struct with missing email, but we'll omit email validation
	input := validationTestStruct{
		Name: "John Doe",
		Age:  30,
	}

	// Should fail without omitting email
	err := StructCtx(ctx, input)
	if err == nil {
		t.Error("expected error without omitting email field")
	}

	// Should pass when omitting email
	err = StructExceptCtx(ctx, input, "Email")
	if err != nil {
		t.Errorf("unexpected error when omitting email: %v", err)
	}
}

func TestFormatError(t *testing.T) {
	// Test with nil error
	err := FormatError(nil)
	if err != nil {
		t.Errorf("expected nil for nil input, got %v", err)
	}

	// Test with non-validation error
	nonValErr := context.Canceled
	err = FormatError(nonValErr)
	if err != nonValErr {
		t.Errorf("expected same error for non-validation error, got %v", err)
	}

	// Test with validation error
	input := validationTestStruct{
		Name: "", // Missing required field
	}
	valErr := Struct(input)
	if valErr == nil {
		t.Fatal("expected validation error")
	}

	// Check that it's a ValidationError
	if _, ok := valErr.(ValidationError); !ok {
		t.Errorf("expected ValidationError type, got %T", valErr)
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		valErr   ValidationError
		expected string
	}{
		{
			name: "single field error",
			valErr: ValidationError{
				Message: "Validation failed",
				FieldErrors: []FieldError{
					{Field: "email", Message: "email is required"},
				},
			},
			expected: "validation failed: email: email is required",
		},
		{
			name: "multiple field errors",
			valErr: ValidationError{
				Message: "Validation failed",
				FieldErrors: []FieldError{
					{Field: "name", Message: "name is required"},
					{Field: "email", Message: "email must be valid"},
				},
			},
			expected: "validation failed: name: name is required; email: email must be valid",
		},
		{
			name: "no field errors",
			valErr: ValidationError{
				Message:     "Validation failed",
				FieldErrors: []FieldError{},
			},
			expected: "validation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.valErr.Error()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFieldError(t *testing.T) {
	// Test FieldError structure
	fe := FieldError{
		Field:   "email",
		Message: "must be a valid email",
	}

	if fe.Field != "email" {
		t.Errorf("expected field 'email', got %q", fe.Field)
	}
	if fe.Message != "must be a valid email" {
		t.Errorf("expected message 'must be a valid email', got %q", fe.Message)
	}
}

func TestJSONTagNames(t *testing.T) {
	// Test that validator uses JSON tag names in error messages
	input := validationTestStructWithJSON{
		UserName: "", // Missing required field
		Password: "short",
	}

	err := Struct(input)
	if err == nil {
		t.Fatal("expected validation error")
	}

	valErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	// Check that field names use JSON tags
	foundUserName := false
	foundPassword := false
	for _, fe := range valErr.FieldErrors {
		if fe.Field == "userName" {
			foundUserName = true
		}
		if fe.Field == "password" {
			foundPassword = true
		}
	}

	if !foundUserName {
		t.Error("expected field name 'userName' from JSON tag")
	}
	if !foundPassword {
		t.Error("expected field name 'password' from JSON tag")
	}
}

func TestRegisterValidation(t *testing.T) {
	// Test that RegisterValidation can be called
	called := false
	RegisterValidation(func(v *validator.Validate, tr ut.Translator) {
		called = true
	})

	if !called {
		t.Error("expected RegisterValidation callback to be called")
	}
}

func TestValidationError_Message(t *testing.T) {
	// Test that validation errors have proper message structure
	input := validationTestStruct{
		Name:  "",
		Email: "invalid",
	}

	err := Struct(input)
	if err == nil {
		t.Fatal("expected validation error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "validation failed") {
		t.Errorf("expected error message to contain 'validation failed', got %q", errMsg)
	}
}

func TestInit(t *testing.T) {
	// Test that init() properly initializes the validator
	// This is implicitly tested by other tests, but we can verify the state
	if validate == nil {
		t.Error("expected validate to be initialized")
	}
	if translator == nil {
		t.Error("expected translator to be initialized")
	}
}

func TestNestedStructFieldPaths(t *testing.T) {
	input := validationTestStructNested{
		Name:        "John",
		Address:     nestedAddress{}, // missing address1 and city
		BankAddress: nestedAddress{Address1: "123 Main St"}, // missing city
	}

	err := Struct(input)
	if err == nil {
		t.Fatal("expected validation error")
	}

	valErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	// Collect field paths
	fields := make(map[string]bool)
	for _, fe := range valErr.FieldErrors {
		fields[fe.Field] = true
	}

	// Should have nested paths like "address.address1", not flat "address1"
	expectedFields := []string{"address.address1", "address.city", "bankAddress.city"}
	for _, expected := range expectedFields {
		if !fields[expected] {
			t.Errorf("expected field path %q, got fields: %v", expected, fields)
		}
	}

	// Should NOT have flat field names for nested fields
	unexpectedFields := []string{"address1", "city"}
	for _, unexpected := range unexpectedFields {
		if fields[unexpected] {
			t.Errorf("should not have flat field path %q for nested field", unexpected)
		}
	}
}

func TestFlatStructFieldPathsUnchanged(t *testing.T) {
	// Ensure flat structs still return simple field names (no prefix)
	input := validationTestStruct{
		Name:  "",
		Email: "invalid",
		Age:   200,
	}

	err := Struct(input)
	if err == nil {
		t.Fatal("expected validation error")
	}

	valErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	fields := make(map[string]bool)
	for _, fe := range valErr.FieldErrors {
		fields[fe.Field] = true
	}

	if !fields["name"] {
		t.Errorf("expected flat field 'name', got fields: %v", fields)
	}
	if !fields["email"] {
		t.Errorf("expected flat field 'email', got fields: %v", fields)
	}
	if !fields["age"] {
		t.Errorf("expected flat field 'age', got fields: %v", fields)
	}
}

func TestMultipleValidationErrors(t *testing.T) {
	// Test struct with multiple validation errors
	input := validationTestStruct{
		Name:  "",          // required
		Email: "not-email", // invalid email
		Age:   200,         // out of range
	}

	err := Struct(input)
	if err == nil {
		t.Fatal("expected validation error")
	}

	valErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	// Should have at least 3 field errors
	if len(valErr.FieldErrors) < 3 {
		t.Errorf("expected at least 3 field errors, got %d", len(valErr.FieldErrors))
	}
}
