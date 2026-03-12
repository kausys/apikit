package tags

import (
	"fmt"

	"github.com/kausys/apikit/cmd/openapi/parser"
	"github.com/kausys/apikit/cmd/openapi/spec"
)

// NewExampleParser creates a parser for the Example directive.
func NewExampleParser() *SingleLineParser {
	return NewSingleLineParser("example", "example:", []parser.Context{
		parser.ContextModel,
		parser.ContextField,
	}, parser.SetterMap{
		parser.ContextModel: func(target any, value any) error {
			if schema, ok := target.(*spec.Schema); ok {
				schema.Examples = append(schema.Examples, value)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "example", ExpectedType: "*spec.Schema", ActualType: fmt.Sprintf("%T", target)}
		},
		parser.ContextField: func(target any, value any) error {
			if schema, ok := target.(*spec.Schema); ok {
				schema.Examples = append(schema.Examples, value)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "example", ExpectedType: "*spec.Schema", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewRequiredParser creates a parser for the Required directive.
func NewRequiredParser() *ListParser {
	return NewListParser("required", "required:", ",", []parser.Context{parser.ContextModel}, parser.SetterMap{
		parser.ContextModel: func(target any, value any) error {
			if schema, ok := target.(*spec.Schema); ok {
				if fields, ok := value.([]string); ok {
					schema.Required = append(schema.Required, fields...)
				}
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "required", ExpectedType: "*spec.Schema", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewMinLengthParser creates a parser for the MinLength directive.
func NewMinLengthParser() *SingleLineParser {
	return NewSingleLineParser("minLength", "minlength:", []parser.Context{parser.ContextField}, parser.SetterMap{
		parser.ContextField: func(target any, value any) error {
			if schema, ok := target.(*spec.Schema); ok {
				var minLen uint64
				_, _ = fmt.Sscanf(value.(string), "%d", &minLen)
				schema.MinLength = minLen
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "minLength", ExpectedType: "*spec.Schema", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewMaxLengthParser creates a parser for the MaxLength directive.
func NewMaxLengthParser() *SingleLineParser {
	return NewSingleLineParser("maxLength", "maxlength:", []parser.Context{parser.ContextField}, parser.SetterMap{
		parser.ContextField: func(target any, value any) error {
			if schema, ok := target.(*spec.Schema); ok {
				var maxLen uint64
				_, _ = fmt.Sscanf(value.(string), "%d", &maxLen)
				schema.MaxLength = &maxLen
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "maxLength", ExpectedType: "*spec.Schema", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewPatternParser creates a parser for the Pattern directive.
func NewPatternParser() *SingleLineParser {
	return NewSingleLineParser("pattern", "pattern:", []parser.Context{parser.ContextField}, parser.SetterMap{
		parser.ContextField: func(target any, value any) error {
			if schema, ok := target.(*spec.Schema); ok {
				schema.Pattern = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "pattern", ExpectedType: "*spec.Schema", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewFormatParser creates a parser for the Format directive.
func NewFormatParser() *SingleLineParser {
	return NewSingleLineParser("format", "format:", []parser.Context{parser.ContextField}, parser.SetterMap{
		parser.ContextField: func(target any, value any) error {
			if schema, ok := target.(*spec.Schema); ok {
				schema.Format = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "format", ExpectedType: "*spec.Schema", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewEnumParser creates a parser for the Enum directive.
func NewEnumParser() *ListParser {
	return NewListParser("enum", "enum:", ",", []parser.Context{parser.ContextField}, parser.SetterMap{
		parser.ContextField: func(target any, value any) error {
			if schema, ok := target.(*spec.Schema); ok {
				if values, ok := value.([]string); ok {
					for _, v := range values {
						schema.Enum = append(schema.Enum, v)
					}
				}
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "enum", ExpectedType: "*spec.Schema", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}
