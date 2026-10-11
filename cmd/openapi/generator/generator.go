package generator

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kausys/apikit/cmd/openapi/spec"
	"github.com/kausys/apikit/cmd/scanner"

	"go.yaml.in/yaml/v3"
)

// Generator orchestrates the OpenAPI spec generation process.
type Generator struct {
	config  *Config
	scanner *scanner.Scanner

	// referencedSchemas tracks which schemas are actually used in the spec
	referencedSchemas map[string]bool

	// structsByNameAndSpec indexes structs by model name → spec name → *StructInfo.
	// The empty string key "" represents the general model (no spec: directive).
	structsByNameAndSpec map[string]map[string]*scanner.StructInfo

	// problems found while assembling, reported with the validation of the
	// documents.
	problems []string

	// warnedTypes are the unknown types already warned about: one line per
	// type, not one per field.
	warnedTypes map[string]bool

	// stale are the files Check found out of date.
	stale []string
}

// New creates a new Generator with the given options.
func New(opts ...Option) *Generator {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	scannerOpts := []scanner.Option{
		scanner.WithDir(cfg.Dir),
		scanner.WithPattern(cfg.Pattern),
		scanner.WithIgnorePaths(cfg.IgnorePaths...),
		scanner.WithStrict(cfg.Strict),
	}

	return &Generator{
		config:            cfg,
		scanner:           scanner.New(scannerOpts...),
		referencedSchemas: make(map[string]bool),
		warnedTypes:       make(map[string]bool),
	}
}

// prepare scans the source files and checks what they declare.
func (g *Generator) prepare() error {
	if err := g.scanner.Scan(); err != nil {
		return fmt.Errorf("failed to scan source files: %w", err)
	}

	// Build secondary index for multi-spec model lookups
	g.buildStructIndex()

	// After the index: that is what a declared schema name resolves against.
	return g.checkDeclaredSchemasResolve()
}

// Generate runs the full generation pipeline.
func (g *Generator) Generate() (*spec.OpenAPI, error) {
	if err := g.prepare(); err != nil {
		return nil, err
	}

	// Phase 4: Assemble OpenAPI spec
	openAPI, err := g.assemble()
	if err != nil {
		return nil, fmt.Errorf("failed to assemble spec: %w", err)
	}
	if err := g.checkDocuments(map[string]*spec.OpenAPI{"": openAPI}); err != nil {
		return nil, err
	}

	// Phase 5: Write output
	if g.config.OutputFile != "" {
		if err := g.writeOutput(openAPI); err != nil {
			return nil, fmt.Errorf("failed to write output: %w", err)
		}
	}

	return openAPI, nil
}

// assemble creates the OpenAPI spec from scanned data.
func (g *Generator) assemble() (*spec.OpenAPI, error) {
	// Reset referenced schemas for each generation
	g.referencedSchemas = make(map[string]bool)

	openAPI := &spec.OpenAPI{
		OpenAPI: "3.1.2",
		Paths: &spec.Paths{
			PathItems: make(map[string]*spec.PathItem),
		},
		Components: &spec.Components{
			Schemas:         make(map[string]*spec.Schema),
			SecuritySchemes: make(map[string]*spec.SecurityScheme),
		},
	}

	// Set info, security schemes, and tags from meta
	g.applyMeta(openAPI, g.scanner.Meta, nil)

	// Add schemas (all models first)
	for name, structInfo := range g.scanner.Structs {
		if structInfo.IsModel {
			openAPI.Components.Schemas[name] = g.structToSchema(structInfo)
		}
	}

	// Add enums as schemas
	for name, enumInfo := range g.scanner.Enums {
		openAPI.Components.Schemas[name] = g.enumToSchema(enumInfo)
	}

	// Add external schemas from config file (highest priority — explicit override)
	maps.Copy(openAPI.Components.Schemas, GetAllExternalSchemas())

	// Add paths (this will mark schemas as referenced)
	for _, routeInfo := range g.scanner.Routes {
		g.addRoute(openAPI, routeInfo)
	}

	// A route can make a model of a type no directive named — its body's
	// struct — so the models are collected again after the routes.
	for name, structInfo := range g.scanner.Structs {
		if _, done := openAPI.Components.Schemas[name]; !done && structInfo.IsModel {
			openAPI.Components.Schemas[name] = g.structToSchema(structInfo)
		}
	}

	// Clean unused schemas if enabled
	if g.config.CleanUnused {
		// First, recursively mark schemas referenced by other referenced schemas
		g.markNestedReferences(openAPI.Components)
		// Then clean unused schemas
		g.cleanUnusedSchemas(openAPI.Components)
	}

	return openAPI, nil
}

