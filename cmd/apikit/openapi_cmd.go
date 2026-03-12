package main

import (
	"fmt"

	"github.com/kausys/apikit/openapi/cache"
	"github.com/kausys/apikit/openapi/generator"

	"github.com/spf13/cobra"
)

var (
	openapiOutputFile   string
	openapiOutputFormat string
	openapiPattern      string
	openapiDir          string
	openapiNoCache      bool
	openapiNoDefault    bool
	openapiFlatten      bool
	openapiValidate     bool
	openapiIgnorePaths  []string
	openapiCleanUnused  bool
	openapiMultiSpec    bool
	openapiSpecName     string
	openapiEnumRefs     bool
)

// openapiCmd is the parent "openapi" subcommand
var openapiCmd = &cobra.Command{
	Use:   "openapi",
	Short: "OpenAPI 3.1 specification generation",
	Long: `Commands for generating and managing OpenAPI 3.1 specifications from Go source code.

It scans your Go code for swagger directives and generates a complete OpenAPI specification
in YAML or JSON format.

Example:
  apikit openapi gen -o openapi.yaml
  apikit openapi gen --pattern ./api/... -o api-spec.json --format json
  apikit openapi clean`,
}

// openapiGenCmd is the "openapi gen" subcommand
var openapiGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate OpenAPI specification from Go source code",
	Long: `Generate scans Go source code for swagger directives and generates
an OpenAPI 3.1 specification.

Supported directives:
  swagger:meta       - API metadata (title, version, description)
  swagger:model      - Schema definitions
  swagger:route      - Operation definitions
  swagger:parameters - Parameter definitions
  swagger:enum       - Enum definitions

Example:
  apikit openapi gen
  apikit openapi gen -o api.yaml -p ./api/...
  apikit openapi gen -o api.json -f json --no-cache`,
	RunE: runOpenapiGen,
}

// openapiCleanCmd is the "openapi clean" subcommand
var openapiCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean the cache directory",
	Long: `Clean removes the .openapi cache directory.

This forces a full rebuild on the next generate command.

Example:
  apikit openapi clean`,
	RunE: runOpenapiClean,
}

// openapiStatusCmd is the "openapi status" subcommand
var openapiStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cache status",
	Long: `Status displays information about the current cache state.

It shows the number of cached files, schemas, routes, and parameters.

Example:
  apikit openapi status`,
	RunE: runOpenapiStatus,
}

//nolint:gochecknoinits // cobra command registration requires init
func init() {
	openapiGenCmd.Flags().StringVarP(&openapiOutputFile, "output", "o", "openapi.yaml", "Output file path")
	openapiGenCmd.Flags().StringVarP(&openapiOutputFormat, "format", "f", "yaml", "Output format: yaml or json")
	openapiGenCmd.Flags().StringVarP(&openapiPattern, "pattern", "p", "./...", "Package pattern to scan")
	openapiGenCmd.Flags().StringVarP(&openapiDir, "dir", "d", ".", "Root directory to scan from")
	openapiGenCmd.Flags().BoolVar(&openapiNoCache, "no-cache", false, "Disable incremental caching")
	openapiGenCmd.Flags().BoolVar(&openapiFlatten, "flatten", false, "Inline $ref schemas instead of using references")
	openapiGenCmd.Flags().BoolVar(&openapiValidate, "validate", false, "Validate the generated spec")
	openapiGenCmd.Flags().StringSliceVar(&openapiIgnorePaths, "ignore", nil, "Path patterns to ignore")
	openapiGenCmd.Flags().BoolVar(&openapiCleanUnused, "clean-unused", false, "Remove unreferenced schemas")
	openapiGenCmd.Flags().BoolVar(&openapiMultiSpec, "multi-specs", false, "Generate multiple specs based on spec: directives")
	openapiGenCmd.Flags().StringVar(&openapiSpecName, "spec", "", "Generate only a specific spec by name")
	openapiGenCmd.Flags().BoolVar(&openapiNoDefault, "no-default", false, "Skip generating the default spec for routes without spec: directives")
	openapiGenCmd.Flags().BoolVar(&openapiEnumRefs, "enum-refs", false, "Generate enums as $ref references instead of inline")

	openapiCmd.AddCommand(openapiGenCmd)
	openapiCmd.AddCommand(openapiCleanCmd)
	openapiCmd.AddCommand(openapiStatusCmd)
	rootCmd.AddCommand(openapiCmd)
}

func runOpenapiGen(cmd *cobra.Command, args []string) error {
	// Load custom types from config file
	if err := generator.LoadConfigFile(openapiDir); err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	gen := generator.New(
		generator.WithDir(openapiDir),
		generator.WithPattern(openapiPattern),
		generator.WithOutput(openapiOutputFile, openapiOutputFormat),
		generator.WithCache(!openapiNoCache),
		generator.WithFlatten(openapiFlatten),
		generator.WithValidation(openapiValidate),
		generator.WithIgnorePaths(openapiIgnorePaths...),
		generator.WithCleanUnused(openapiCleanUnused),
		generator.WithNoDefault(openapiNoDefault),
		generator.WithEnumRefs(openapiEnumRefs),
	)

	if openapiMultiSpec {
		_, err := gen.GenerateMulti()
		if err != nil {
			return fmt.Errorf("multi-spec generation failed: %w", err)
		}
		return nil
	}

	if openapiSpecName != "" {
		_, err := gen.GenerateSpec(openapiSpecName)
		if err != nil {
			return fmt.Errorf("spec generation failed: %w", err)
		}
		return nil
	}

	_, err := gen.Generate()
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	return nil
}

func runOpenapiClean(cmd *cobra.Command, args []string) error {
	mgr := cache.NewManager(".")

	if err := mgr.Clean(); err != nil {
		return fmt.Errorf("failed to clean cache: %w", err)
	}

	fmt.Println("Cache cleaned successfully")
	return nil
}

func runOpenapiStatus(cmd *cobra.Command, args []string) error {
	mgr := cache.NewManager(".")

	if err := mgr.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	if err := mgr.Load(); err != nil {
		fmt.Println("No cache found")
		return nil //nolint:nilerr // intentionally ignoring load error when cache doesn't exist
	}

	stats := mgr.Stats()

	fmt.Println("Cache Status")
	fmt.Println("───────────────")
	fmt.Printf("   Files:      %d\n", stats.FileCount)
	fmt.Printf("   Schemas:    %d\n", stats.SchemaCount)
	fmt.Printf("   Routes:     %d\n", stats.RouteCount)
	fmt.Printf("   Parameters: %d\n", stats.ParameterCount)

	return nil
}
