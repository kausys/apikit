package generator

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/kausys/apikit/openapi/cache"
	"github.com/kausys/apikit/openapi/spec"
	"github.com/kausys/apikit/scanner"

	"gopkg.in/yaml.v3"
)

// Generator orchestrates the OpenAPI spec generation process.
type Generator struct {
	config  *Config
	cache   *cache.Manager
	scanner *scanner.Scanner

	// referencedSchemas tracks which schemas are actually used in the spec
	referencedSchemas map[string]bool

	// structsByNameAndSpec indexes structs by model name → spec name → *StructInfo.
	// The empty string key "" represents the general model (no spec: directive).
	structsByNameAndSpec map[string]map[string]*scanner.StructInfo
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
	}

	return &Generator{
		config:            cfg,
		cache:             cache.NewManager(cfg.Dir),
		scanner:           scanner.New(scannerOpts...),
		referencedSchemas: make(map[string]bool),
	}
}

// prepare initializes cache, scans source files, and caches scanned data.
func (g *Generator) prepare() error {
	if g.config.UseCache {
		if err := g.cache.Init(); err != nil {
			return fmt.Errorf("failed to initialize cache: %w", err)
		}
		if err := g.cache.Load(); err != nil {
			return fmt.Errorf("failed to load cache: %w", err)
		}
	}

	if err := g.scanner.Scan(); err != nil {
		return fmt.Errorf("failed to scan source files: %w", err)
	}

	// Build secondary index for multi-spec model lookups
	g.buildStructIndex()

	if g.config.UseCache {
		if err := g.cacheScannedData(); err != nil {
			return fmt.Errorf("failed to cache scanned data: %w", err)
		}
	}

	return nil
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

	// Phase 5: Write output
	if g.config.OutputFile != "" {
		if err := g.writeOutput(openAPI); err != nil {
			return nil, fmt.Errorf("failed to write output: %w", err)
		}
	}

	return openAPI, nil
}

// cacheScannedData saves source file mappings to cache.
// Schema/route conversion is deferred to assemble() to avoid duplicate work.
func (g *Generator) cacheScannedData() error {
	// Group schemas and routes by source file (using relative paths)
	fileSchemas := make(map[string][]string)
	fileRoutes := make(map[string][]string)

	// Track which file each schema came from (convert to relative path)
	for name := range g.scanner.Structs {
		if sourceFile, ok := g.scanner.StructSources[name]; ok {
			relPath := g.toRelativePath(sourceFile)
			fileSchemas[relPath] = append(fileSchemas[relPath], name)
		}
	}

	// Track which file each route came from (convert to relative path)
	for opID := range g.scanner.Routes {
		if sourceFile, ok := g.scanner.RouteSources[opID]; ok {
			relPath := g.toRelativePath(sourceFile)
			fileRoutes[relPath] = append(fileRoutes[relPath], opID)
		}
	}

	// Update cache index with file entries
	allFiles := make(map[string]bool)
	for f := range fileSchemas {
		allFiles[f] = true
	}
	for f := range fileRoutes {
		allFiles[f] = true
	}

	for filePath := range allFiles {
		schemas := fileSchemas[filePath]
		routes := fileRoutes[filePath]
		if err := g.cache.UpdateEntry(filePath, schemas, routes, nil); err != nil {
			return err
		}
	}

	// Save cache index
	return g.cache.Save()
}

// toRelativePath converts an absolute path to a path relative to the config directory.
func (g *Generator) toRelativePath(absPath string) string {
	relPath, err := filepath.Rel(g.config.Dir, absPath)
	if err != nil {
		return absPath // fallback to absolute if conversion fails
	}
	return relPath
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

	// Clean unused schemas if enabled
	if g.config.CleanUnused {
		// First, recursively mark schemas referenced by other referenced schemas
		g.markNestedReferences(openAPI.Components)
		// Then clean unused schemas
		g.cleanUnusedSchemas(openAPI.Components)
	}

	return openAPI, nil
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
	dir := filepath.Dir(g.config.OutputFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	var data []byte
	var err error

	switch g.config.OutputFormat {
	case "json":
		data, err = json.MarshalIndent(openAPI, "", "  ")
	default:
		data, err = yaml.Marshal(openAPI)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(g.config.OutputFile, data, 0600)
}
