package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version number of apikit`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("apikit %s (commit: %s, date: %s)\n", version, commit, date)
		fmt.Println("API toolkit for Go")
		fmt.Println("https://github.com/kausys/apikit")
	},
}

//nolint:gochecknoinits // cobra command registration requires init
func init() {
	rootCmd.AddCommand(versionCmd)
}
