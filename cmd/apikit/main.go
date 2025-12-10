package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "1.0.0"

var (
	verbose bool
	dryRun  bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "apikit",
	Short: "HTTP handler wrapper generator for Go",
	Long: `apikit is a code generation tool that creates HTTP handler wrappers
from annotated Go functions, handling request parsing and response serialization.

Features:
  • Automatic request parsing (path, query, headers, body)
  • Type-safe parameter extraction
  • Nested and embedded struct support
  • Extensible extractor system
  • Support for http.ResponseWriter and *http.Request

Example:
  Add to your handler file:
    //go:generate apikit generate

  Then run:
    go generate ./...`,
	Version:           version,
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func main() {

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be generated without writing files")
}
