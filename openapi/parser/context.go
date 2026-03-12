// Package parser provides an extensible parsing system for swagger directives.
// It allows registering custom parsers for different contexts (meta, route, model, etc.)
// and provides a thread-safe registry for managing parsers.
package parser

// Context defines the context where parsing is happening.
// Different contexts may have different parsing rules and targets.
type Context string

const (
	// ContextMeta is used when parsing swagger:meta comments.
	ContextMeta Context = "meta"

	// ContextRoute is used when parsing swagger:route comments.
	ContextRoute Context = "route"

	// ContextModel is used when parsing swagger:model comments.
	ContextModel Context = "model"

	// ContextField is used when parsing field-level comments in models.
	ContextField Context = "field"

	// ContextParameter is used when parsing swagger:parameters comments.
	ContextParameter Context = "parameter"

	// ContextEnum is used when parsing swagger:enum comments.
	ContextEnum Context = "enum"
)

// Directive represents a swagger directive type.
type Directive string

const (
	DirectiveMeta       Directive = "swagger:meta"
	DirectiveRoute      Directive = "swagger:route"
	DirectiveModel      Directive = "swagger:model"
	DirectiveParameters Directive = "swagger:parameters"
	DirectiveEnum       Directive = "swagger:enum"
	DirectiveAllOf      Directive = "swagger:allOf"
	DirectiveStrFmt     Directive = "swagger:strfmt"
)
