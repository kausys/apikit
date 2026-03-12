package parser

import (
	"errors"
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Error Tests ====================

func TestErrNoSetterForContext(t *testing.T) {
	err := &ErrNoSetterForContext{
		ParserName: "TestParser",
		Context:    ContextRoute,
	}

	assert.Contains(t, err.Error(), "TestParser")
	assert.Contains(t, err.Error(), string(ContextRoute))
}

func TestErrInvalidTarget(t *testing.T) {
	err := &ErrInvalidTarget{
		ParserName:   "TestParser",
		ExpectedType: "*Operation",
		ActualType:   "string",
	}

	assert.Contains(t, err.Error(), "TestParser")
	assert.Contains(t, err.Error(), "*Operation")
	assert.Contains(t, err.Error(), "string")
}

func TestErrParseFailure(t *testing.T) {
	cause := errors.New("invalid format")
	err := &ErrParseFailure{
		ParserName: "TestParser",
		Context:    ContextModel,
		Cause:      cause,
	}

	assert.Contains(t, err.Error(), "TestParser")
	assert.Contains(t, err.Error(), string(ContextModel))
	assert.Contains(t, err.Error(), "invalid format")
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestErrParserNotFound(t *testing.T) {
	err := &ErrParserNotFound{
		Directive: DirectiveRoute,
	}

	assert.Contains(t, err.Error(), string(DirectiveRoute))
}

// ==================== BaseParser Tests ====================

func TestNewBaseParser(t *testing.T) {
	setters := SetterMap{
		ContextRoute: func(target any, value any) error { return nil },
	}

	bp := NewBaseParser("test", ParserTypeSingleLine, []Context{ContextRoute}, setters)

	assert.Equal(t, "test", bp.Name())
	assert.Equal(t, ParserTypeSingleLine, bp.Type())
	assert.Equal(t, []Context{ContextRoute}, bp.Contexts())
}

func TestBaseParserSupportsContext(t *testing.T) {
	bp := NewBaseParser("test", ParserTypeSingleLine, []Context{ContextRoute, ContextModel}, nil)

	assert.True(t, bp.SupportsContext(ContextRoute))
	assert.True(t, bp.SupportsContext(ContextModel))
	assert.False(t, bp.SupportsContext(ContextMeta))
}

func TestBaseParserGetSetter(t *testing.T) {
	setter := func(target any, value any) error { return nil }
	setters := SetterMap{ContextRoute: setter}
	bp := NewBaseParser("test", ParserTypeSingleLine, nil, setters)

	s, ok := bp.GetSetter(ContextRoute)
	assert.True(t, ok)
	assert.NotNil(t, s)

	_, ok = bp.GetSetter(ContextModel)
	assert.False(t, ok)
}

func TestBaseParserApplyWithSetter(t *testing.T) {
	var capturedValue any
	setter := func(target any, value any) error {
		capturedValue = value
		return nil
	}
	setters := SetterMap{ContextRoute: setter}
	bp := NewBaseParser("test", ParserTypeSingleLine, nil, setters)

	err := bp.ApplyWithSetter(nil, "test-value", ContextRoute)

	require.NoError(t, err)
	assert.Equal(t, "test-value", capturedValue)
}

func TestBaseParserApplyWithSetterNoSetter(t *testing.T) {
	bp := NewBaseParser("test", ParserTypeSingleLine, nil, nil)

	err := bp.ApplyWithSetter(nil, "value", ContextRoute)

	require.Error(t, err)
	var noSetterErr *ErrNoSetterForContext
	assert.ErrorAs(t, err, &noSetterErr)
}

// ==================== Registry Tests ====================

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()

	assert.NotNil(t, r)
	assert.Equal(t, 0, r.Count())
}

func TestGlobalRegistry(t *testing.T) {
	r := Global()

	assert.NotNil(t, r)
	assert.Same(t, globalRegistry, r)
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	parser := &mockParser{name: "test"}

	r.Register(DirectiveRoute, parser)
	parsers := r.GetParsers(DirectiveRoute)

	assert.Len(t, parsers, 1)
	assert.Equal(t, "test", parsers[0].Name())
}

func TestRegistryUnregister(t *testing.T) {
	r := NewRegistry()
	parser := &mockParser{name: "test"}
	r.Register(DirectiveRoute, parser)

	r.Unregister(DirectiveRoute, "test")
	parsers := r.GetParsers(DirectiveRoute)

	assert.Empty(t, parsers)
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry()
	r.Register(DirectiveRoute, &mockParser{name: "parser1"})
	r.Register(DirectiveRoute, &mockParser{name: "parser2"})
	r.Register(DirectiveModel, &mockParser{name: "parser3"})

	list := r.List()

	assert.Contains(t, list, DirectiveRoute)
	assert.Contains(t, list, DirectiveModel)
	assert.Len(t, list[DirectiveRoute], 2)
	assert.Len(t, list[DirectiveModel], 1)
}

func TestRegistryCount(t *testing.T) {
	r := NewRegistry()
	r.Register(DirectiveRoute, &mockParser{name: "p1"})
	r.Register(DirectiveRoute, &mockParser{name: "p2"})
	r.Register(DirectiveModel, &mockParser{name: "p3"})

	assert.Equal(t, 3, r.Count())
}

func TestRegistryClear(t *testing.T) {
	r := NewRegistry()
	r.Register(DirectiveRoute, &mockParser{name: "test"})
	assert.Equal(t, 1, r.Count())

	r.Clear()

	assert.Equal(t, 0, r.Count())
}

func TestRegistryParseNilComments(t *testing.T) {
	r := NewRegistry()

	err := r.Parse(DirectiveRoute, nil, nil, ContextRoute)

	assert.NoError(t, err)
}

func TestRegistryParseNoParsers(t *testing.T) {
	r := NewRegistry()
	comments := &ast.CommentGroup{
		List: []*ast.Comment{{Text: "// test"}},
	}

	err := r.Parse(DirectiveRoute, comments, nil, ContextRoute)

	assert.NoError(t, err)
}

func TestRegistryParseWithMatchingParser(t *testing.T) {
	r := NewRegistry()
	parser := &mockParser{
		name:        "test",
		matches:     true,
		parseValue:  "parsed-value",
		applyTarget: nil,
	}
	r.Register(DirectiveRoute, parser)

	comments := &ast.CommentGroup{
		List: []*ast.Comment{{Text: "// test"}},
	}
	target := make(map[string]any)

	err := r.Parse(DirectiveRoute, comments, target, ContextRoute)

	require.NoError(t, err)
	assert.True(t, parser.applyCalled)
}

func TestRegistryParseParserError(t *testing.T) {
	r := NewRegistry()
	parser := &mockParser{
		name:       "test",
		matches:    true,
		parseError: errors.New("parse error"),
	}
	r.Register(DirectiveRoute, parser)

	comments := &ast.CommentGroup{
		List: []*ast.Comment{{Text: "// test"}},
	}

	err := r.Parse(DirectiveRoute, comments, nil, ContextRoute)

	require.Error(t, err)
	var parseFailure *ErrParseFailure
	assert.ErrorAs(t, err, &parseFailure)
}

func TestRegistryParseAll(t *testing.T) {
	r := NewRegistry()
	parser := &mockParser{
		name:       "test",
		matches:    true,
		parseValue: "value",
	}
	r.Register(DirectiveRoute, parser)

	comments := &ast.CommentGroup{
		List: []*ast.Comment{{Text: "// test"}},
	}
	targets := map[Context]any{
		ContextRoute: make(map[string]any),
	}

	err := r.ParseAll(DirectiveRoute, comments, targets)

	assert.NoError(t, err)
}

// ==================== Mock Parser ====================

type mockParser struct {
	name        string
	matches     bool
	parseValue  any
	parseError  error
	applyError  error
	applyCalled bool
	applyTarget any
}

func (m *mockParser) Name() string {
	return m.name
}

func (m *mockParser) Contexts() []Context {
	return []Context{ContextRoute, ContextModel}
}

func (m *mockParser) Matches(comment string, ctx Context) bool {
	return m.matches
}

func (m *mockParser) Parse(comments *ast.CommentGroup, ctx Context) (any, error) {
	if m.parseError != nil {
		return nil, m.parseError
	}
	return m.parseValue, nil
}

func (m *mockParser) Apply(target, value any, ctx Context) error {
	m.applyCalled = true
	m.applyTarget = target
	return m.applyError
}
