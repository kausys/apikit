package validator

import (
	"errors"
	"testing"
)

// testStatus is a test enum type.
type testStatus string

const (
	testStatusActive   testStatus = "active"
	testStatusInactive testStatus = "inactive"
)

func (s testStatus) IsValid() bool {
	switch s {
	case testStatusActive, testStatusInactive:
		return true
	}
	return false
}

func (s testStatus) Values() []string {
	return []string{string(testStatusActive), string(testStatusInactive)}
}

type enumTestStruct struct {
	Status testStatus `json:"status" validate:"required,validEnum"`
}

func TestValidEnum_Valid(t *testing.T) {
	input := enumTestStruct{Status: testStatusActive}

	err := Struct(input)
	if err != nil {
		t.Errorf("expected no error for valid enum, got: %v", err)
	}
}

func TestValidEnum_Invalid(t *testing.T) {
	input := enumTestStruct{Status: "unknown"}

	err := Struct(input)
	if err == nil {
		t.Fatal("expected validation error for invalid enum")
	}

	var valErr ValidationError
	ok := errors.As(err, &valErr)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if len(valErr.FieldErrors) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(valErr.FieldErrors))
	}

	fe := valErr.FieldErrors[0]
	if fe.Field != "status" {
		t.Errorf("expected field 'status', got %q", fe.Field)
	}

	expected := "'status' must be a valid value, allowed values [active, inactive]"
	if fe.Message != expected {
		t.Errorf("expected %q, got %q", expected, fe.Message)
	}
}

func TestValidEnum_Empty(t *testing.T) {
	input := enumTestStruct{Status: ""}

	err := Struct(input)
	if err == nil {
		t.Fatal("expected validation error for empty enum")
	}
}
