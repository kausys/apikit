package main

import (
	"fmt"

	"github.com/kausys/apikit/cmd/authz"

	"github.com/spf13/cobra"
)

var (
	authzInput   string
	authzOutput  string
	authzPackage string
	authzSchema  string
)

// authzCmd is the parent "authz" subcommand
var authzCmd = &cobra.Command{
	Use:   "authz",
	Short: "Authorization code generation",
	Long:  `Commands for generating authorization code (resources, actions, roles) from a CSV definition file.`,
}

// authzGenCmd is the "authz gen" subcommand
var authzGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate resources, actions, and roles from CSV",
	Long: `Generate Go source files for authorization resources, actions, and roles from a CSV definition file.

The CSV format is:
  # scope, resource, action (verbose), roles (pipe-separated)
  global, admins, view and list all admins, admin
  tenant, users, create new user, admin|editor

Examples:
  # Generate from default CSV
  apikit authz gen

  # Specify input and output
  apikit authz gen --input resource_actions.csv --output ./internal/authz --package authz

  # From go:generate
  //go:generate apikit authz gen --input resource_actions.csv --output ./internal/authz`,
	RunE: runAuthzGen,
}

//nolint:gochecknoinits // cobra command registration requires init
func init() {
	authzGenCmd.Flags().StringVar(&authzInput, "input", "resource_actions.csv", "path to the CSV definition file")
	authzGenCmd.Flags().StringVar(&authzOutput, "output", ".", "output directory for generated files")
	authzGenCmd.Flags().StringVar(&authzPackage, "package", "authz", "Go package name for generated code")
	authzGenCmd.Flags().StringVar(&authzSchema, "schema", "", "path to authz.yaml schema file (auto-detected next to CSV if not specified)")

	authzCmd.AddCommand(authzGenCmd)
	rootCmd.AddCommand(authzCmd)
}

func runAuthzGen(_ *cobra.Command, _ []string) error {
	if verbose {
		fmt.Printf("Generating authz code from %s into %s (package %s)\n", authzInput, authzOutput, authzPackage)
	}

	cfg := authz.Config{
		InputCSV:    authzInput,
		OutputDir:   authzOutput,
		PackageName: authzPackage,
		SchemaFile:  authzSchema,
	}

	if err := authz.Generate(cfg); err != nil {
		return fmt.Errorf("authz generation failed: %w", err)
	}

	if verbose {
		fmt.Println("Authorization code generated successfully")
	}

	return nil
}
