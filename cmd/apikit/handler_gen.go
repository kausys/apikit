package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kausys/apikit/cmd/handler/checksum"
	"github.com/kausys/apikit/cmd/handler/codegen"
	"github.com/kausys/apikit/cmd/handler/parser"

	"github.com/spf13/cobra"
)

var (
	handlerSourceFile string
	handlerOutputFile string
	handlerForce      bool
	handlerFramework  string
	handlerCheck      bool
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

Arguments may be files, directories, or Go-style recursive patterns:
a directory processes its .go files, "dir/..." (or "./...") walks the
subtree. Expanded patterns skip *_apikit.go, *_test.go, hidden/underscore
directories, testdata and vendor; files without annotations are skipped.

Examples:
  # From go:generate (automatic)
  //go:generate apikit handler gen

  # Everything below the current directory
  apikit handler gen ./...

  # One package directory
  apikit handler gen ./auth

  # Explicit file
  apikit handler gen --file handlers.go

  # With verbose output
  apikit handler gen --verbose

  # Dry run (show output without writing)
  apikit handler gen --dry-run

  # In CI: fail if any wrapper is not what the source generates
  apikit handler gen --check ./...`,
	RunE: runHandlerGen,
}

//nolint:gochecknoinits // cobra command registration requires init
func init() {
	handlerGenCmd.Flags().StringVarP(&handlerSourceFile, "file", "f", "", "source file to process (defaults to GOFILE env var)")
	handlerGenCmd.Flags().StringVarP(&handlerOutputFile, "output", "o", "", "output file (defaults to <source>_apikit.go)")
	handlerGenCmd.Flags().BoolVar(&handlerForce, "force", false, "force regeneration even if source hasn't changed")
	handlerGenCmd.Flags().StringVar(&handlerFramework, "framework", "http", "target framework: http (default), fiber, gin, echo")
	handlerGenCmd.Flags().BoolVar(&handlerCheck, "check", false, "write nothing; fail listing every wrapper that is missing, stale or orphaned")

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

	// Expand file, directory, and dir/... arguments into concrete source files
	resolvedFiles, err := expandSourceArgs(cwd, sourceFiles)
	if err != nil {
		return err
	}

	if verbose {
		log.Printf("Processing %d file(s)...", len(resolvedFiles)) //nolint:gosec // count is safe
	}

	if handlerOutputFile != "" && len(resolvedFiles) > 1 {
		return fmt.Errorf("--output names one file, but %d sources were given: each would overwrite the last", len(resolvedFiles))
	}

	// Create a single parser instance to share cache across all files
	p := parser.New()
	var stale []string

	// Process each file
	for i, sourceFilePath := range resolvedFiles {
		if verbose {
			log.Printf("[%d/%d] Processing %s", i+1, len(resolvedFiles), sourceFilePath) //nolint:gosec // file path from user args
		}

		changed, err := handlerGenerateWithParser(p, sourceFilePath)
		if err != nil {
			return fmt.Errorf("processing %s: %w", sourceFilePath, err)
		}
		stale = append(stale, changed...)
	}

	orphans, err := orphanWrappers(cwd, sourceFiles)
	if err != nil {
		return err
	}
	for _, orphan := range orphans {
		if err := removeWrapper(orphan); err != nil {
			return err
		}
		stale = append(stale, orphan)
	}

	if handlerCheck && len(stale) > 0 {
		return fmt.Errorf("%d generated file(s) are not up to date — run apikit handler gen:\n  %s",
			len(stale), strings.Join(stale, "\n  "))
	}

	if verbose {
		log.Println("Generation completed successfully")
	}

	return nil
}

// expandSourceArgs resolves handler-gen arguments into concrete .go paths.
// Files pass through untouched; a directory contributes its own .go files;
// "dir/..." (Go tooling style) walks the subtree. Expanded matches skip
// generated wrappers (*_apikit.go), tests, hidden/underscore directories,
// testdata and vendor.
func expandSourceArgs(cwd string, args []string) ([]string, error) {
	var files []string
	for _, arg := range args {
		path := arg
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}

		if base, ok := strings.CutSuffix(path, "..."); ok {
			root := filepath.Clean(base)
			matched, err := walkSources(root)
			if err != nil {
				return nil, fmt.Errorf("walking %s: %w", arg, err)
			}
			if len(matched) == 0 {
				return nil, fmt.Errorf("no Go source files under %s", arg)
			}
			files = append(files, matched...)
			continue
		}

		path = filepath.Clean(path)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source file not found: %s", path)
		}
		if err != nil {
			return nil, fmt.Errorf("inspecting %s: %w", arg, err)
		}
		if !info.IsDir() {
			files = append(files, path)
			continue
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", arg, err)
		}
		n := 0
		for _, e := range entries {
			if !e.IsDir() && isSourceCandidate(e.Name()) {
				files = append(files, filepath.Join(path, e.Name()))
				n++
			}
		}
		if n == 0 {
			return nil, fmt.Errorf("no Go source files in %s", arg)
		}
	}
	return files, nil
}

// walkSources collects source candidates below root, skipping directories the
// Go toolchain also ignores (hidden, underscore-prefixed, testdata, vendor).
func walkSources(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
				name == "testdata" || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if isSourceCandidate(d.Name()) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// isSourceCandidate reports whether a file name is a hand-written Go source.
func isSourceCandidate(name string) bool {
	return strings.HasSuffix(name, ".go") &&
		!strings.HasSuffix(name, "_apikit.go") &&
		!strings.HasSuffix(name, "_test.go")
}

// handlerGenerateWithParser brings the wrapper of one source file up to date
// and returns the files that were not: written, removed, or under --check left
// alone and reported.
func handlerGenerateWithParser(p *parser.Parser, sourceFilePath string) ([]string, error) {
	// Determine output file name
	output := handlerOutputFile
	if output == "" {
		output = strings.TrimSuffix(sourceFilePath, ".go") + "_apikit.go"
	}

	// Parse the source file: the handlers, and every struct they read from
	// whatever package declares it.
	if verbose {
		log.Printf("Parsing %s...", sourceFilePath) //nolint:gosec // file path from user args
	}

	result, err := p.ParseFile(sourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	// Errors first: these describe a wrapper that would be generated WRONG rather
	// than not at all — one that compiles and misbehaves at runtime. Reported
	// together so one run shows every problem in the file, and always fatal.
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("%s:\n  %s", sourceFilePath, strings.Join(result.Errors, "\n  "))
	}

	// Warnings are printed unconditionally, not only under --verbose. A warning
	// nobody sees is not a warning; the conditions that used to hide here are
	// errors now, and whatever lands in this slice next deserves to be read.
	for _, warning := range result.Warnings {
		log.Printf("Warning: %s", warning) //nolint:gosec // warning from parser output
	}

	// A file without handlers has no wrapper. One left from when it had some
	// would keep registering handlers that no longer exist.
	if len(result.Handlers) == 0 {
		if verbose {
			log.Println("No handlers found with //apikit:handler comment")
		}
		if isGeneratedWrapper(output) {
			return []string{output}, removeWrapper(output)
		}
		return nil, nil
	}

	// What the wrapper is generated from: this generator, its options, and
	// the packages every struct came from. Unchanged inputs mean an unchanged
	// wrapper, and the expensive part — type-checking and formatting — is
	// skipped. Under --check nothing is skipped: the point is to compare.
	sourceChecksum, err := checksum.CalculateInputsChecksum(
		filepath.Dir(sourceFilePath),
		[]string{"apikit-wrapper/" + codegen.FormatVersion, handlerFramework},
		result.Packages,
	)
	if err != nil {
		return nil, fmt.Errorf("calculating source checksum: %w", err)
	}
	if !handlerForce && !handlerCheck {
		if stored, err := checksum.ExtractChecksum(output); err == nil && stored == sourceChecksum {
			if verbose {
				log.Printf("Source unchanged, skipping %s", sourceFilePath) //nolint:gosec // file path from user args
			}
			return nil, nil
		}
	}

	// Detect encoding.TextUnmarshaler on custom field types so extractors can
	// emit UnmarshalText instead of an invalid string cast.
	if err := p.AnnotateTextUnmarshalers(sourceFilePath, result); err != nil {
		if verbose {
			log.Printf("Warning: TextUnmarshaler annotation failed: %v", err)
		}
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
		return nil, fmt.Errorf("creating generator: %w", err)
	}

	code, err := gen.Generate(result)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	code = checksum.AddChecksumToGenerated(code, sourceChecksum)

	if dryRun {
		fmt.Printf("Would write to %s:\n", output)
		fmt.Println(string(code))
		return nil, nil
	}

	// Written only when it changed, so an up-to-date tree keeps its
	// modification times and a build cache keyed on them stays warm.
	current, err := os.ReadFile(filepath.Clean(output))
	if err == nil && bytes.Equal(current, code) {
		return nil, nil
	}
	if handlerCheck {
		return []string{output}, nil
	}

	if verbose {
		log.Printf("Writing %s...", output) //nolint:gosec // output path validated before use
	}
	// Generated source is as readable as the source it is generated from.
	if err := os.WriteFile(filepath.Clean(output), code, 0o644); err != nil { //nolint:gosec // G306: a Go source file, not a secret
		return nil, fmt.Errorf("writing output file: %w", err)
	}
	return []string{output}, nil
}

// generatedHeader is the first line of every file apikit generates.
const generatedHeader = "// Code generated by apikit. DO NOT EDIT."

// isGeneratedWrapper reports whether path is a file apikit generated, so it
// is never asked to remove one somebody wrote by hand.
func isGeneratedWrapper(path string) bool {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	line, err := bufio.NewReader(f).ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	return strings.TrimSpace(line) == generatedHeader
}

// removeWrapper removes a generated wrapper, or under --check leaves it for
// the report. Under --dry-run it only says so.
func removeWrapper(path string) error {
	if handlerCheck {
		return nil
	}
	if dryRun {
		fmt.Printf("Would remove %s\n", path)
		return nil
	}
	if verbose {
		log.Printf("Removing %s: its source declares no handler", path) //nolint:gosec // path from the scanned tree
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("removing stale wrapper: %w", err)
	}
	return nil
}

// orphanWrappers finds generated wrappers under directory and dir/...
// arguments whose source file no longer exists. They would keep compiling
// handlers that are gone.
func orphanWrappers(cwd string, args []string) ([]string, error) {
	var orphans []string
	for _, arg := range args {
		path := arg
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}
		var candidates []string
		if base, ok := strings.CutSuffix(path, "..."); ok {
			root := filepath.Clean(base)
			err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					name := d.Name()
					if p != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
						name == "testdata" || name == "vendor") {
						return filepath.SkipDir
					}
					return nil
				}
				if strings.HasSuffix(p, "_apikit.go") {
					candidates = append(candidates, p)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walking %s: %w", arg, err)
			}
		} else if info, err := os.Stat(path); err == nil && info.IsDir() {
			matches, _ := filepath.Glob(filepath.Join(path, "*_apikit.go"))
			candidates = matches
		}
		for _, wrapper := range candidates {
			source := strings.TrimSuffix(wrapper, "_apikit.go") + ".go"
			if _, err := os.Stat(source); errors.Is(err, fs.ErrNotExist) && isGeneratedWrapper(wrapper) {
				orphans = append(orphans, wrapper)
			}
		}
	}
	return orphans, nil
}
