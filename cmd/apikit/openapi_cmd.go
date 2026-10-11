package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kausys/apikit/cmd/openapi/generator"

	"github.com/spf13/cobra"
)

var (
	openapiOutputFile   string
	openapiOutputFormat string
	openapiPattern      string
	openapiDir          string
	openapiNoCache      bool
	openapiCheck        bool
	openapiNoDefault    bool
	openapiStrict       bool
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
	Short: "Remove the .openapi directory older versions left behind",
	Long: `Clean removes the .openapi directory under --dir.

apikit no longer keeps a cache: every run scans the source, which is what
produced every spec anyway. The directory is only what older versions left.

Example:
  apikit openapi clean`,
	RunE: runOpenapiClean,
}

//nolint:gochecknoinits // cobra command registration requires init
func init() {
	openapiGenCmd.Flags().StringVarP(&openapiOutputFile, "output", "o", "openapi.yaml", "Output file path")
	openapiGenCmd.Flags().StringVarP(&openapiOutputFormat, "format", "f", "yaml", "Output format: yaml or json")
	openapiGenCmd.Flags().StringVarP(&openapiPattern, "pattern", "p", "./...", "Package pattern to scan")
	openapiGenCmd.Flags().StringVarP(&openapiDir, "dir", "d", ".", "Root directory to scan from")
	openapiGenCmd.Flags().BoolVar(&openapiNoCache, "no-cache", false, "Has no effect: there is no cache")
	_ = openapiGenCmd.Flags().MarkDeprecated("no-cache", "there is no cache; every run scans the source")
	openapiGenCmd.Flags().BoolVar(&openapiFlatten, "flatten", false, "Has no effect")
	_ = openapiGenCmd.Flags().MarkDeprecated("flatten", "it never had an effect; schemas are referenced")
	openapiGenCmd.Flags().BoolVar(&openapiValidate, "validate", false, "Fail when the generated spec is not valid OpenAPI (implied by --strict)")
	openapiGenCmd.Flags().BoolVar(&openapiCheck, "check", false, "Write nothing; fail when a spec on disk is not what the source generates")
	openapiGenCmd.Flags().StringSliceVar(&openapiIgnorePaths, "ignore", nil, "Path patterns to ignore")
	openapiGenCmd.Flags().BoolVar(&openapiCleanUnused, "clean-unused", false, "Remove unreferenced schemas")
	openapiGenCmd.Flags().BoolVar(&openapiMultiSpec, "multi-specs", false, "Generate multiple specs based on spec: directives")
	openapiGenCmd.Flags().StringVar(&openapiSpecName, "spec", "", "Generate only a specific spec by name")
	openapiGenCmd.Flags().BoolVar(&openapiNoDefault, "no-default", false, "Skip generating the default spec for routes without spec: directives")
	openapiGenCmd.Flags().BoolVar(&openapiStrict, "strict", false, "Fail instead of degrading the spec: routes dropped (no spec: under --no-default, a package that does not type-check, a malformed swagger:route), response types that resolve to nothing, and an invalid document")
	openapiGenCmd.Flags().BoolVar(&openapiEnumRefs, "enum-refs", false, "Generate enums as $ref references instead of inline")

	openapiCleanCmd.Flags().StringVarP(&openapiDir, "dir", "d", ".", "Root directory the .openapi directory is under")

	openapiCmd.AddCommand(openapiGenCmd)
	openapiCmd.AddCommand(openapiCleanCmd)
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
		generator.WithValidation(openapiValidate),
		generator.WithCheck(openapiCheck),
		generator.WithIgnorePaths(openapiIgnorePaths...),
		generator.WithCleanUnused(openapiCleanUnused),
		generator.WithNoDefault(openapiNoDefault),
		generator.WithEnumRefs(openapiEnumRefs),
		generator.WithStrict(openapiStrict),
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
	dir := filepath.Join(openapiDir, ".openapi")
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing %s: %w", dir, err)
	}
	fmt.Printf("Removed %s\n", dir)
	return nil
}
