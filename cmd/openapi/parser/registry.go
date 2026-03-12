package parser

import (
	"errors"
	"go/ast"
	"sync"
)

// Registry maintains a thread-safe registry of all available parsers.
// Parsers can be registered for specific directives and are executed
// in the order they were registered.
type Registry struct {
	mu      sync.RWMutex
	parsers map[Directive][]TagParser
}

// globalRegistry is the default registry used by the package-level functions.
var globalRegistry = NewRegistry()

// NewRegistry creates a new empty parser registry.
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[Directive][]TagParser),
	}
}

// Global returns the global parser registry.
func Global() *Registry {
	return globalRegistry
}

// Register adds a parser for a specific directive to the global registry.
func Register(directive Directive, parser TagParser) {
	globalRegistry.Register(directive, parser)
}

// Register adds a parser for a specific directive.
// Multiple parsers can be registered for the same directive.
func (r *Registry) Register(directive Directive, parser TagParser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parsers[directive] = append(r.parsers[directive], parser)
}

// Unregister removes a parser by name from a directive.
func (r *Registry) Unregister(directive Directive, parserName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	parsers := r.parsers[directive]
	for i, p := range parsers {
		if p.Name() == parserName {
			r.parsers[directive] = append(parsers[:i], parsers[i+1:]...)
			return
		}
	}
}

// GetParsers returns all parsers registered for a directive.
func (r *Registry) GetParsers(directive Directive) []TagParser {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.parsers[directive]
}

// Parse executes all matching parsers for a directive with a specific context.
// It iterates through all registered parsers for the directive and:
// 1. Checks if the parser matches the comment
// 2. Parses the value
// 3. Applies the value to the target
func (r *Registry) Parse(
	directive Directive,
	comments *ast.CommentGroup,
	target any,
	ctx Context,
) error {
	if comments == nil {
		return nil
	}

	parsers := r.GetParsers(directive)
	if len(parsers) == 0 {
		return nil // No parsers registered, not an error
	}

	commentText := comments.Text()

	for _, parser := range parsers {
		// Check if the parser supports this context and matches the comment
		if !parser.Matches(commentText, ctx) {
			continue
		}

		// Parse the value from comments
		value, err := parser.Parse(comments, ctx)
		if err != nil {
			return &ErrParseFailure{
				ParserName: parser.Name(),
				Context:    ctx,
				Cause:      err,
			}
		}

		// Apply the value to the target
		if err := parser.Apply(target, value, ctx); err != nil {
			// Ignore invalid target errors - allows calling Parse with different targets
			errInvalidTarget := &ErrInvalidTarget{}
			if errors.As(err, &errInvalidTarget) {
				continue
			}
			return err
		}
	}

	return nil
}

// ParseAll executes parsers for multiple contexts with their respective targets.
func (r *Registry) ParseAll(
	directive Directive,
	comments *ast.CommentGroup,
	targets map[Context]any,
) error {
	for ctx, target := range targets {
		if err := r.Parse(directive, comments, target, ctx); err != nil {
			return err
		}
	}
	return nil
}

// List returns a map of all registered directives and their parser names.
func (r *Registry) List() map[Directive][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[Directive][]string)
	for directive, parsers := range r.parsers {
		names := make([]string, len(parsers))
		for i, p := range parsers {
			names[i] = p.Name()
		}
		result[directive] = names
	}
	return result
}

// Clear removes all registered parsers. Useful for testing.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parsers = make(map[Directive][]TagParser)
}

// Count returns the total number of registered parsers.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, parsers := range r.parsers {
		count += len(parsers)
	}
	return count
}
