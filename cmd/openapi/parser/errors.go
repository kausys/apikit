package parser

import "fmt"

// ErrNoSetterForContext is returned when a parser doesn't have a setter for the given context.
type ErrNoSetterForContext struct {
	ParserName string
	Context    Context
}

func (e *ErrNoSetterForContext) Error() string {
	return fmt.Sprintf("parser %q has no setter for context %q", e.ParserName, e.Context)
}

// ErrInvalidTarget is returned when the target type doesn't match what the parser expects.
type ErrInvalidTarget struct {
	ParserName   string
	ExpectedType string
	ActualType   string
}

func (e *ErrInvalidTarget) Error() string {
	return fmt.Sprintf("parser %q expected target of type %s, got %s",
		e.ParserName, e.ExpectedType, e.ActualType)
}

// ErrParseFailure is returned when parsing fails.
type ErrParseFailure struct {
	ParserName string
	Context    Context
	Cause      error
}

func (e *ErrParseFailure) Error() string {
	return fmt.Sprintf("parser %q failed in context %q: %v", e.ParserName, e.Context, e.Cause)
}

func (e *ErrParseFailure) Unwrap() error {
	return e.Cause
}

// ErrParserNotFound is returned when no parser is registered for a directive.
type ErrParserNotFound struct {
	Directive Directive
}

func (e *ErrParserNotFound) Error() string {
	return fmt.Sprintf("no parsers registered for directive %q", e.Directive)
}
