// Package openapi provides a Go library for generating OpenAPI 3.1 specifications
// from Go source code using swagger directives.
//
// # Quick Start
//
// Generate an OpenAPI spec from your Go code:
//
//	spec, err := openapi.Generate(
//		openapi.WithDir("."),
//		openapi.WithPattern("./..."),
//		openapi.WithOutput("openapi.yaml", "yaml"),
//	)
//
// # Directives
//
// The library scans Go source code for the following directives:
//
//   - swagger:meta - API metadata (title, version, description)
//   - swagger:model - Schema definitions
//   - swagger:route - Operation definitions
//   - swagger:parameters - Parameter definitions
//   - swagger:enum - Enum definitions
//
// # Parameters from handlers
//
// A swagger:route declared on a handler — func(context.Context, Request, ...) —
// needs no swagger:parameters struct: its parameters and its body are read from
// the request type, with the binding tags the generated wrapper reads (path,
// query, header, cookie, form, and the field tagged json:"body").
package openapi

import (
	"github.com/kausys/apikit/cmd/openapi/generator"
	"github.com/kausys/apikit/cmd/openapi/spec"
)

// Option is a function type for configuring the generator.
type Option = generator.Option

// Generate creates an OpenAPI specification from Go source code.
// It scans the specified packages for swagger directives and generates
// a complete OpenAPI 3.1 specification.
func Generate(opts ...Option) (*spec.OpenAPI, error) {
	gen := generator.New(opts...)
	return gen.Generate()
}

// WithDir sets the root directory to scan from.
var WithDir = generator.WithDir

// WithPattern sets the package pattern to scan (e.g., "./...", "./api/...").
var WithPattern = generator.WithPattern

// WithIgnorePaths sets path patterns to exclude during scanning.
var WithIgnorePaths = generator.WithIgnorePaths

// WithOutput sets the output file path and format ("yaml" or "json").
var WithOutput = generator.WithOutput

// WithCache does nothing: there is no cache.
//
// Deprecated: every run scans the source.
var WithCache = generator.WithCache //nolint:staticcheck // SA1019: kept so existing callers compile

// WithFlatten does nothing; it never did.
//
// Deprecated: schemas are always referenced.
var WithFlatten = generator.WithFlatten //nolint:staticcheck // SA1019: kept so existing callers compile

// WithValidation makes a problem in the generated document fatal.
var WithValidation = generator.WithValidation

// WithStrict makes every silent loss fatal: dropped routes, unresolved
// response types, an invalid document.
var WithStrict = generator.WithStrict

// WithCheck writes nothing and fails when a file on disk is not what the
// source generates.
var WithCheck = generator.WithCheck

// WithCleanUnused enables or disables removal of unreferenced schemas.
var WithCleanUnused = generator.WithCleanUnused

// WithEnumRefs enables generating enums as $ref references instead of inline.
var WithEnumRefs = generator.WithEnumRefs
