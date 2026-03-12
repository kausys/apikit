package parser

import (
	"go/ast"
	"slices"
)

// TagParser is the interface that all tag/directive parsers must implement.
// Parsers are responsible for:
// 1. Matching comments they can handle
// 2. Extracting values from comments
// 3. Applying values to target objects
type TagParser interface {
	// Name returns the unique name of this parser.
	Name() string

	// Contexts returns the contexts where this parser is applicable.
	// For example, a "Description" parser might work in both ContextRoute and ContextModel.
	Contexts() []Context

	// Matches checks if this parser can handle the given comment in the specified context.
	Matches(comment string, ctx Context) bool

	// Parse extracts the value from the comment group.
	// Returns the parsed value and any error encountered.
	Parse(comments *ast.CommentGroup, ctx Context) (any, error)

	// Apply applies the parsed value to the target object.
	// The target type depends on the context (e.g., *spec.Operation for ContextRoute).
	Apply(target any, value any, ctx Context) error
}

// SetterFunc is a function that applies a value to a target object.
// It's used to decouple parsing from applying values.
type SetterFunc func(target any, value any) error

// SetterMap maps contexts to their corresponding setter functions.
// This allows a single parser to have different behaviors for different contexts.
type SetterMap map[Context]SetterFunc

// ParserType indicates how the parser processes comments.
type ParserType int

const (
	// ParserTypeSingleLine parses a single line (e.g., "Summary: text").
	ParserTypeSingleLine ParserType = iota

	// ParserTypeMultiLine parses multiple lines (e.g., "Description:\n line1\n line2").
	ParserTypeMultiLine

	// ParserTypeYAML parses YAML content from comments.
	ParserTypeYAML

	// ParserTypeJSON parses JSON content from comments.
	ParserTypeJSON

	// ParserTypeList parses comma or space-separated lists.
	ParserTypeList

	// ParserTypeKeyValue parses key:value pairs.
	ParserTypeKeyValue
)

// BaseParser provides common functionality for all parsers.
// Embed this in your parser implementation to get default behaviors.
type BaseParser struct {
	name       string
	parserType ParserType
	contexts   []Context
	setters    SetterMap
}

// NewBaseParser creates a new BaseParser with the given configuration.
func NewBaseParser(name string, parserType ParserType, contexts []Context, setters SetterMap) BaseParser {
	return BaseParser{
		name:       name,
		parserType: parserType,
		contexts:   contexts,
		setters:    setters,
	}
}

// Name returns the parser name.
func (p *BaseParser) Name() string {
	return p.name
}

// Contexts returns the supported contexts.
func (p *BaseParser) Contexts() []Context {
	return p.contexts
}

// Type returns the parser type.
func (p *BaseParser) Type() ParserType {
	return p.parserType
}

// SupportsContext checks if the parser supports a specific context.
func (p *BaseParser) SupportsContext(ctx Context) bool {
	return slices.Contains(p.contexts, ctx)
}

// GetSetter returns the setter function for a specific context.
func (p *BaseParser) GetSetter(ctx Context) (SetterFunc, bool) {
	setter, ok := p.setters[ctx]
	return setter, ok
}

// ApplyWithSetter applies a value using the context's setter function.
func (p *BaseParser) ApplyWithSetter(target, value any, ctx Context) error {
	setter, ok := p.GetSetter(ctx)
	if !ok {
		return &ErrNoSetterForContext{
			ParserName: p.name,
			Context:    ctx,
		}
	}
	return setter(target, value)
}
