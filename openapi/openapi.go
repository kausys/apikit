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
// # Caching
//
// The library supports incremental builds through caching. By default, parsed
// schemas and routes are cached in a .openapi directory. Only files that have
// changed since the last build are re-parsed.
//
// To disable caching:
//
//	spec, err := openapi.Generate(
//		openapi.WithCache(false),
//	)
package openapi

import (
	"github.com/kausys/apikit/openapi/generator"
	"github.com/kausys/apikit/openapi/spec"
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

// WithCache enables or disables incremental caching.
var WithCache = generator.WithCache

// WithFlatten enables or disables schema flattening (inlining $refs).
var WithFlatten = generator.WithFlatten

// WithValidation enables or disables spec validation after generation.
var WithValidation = generator.WithValidation

// WithCleanUnused enables or disables removal of unreferenced schemas.
var WithCleanUnused = generator.WithCleanUnused

// WithEnumRefs enables generating enums as $ref references instead of inline.
var WithEnumRefs = generator.WithEnumRefs
