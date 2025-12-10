package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kausys/apikit/checksum"
	"github.com/kausys/apikit/codegen"
	"github.com/kausys/apikit/parser"
	"github.com/spf13/cobra"
)

var (
	sourceFile string
	outputFile string
	force      bool
	framework  string
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate HTTP handler wrappers",
	Long: `Generate HTTP handler wrappers from annotated Go functions.

When called from //go:generate, it automatically detects the source file
from the GOFILE environment variable.

You can also specify a source file explicitly using the --file flag.

Examples:
  # From go:generate (automatic)
  //go:generate apikit generate

  # Explicit file
  apikit generate --file handlers.go

  # With verbose output
  apikit generate --verbose

  # Dry run (show output without writing)
  apikit generate --dry-run`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&sourceFile, "file", "f", "", "source file to process (defaults to GOFILE env var)")
	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (defaults to <source>_apikit.go)")
	generateCmd.Flags().BoolVar(&force, "force", false, "force regeneration even if source hasn't changed")
	generateCmd.Flags().StringVar(&framework, "framework", "http", "target framework: http (default), fiber, gin, echo")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Check for APIKIT_FORCE environment variable
	if !force && os.Getenv("APIKIT_FORCE") != "" {
		force = true
	}

	// Collect all source files to process
	var sourceFiles []string

	// If --file flag is provided, use it
	if sourceFile != "" {
		sourceFiles = append(sourceFiles, sourceFile)
	}

	// Add any positional arguments as source files
	sourceFiles = append(sourceFiles, args...)

	// If no files specified, try GOFILE env var (from go:generate)
	if len(sourceFiles) == 0 {
		goFile := os.Getenv("GOFILE")
		if goFile == "" {
			return fmt.Errorf("no source file specified\n" +
				"Use --file flag, provide files as arguments, or call from //go:generate directive:\n" +
				"  //go:generate apikit generate\n" +
				"  apikit generate file1.go file2.go\n" +
				"  apikit generate --file file.go")
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
		filePath := filepath.Join(cwd, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("source file not found: %s", filePath)
		}
		resolvedFiles = append(resolvedFiles, filePath)
	}

	if verbose {
		log.Printf("Processing %d file(s)...", len(resolvedFiles))
	}

	// Create a single parser instance to share cache across all files
	p := parser.New()

	// Process each file
	for i, sourceFilePath := range resolvedFiles {
		if verbose {
			log.Printf("[%d/%d] Processing %s", i+1, len(resolvedFiles), sourceFilePath)
		}

		if err := generateWithParser(p, sourceFilePath); err != nil {
			return fmt.Errorf("processing %s: %w", sourceFilePath, err)
		}
	}

	if verbose {
		log.Println("Generation completed successfully")
	}

	return nil
}

func generateWithParser(p *parser.Parser, sourceFilePath string) error {
	// Determine output file name
	output := outputFile
	if output == "" {
		output = strings.TrimSuffix(sourceFilePath, ".go") + "_apikit.go"
	}

	// Check if source has changed (unless --force is used)
	if !force {
		changed, err := checksum.HasSourceChanged(sourceFilePath, output)
		if err != nil {
			if verbose {
				log.Printf("Warning: could not check if source changed: %v", err)
			}
		} else if !changed {
			if verbose {
				log.Printf("Source unchanged, skipping %s", sourceFilePath)
			}
			return nil
		}
	}

	// Parse the source file
	if verbose {
		log.Printf("Parsing %s...", sourceFilePath)
	}

	result, err := p.ParseFile(sourceFilePath)
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	// Print warnings if any
	if len(result.Warnings) > 0 && verbose {
		for _, warning := range result.Warnings {
			log.Printf("Warning: %s", warning)
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
		log.Printf("Found %d handler(s):", len(result.Handlers))
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
	gen, err := codegen.NewWithFramework(framework)
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
		log.Printf("Writing %s...", output)
	}

	if err := os.WriteFile(output, code, 0644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	if verbose {
		log.Printf("Successfully generated %s", output)
	}

	return nil
}
