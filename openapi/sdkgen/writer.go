package sdkgen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/tools/imports"
)

// writeFiles creates directories, formats Go files with goimports, and writes them to disk.
func writeFiles(outputDir string, files map[string][]byte) error {
	// Sort filenames for deterministic output order
	filenames := make([]string, 0, len(files))
	for name := range files {
		filenames = append(filenames, name)
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		content := files[filename]
		outPath := filepath.Join(outputDir, filename)

		// Create parent directory
		dir := filepath.Dir(outPath)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Format with goimports
		formatted, err := imports.Process(outPath, content, &imports.Options{
			Comments:   true,
			TabIndent:  true,
			TabWidth:   8,
			FormatOnly: false, // Also fix imports
		})
		if err != nil {
			// Write unformatted for debugging
			_ = os.WriteFile(outPath+".unformatted", content, 0600)
			return fmt.Errorf("failed to format %s: %w", filename, err)
		}

		if err := os.WriteFile(outPath, formatted, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", outPath, err)
		}
	}

	return nil
}
