package main

import (
	"fmt"

	"github.com/kausys/apikit/openapi/sdkgen"

	"github.com/spf13/cobra"
)

var (
	sdkgenOutputDir string
	sdkgenProvider  string
)

// sdkCmd is the parent "sdk" subcommand
var sdkCmd = &cobra.Command{
	Use:   "sdk",
	Short: "Go SDK generation from OpenAPI specifications",
	Long:  `Commands for generating Go SDK packages from OpenAPI specifications.`,
}

// sdkGenCmd is the "sdk gen" subcommand
var sdkGenCmd = &cobra.Command{
	Use:   "gen <config.sdkgen.yaml>",
	Short: "Generate Go SDK package from OpenAPI spec",
	Long: `Generates a complete Go SDK package from an OpenAPI specification
and an SDK configuration file (.sdkgen.yaml).

The generated SDK follows the standard 4-layer architecture:
  - client/   HTTP client with middleware chain
  - config/   Configuration with gookit/config
  - models/   Request/response structs and enums
  - services/ Service methods per API tag

Example:
  apikit sdk gen pokemon.sdkgen.yaml -o ./pkg/sdk/pokemon
  apikit sdk gen pokemon.sdkgen.yaml -o ./pkg/sdk/pokemon --provider myProvider`,
	Args: cobra.ExactArgs(1),
	RunE: runSDKGen,
}

//nolint:gochecknoinits // cobra command registration requires init
func init() {
	sdkGenCmd.Flags().StringVarP(&sdkgenOutputDir, "output", "o", "", "Output directory for generated SDK (required)")
	sdkGenCmd.Flags().StringVar(&sdkgenProvider, "provider", "", "Override provider name from config")
	_ = sdkGenCmd.MarkFlagRequired("output")

	sdkCmd.AddCommand(sdkGenCmd)
	rootCmd.AddCommand(sdkCmd)
}

func runSDKGen(cmd *cobra.Command, args []string) error {
	opts := []sdkgen.Option{
		sdkgen.WithConfigPath(args[0]),
		sdkgen.WithOutputDir(sdkgenOutputDir),
	}

	if sdkgenProvider != "" {
		opts = append(opts, sdkgen.WithProvider(sdkgenProvider))
	}

	gen := sdkgen.New(opts...)

	if err := gen.Generate(); err != nil {
		return fmt.Errorf("SDK generation failed: %w", err)
	}

	fmt.Println("SDK generated successfully")
	return nil
}
