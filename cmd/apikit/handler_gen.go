package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kausys/apikit/handler/checksum"
	"github.com/kausys/apikit/handler/codegen"
	"github.com/kausys/apikit/handler/parser"

	"github.com/spf13/cobra"
)

var (
	handlerSourceFile string
	handlerOutputFile string
	handlerForce      bool
	handlerFramework  string
)

// handlerCmd is the parent "handler" subcommand
var handlerCmd = &cobra.Command{
	Use:   "handler",
	Short: "HTTP handler code generation",
	Long:  `Commands for generating HTTP handler wrapper code from annotated Go functions.`,
}

// handlerGenCmd is the "handler gen" subcommand
var handlerGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate HTTP handler wrappers",
	Long: `Generate HTTP handler wrappers from annotated Go functions.

When called from //go:generate, it automatically detects the source file
from the GOFILE environment variable.

You can also specify a source file explicitly using the --file flag.

Examples:
  # From go:generate (automatic)
  //go:generate apikit handler gen

  # Explicit file
  apikit handler gen --file handlers.go

  # With verbose output
  apikit handler gen --verbose

  # Dry run (show output without writing)
  apikit handler gen --dry-run`,
	RunE: runHandlerGen,
}

//nolint:gochecknoinits // cobra command registration requires init
func init() {
	handlerGenCmd.Flags().StringVarP(&handlerSourceFile, "file", "f", "", "source file to process (defaults to GOFILE env var)")
	handlerGenCmd.Flags().StringVarP(&handlerOutputFile, "output", "o", "", "output file (defaults to <source>_apikit.go)")
	handlerGenCmd.Flags().BoolVar(&handlerForce, "force", false, "force regeneration even if source hasn't changed")
	handlerGenCmd.Flags().StringVar(&handlerFramework, "framework", "http", "target framework: http (default), fiber, gin, echo")

	handlerCmd.AddCommand(handlerGenCmd)
	rootCmd.AddCommand(handlerCmd)
}

func runHandlerGen(cmd *cobra.Command, args []string) error {
	// Validate framework flag
	validFrameworks := map[string]bool{"http": true, "fiber": true, "gin": true, "echo": true}
	if !validFrameworks[handlerFramework] {
		return fmt.Errorf("invalid framework %q: must be one of http, fiber, gin, echo", handlerFramework)
	}

	// Check for APIKIT_FORCE environment variable
	if !handlerForce && os.Getenv("APIKIT_FORCE") != "" {
		handlerForce = true
	}

	// Collect all source files to process
	var sourceFiles []string

	// If --file flag is provided, use it
	if handlerSourceFile != "" {
		sourceFiles = append(sourceFiles, handlerSourceFile)
	}

	// Add any positional arguments as source files
	sourceFiles = append(sourceFiles, args...)

	// If no files specified, try GOFILE env var (from go:generate)
	if len(sourceFiles) == 0 {
		goFile := os.Getenv("GOFILE")
		if goFile == "" {
			return errors.New("no source file specified\n" +
				"Use --file flag, provide files as arguments, or call from //go:generate directive:\n" +
				"  //go:generate apikit handler gen\n" +
				"  apikit handler gen file1.go file2.go\n" +
				"  apikit handler gen --file file.go")
		}
		sourceFiles = append(sourceFiles, goFile)
	}

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Resolve and validate all source files
	var resolvedFiles []string
	for _, file := range sourceFiles {
		filePath := filepath.Clean(filepath.Join(cwd, file))
		if _, err := os.Stat(filePath); os.IsNotExist(err) { //nolint:gosec // path is cleaned above
			return fmt.Errorf("source file not found: %s", filePath)
		}
		resolvedFiles = append(resolvedFiles, filePath)
	}

	if verbose {
		log.Printf("Processing %d file(s)...", len(resolvedFiles)) //nolint:gosec // count is safe
	}

	// Create a single parser instance to share cache across all files
	p := parser.New()

	// Process each file
	for i, sourceFilePath := range resolvedFiles {
		if verbose {
			log.Printf("[%d/%d] Processing %s", i+1, len(resolvedFiles), sourceFilePath) //nolint:gosec // file path from user args
		}

		if err := handlerGenerateWithParser(p, sourceFilePath); err != nil {
			return fmt.Errorf("processing %s: %w", sourceFilePath, err)
		}
	}

	if verbose {
		log.Println("Generation completed successfully")
	}

	return nil
}

func handlerGenerateWithParser(p *parser.Parser, sourceFilePath string) error {
	// Determine output file name
	output := handlerOutputFile
	if output == "" {
		output = strings.TrimSuffix(sourceFilePath, ".go") + "_apikit.go"
	}

	// Check if source has changed (unless --force is used)
	if !handlerForce {
		changed, err := checksum.HasSourceChanged(sourceFilePath, output)
		if err != nil {
			if verbose {
				log.Printf("Warning: could not check if source changed: %v", err)
			}
		} else if !changed {
			if verbose {
				log.Printf("Source unchanged, skipping %s", sourceFilePath) //nolint:gosec // file path from user args
			}
			return nil
		}
	}

	// Parse the source file
	if verbose {
		log.Printf("Parsing %s...", sourceFilePath) //nolint:gosec // file path from user args
	}

	result, err := p.ParseFile(sourceFilePath)
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	// Print warnings if any
	if len(result.Warnings) > 0 && verbose {
		for _, warning := range result.Warnings {
			log.Printf("Warning: %s", warning) //nolint:gosec // warning from parser output
		}
	}

	// Check if any handlers were found
	if len(result.Handlers) == 0 {
		if verbose {
			log.Println("No handlers found with //apikit:handler comment")
		}
		return nil
	}

	if verbose {
		log.Printf("Found %d handler(s):", len(result.Handlers)) //nolint:gosec // count is safe
		for _, h := range result.Handlers {
			log.Printf("  - %s", h.Name)
			if h.HasResponseWriter {
				log.Printf("    → with http.ResponseWriter")
			}
			if h.HasRequest {
				log.Printf("    → with *http.Request")
			}
		}
	}

	// Create generator with specified framework
	gen, err := codegen.NewWithFramework(handlerFramework)
	if err != nil {
		return fmt.Errorf("creating generator: %w", err)
	}

	// Generate code
	if verbose {
		log.Println("Generating wrapper code...")
	}

	code, err := gen.Generate(result)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	// Calculate source checksum and add to generated code
	sourceChecksum, err := checksum.CalculateFileChecksum(sourceFilePath)
	if err != nil {
		return fmt.Errorf("calculating source checksum: %w", err)
	}
	code = checksum.AddChecksumToGenerated(code, sourceChecksum)

	if dryRun {
		fmt.Printf("Would write to %s:\n", output)
		fmt.Println(string(code))
		return nil
	}

	// Write output file
	if verbose {
		log.Printf("Writing %s...", output) //nolint:gosec // output path validated before use
	}

	if err := os.WriteFile(filepath.Clean(output), code, 0600); err != nil { //nolint:gosec // path validated before use
		return fmt.Errorf("writing output file: %w", err)
	}

	if verbose {
		log.Printf("Successfully generated %s", output) //nolint:gosec // output path validated before use
	}

	return nil
}
