package tags

import (
	"fmt"

	"github.com/kausys/apikit/openapi/parser"
	"github.com/kausys/apikit/openapi/spec"
)

// NewSummaryParser creates a parser for the Summary directive.
func NewSummaryParser() *SingleLineParser {
	return NewSingleLineParser("summary", "summary:", []parser.Context{parser.ContextRoute}, parser.SetterMap{
		parser.ContextRoute: func(target any, value any) error {
			if op, ok := target.(*spec.Operation); ok {
				op.Summary = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "summary", ExpectedType: "*spec.Operation", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewDescriptionParser creates a parser for the Description directive.
// It works in multiple contexts: meta, route, model, field.
func NewDescriptionParser() *MultiLineParser {
	return NewMultiLineParser("description", "description:", []parser.Context{
		parser.ContextMeta,
		parser.ContextRoute,
		parser.ContextModel,
		parser.ContextField,
	}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			if info, ok := target.(*spec.Info); ok {
				info.Description = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "description", ExpectedType: "*spec.Info", ActualType: fmt.Sprintf("%T", target)}
		},
		parser.ContextRoute: func(target any, value any) error {
			if op, ok := target.(*spec.Operation); ok {
				op.Description = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "description", ExpectedType: "*spec.Operation", ActualType: fmt.Sprintf("%T", target)}
		},
		parser.ContextModel: func(target any, value any) error {
			if schema, ok := target.(*spec.Schema); ok {
				schema.Description = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "description", ExpectedType: "*spec.Schema", ActualType: fmt.Sprintf("%T", target)}
		},
		parser.ContextField: func(target any, value any) error {
			if schema, ok := target.(*spec.Schema); ok {
				schema.Description = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "description", ExpectedType: "*spec.Schema", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewTagsParser creates a parser for the Tags directive.
func NewTagsParser() *ListParser {
	return NewListParser("tags", "tags:", ",", []parser.Context{parser.ContextRoute}, parser.SetterMap{
		parser.ContextRoute: func(target any, value any) error {
			if op, ok := target.(*spec.Operation); ok {
				if tags, ok := value.([]string); ok {
					op.Tags = append(op.Tags, tags...)
				}
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "tags", ExpectedType: "*spec.Operation", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewDeprecatedParser creates a parser for the Deprecated directive.
func NewDeprecatedParser() *SingleLineParser {
	return NewSingleLineParser("deprecated", "deprecated:", []parser.Context{parser.ContextRoute}, parser.SetterMap{
		parser.ContextRoute: func(target any, value any) error {
			if op, ok := target.(*spec.Operation); ok {
				op.Deprecated = value.(string) == "true" || value.(string) == ""
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "deprecated", ExpectedType: "*spec.Operation", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewOperationIDParser creates a parser for the OperationID directive.
func NewOperationIDParser() *SingleLineParser {
	return NewSingleLineParser("operationId", "operationid:", []parser.Context{parser.ContextRoute}, parser.SetterMap{
		parser.ContextRoute: func(target any, value any) error {
			if op, ok := target.(*spec.Operation); ok {
				op.OperationID = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "operationId", ExpectedType: "*spec.Operation", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}

// NewSecurityParser creates a parser for the Security directive.
func NewSecurityParser() *ListParser {
	return NewListParser("security", "security:", ",", []parser.Context{parser.ContextRoute}, parser.SetterMap{
		parser.ContextRoute: func(target any, value any) error {
			if op, ok := target.(*spec.Operation); ok {
				if schemes, ok := value.([]string); ok {
					for _, scheme := range schemes {
						op.Security = append(op.Security, &spec.SecurityRequirement{
							Requirements: map[string][]string{
								scheme: {},
							},
						})
					}
				}
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "security", ExpectedType: "*spec.Operation", ActualType: fmt.Sprintf("%T", target)}
		},
	})
}