// checkDocuments validates the documents about to be written. A problem fails
// the run under --validate or --strict, and is a warning otherwise.
func (g *Generator) checkDocuments(docs map[string]*spec.OpenAPI) error {
	problems := slices.Clone(g.problems)
	for _, name := range slices.Sorted(maps.Keys(docs)) {
		for _, problem := range ValidateDocument(docs[name]) {
			if name != "" {
				problem = name + ": " + problem
			}
			problems = append(problems, problem)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	if g.config.Validate || g.config.Strict {
		return fmt.Errorf("the generated spec is not valid:\n  %s", strings.Join(problems, "\n  "))
	}
	for _, problem := range problems {
		warn("%s", problem)
	}
	return nil
}

// warn prints a warning on standard error, where it does not mix with a spec
// written to standard output.
func warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

// markSchemaAsReferenced marks a schema as being used.
func (g *Generator) markSchemaAsReferenced(schemaName string) {
	if schemaName != "" {
		g.referencedSchemas[schemaName] = true
	}
}

// markNestedReferences uses BFS to mark schemas referenced by already-referenced schemas.
// Each schema is processed exactly once.
func (g *Generator) markNestedReferences(components *spec.Components) {
	if components == nil || components.Schemas == nil {
		return
	}

	// Seed queue with currently-referenced schemas
	queue := make([]string, 0, len(g.referencedSchemas))
	for name := range g.referencedSchemas {
		queue = append(queue, name)
	}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		schema, exists := components.Schemas[name]
		if !exists || schema == nil {
			continue
		}

		// Collect all refs from this schema and enqueue new ones
		for _, ref := range g.collectSchemaRefs(schema) {
			if !g.referencedSchemas[ref] {
				g.referencedSchemas[ref] = true
				queue = append(queue, ref)
			}
		}
	}
}

// collectSchemaRefs returns all schema names referenced by a schema.
func (g *Generator) collectSchemaRefs(schema *spec.Schema) []string {
	var refs []string
	addRef := func(s *spec.Schema) {
		if ref := g.extractSchemaRef(s); ref != "" {
			refs = append(refs, ref)
		}
	}

	for _, propSchema := range schema.Properties {
		addRef(propSchema)
	}
	if schema.Items != nil {
		addRef(schema.Items)
	}
	if schema.AdditionalProperties != nil {
		addRef(schema.AdditionalProperties)
	}
	for _, s := range schema.AllOf {
		addRef(s)
	}
	for _, s := range schema.OneOf {
		addRef(s)
	}
	for _, s := range schema.AnyOf {
		addRef(s)
	}
	return refs
}

// extractSchemaRef extracts the schema name from a $ref if present.
func (g *Generator) extractSchemaRef(schema *spec.Schema) string {
	if schema == nil || schema.Ref == "" {
		return ""
	}
	// Extract schema name from "#/components/schemas/SchemaName"
	const prefix = "#/components/schemas/"
	if len(schema.Ref) > len(prefix) {
		return schema.Ref[len(prefix):]
	}
	return ""
}

// cleanUnusedSchemas removes schemas that are declared but not referenced.
func (g *Generator) cleanUnusedSchemas(components *spec.Components) {
	if components == nil || components.Schemas == nil {
		return
	}

	// Remove unreferenced schemas
	for schemaName := range components.Schemas {
		if !g.referencedSchemas[schemaName] {
			delete(components.Schemas, schemaName)
		}
	}
}

// writeOutput writes the spec to the output file.
func (g *Generator) writeOutput(openAPI *spec.OpenAPI) error {
	data, err := g.marshal(openAPI)
	if err != nil {
		return err
	}
	if err := g.emit(g.config.OutputFile, data); err != nil {
		return err
	}
	return g.staleError()
}

// marshal encodes a document in the configured format.
func (g *Generator) marshal(openAPI *spec.OpenAPI) ([]byte, error) {
	if g.config.OutputFormat == "json" {
		return json.Marshal(openAPI, json.Deterministic(true), jsontext.WithIndent("  "))
	}
	return yaml.Marshal(openAPI)
}

// emit writes a generated file when its content changed, so an unchanged
// spec keeps its modification time. Under Check it writes nothing and records
// the file as stale instead.
func (g *Generator) emit(path string, data []byte) error {
	current, err := os.ReadFile(filepath.Clean(path))
	if err == nil && bytes.Equal(current, data) {
		return nil
	}
	if g.config.Check {
		g.stale = append(g.stale, path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	// A spec is a document to read and to embed, not a secret.
	return os.WriteFile(path, data, 0o644) //nolint:gosec // G306: a generated document
}

// staleError reports, under Check, the files that are not what the source
// generates.
func (g *Generator) staleError() error {
	if len(g.stale) == 0 {
		return nil
	}
	return fmt.Errorf("%d spec file(s) are not up to date — run apikit openapi gen:\n  %s",
		len(g.stale), strings.Join(g.stale, "\n  "))
}
