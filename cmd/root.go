package main

import (
	"github.com/spf13/cobra"
)

var (
	verbose bool
	dryRun  bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "apikit",
	Short: "API toolkit for Go — handler codegen, OpenAPI spec generation, and more",
	Long: `apikit is a unified CLI for the kausys API toolkit.

Commands:
  handler gen   — Generate HTTP handler wrappers from annotated Go functions
  openapi gen   — Generate OpenAPI 3.1 specifications from Go source code
  openapi clean — Clean the OpenAPI cache
  openapi status — Show OpenAPI cache status
  swagger       — Manage Swagger UI assets
  sdk gen       — Generate Go SDK from an OpenAPI specification

Example:
  # Generate handler wrappers (from go:generate)
  //go:generate apikit handler gen

  # Generate OpenAPI specification
  apikit openapi gen -o openapi.yaml`,
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

//nolint:gochecknoinits // cobra flag registration requires init
func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be generated without writing files")
}
