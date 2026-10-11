package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "apikit:", err)
		os.Exit(1)
	}
}

//nolint:gochecknoinits // the version must be known before any command runs
func init() {
	// A release build gets its version from the linker. One built by go
	// install or go tool does not, but the module version is in its build
	// info.
	if version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
}
