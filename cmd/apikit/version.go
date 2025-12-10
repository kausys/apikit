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
		fmt.Printf("apikit v%s\n", version)
		fmt.Println("HTTP handler wrapper generator for Go")
		fmt.Println("https://github.com/kausys/apikit")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
