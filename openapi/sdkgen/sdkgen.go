// Package sdkgen generates Go SDK packages from OpenAPI specs and SDK config files.
package sdkgen

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/kausys/apikit/openapi/spec"
)

// Generator orchestrates the SDK code generation pipeline.
type Generator struct {
	configPath string
	config     *SDKGenConfig
	spec       *spec.OpenAPI

	// Overrides from CLI flags
	outputDir string
	provider  string
}

// Option configures the Generator.
type Option func(*Generator)

// WithConfigPath sets the path to the .sdkgen.yaml config file.
func WithConfigPath(path string) Option {
	return func(g *Generator) {
		g.configPath = path
	}
}

// WithOutputDir sets the output directory for generated code.
func WithOutputDir(dir string) Option {
	return func(g *Generator) {
		g.outputDir = dir
	}
}

// WithProvider overrides the provider name from the config.
func WithProvider(name string) Option {
	return func(g *Generator) {
		g.provider = name
	}
}

// New creates a new Generator with the given options.
func New(opts ...Option) *Generator {
	g := &Generator{}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Generate runs the full generation pipeline:
// Parse → Transform → Render → Format → Write
func (g *Generator) Generate() error {
	if g.outputDir == "" {
		return errors.New("output directory is required")
	}

	// 1. Load config
	cfg, err := LoadSDKGenConfig(g.configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	g.config = cfg

	// Apply CLI overrides
	if g.provider != "" {
		g.config.Provider.Name = g.provider
		g.config.Provider.DisplayName = toPascalCase(g.provider)
	}

	// Resolve spec path relative to config file directory
	specPath := g.config.Spec.Path
	if !filepath.IsAbs(specPath) {
		specPath = filepath.Join(filepath.Dir(g.configPath), specPath)
	}

	// 2. Parse OpenAPI spec
	openAPI, err := parseSpec(specPath)
	if err != nil {
		return fmt.Errorf("failed to parse spec: %w", err)
	}
	g.spec = openAPI

	// 3. Transform spec + config → SDKData
	data, err := transform(g.config, g.spec)
	if err != nil {
		return fmt.Errorf("failed to transform spec: %w", err)
	}

	// 4. Render templates → file contents
	files, err := render(data)
	if err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	// 5. Format + Write files
	if err := writeFiles(g.outputDir, files); err != nil {
		return fmt.Errorf("failed to write files: %w", err)
	}

	return nil
}
