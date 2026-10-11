package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// checksumPattern matches the checksum comment in generated files
var checksumPattern = regexp.MustCompile(`// apikit:checksum:([a-f0-9]{64})`)

// CalculateFileChecksum calculates SHA256 checksum of a file
func CalculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// CalculateInputsChecksum calculates the SHA256 checksum of everything a
// wrapper is generated from: the generator itself (its version and options,
// given as parts) and the Go source of the packages its types were read from.
// Tests and generated wrappers are left out of a package; neither changes a
// wrapper.
//
// A wrapper depends on more than its handler's file. Its payload may be
// declared in another file, or another package, and a change there must
// regenerate it; so must a new apikit, whose output may differ for the same
// source.
//
// Paths are hashed relative to base, the handler's directory, so the same tree
// hashes the same on every machine: the checksum is committed with the wrapper.
func CalculateInputsChecksum(base string, parts, packageDirs []string) (string, error) {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = fmt.Fprintf(hash, "%d:%s\n", len(part), part)
	}
	for _, dir := range packageDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			return "", err
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") || strings.HasSuffix(file, "_apikit.go") {
				continue
			}
			content, err := os.ReadFile(filepath.Clean(file))
			if err != nil {
				return "", err
			}
			name, err := filepath.Rel(base, file)
			if err != nil {
				name = filepath.Base(file)
			}
			_, _ = fmt.Fprintf(hash, "%s:%d\n", filepath.ToSlash(name), len(content))
			_, _ = hash.Write(content)
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ExtractChecksum extracts the checksum from a generated file
func ExtractChecksum(filePath string) (string, error) {
	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // File doesn't exist, no checksum
		}
		return "", err
	}

	// Look for checksum in first few lines
	lines := strings.Split(string(content), "\n")
	for i := 0; i < len(lines) && i < 10; i++ {
		if matches := checksumPattern.FindStringSubmatch(lines[i]); len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", nil
}

// HasSourceChanged checks if the source file has changed since generation
func HasSourceChanged(sourceFile, generatedFile string) (bool, error) {
	// Get current source checksum
	currentChecksum, err := CalculateFileChecksum(sourceFile)
	if err != nil {
		return false, fmt.Errorf("calculating source checksum: %w", err)
	}

	// Get stored checksum from generated file
	storedChecksum, err := ExtractChecksum(generatedFile)
	if err != nil {
		return false, fmt.Errorf("extracting stored checksum: %w", err)
	}

	// If no stored checksum, consider it changed
	if storedChecksum == "" {
		return true, nil
	}

	return currentChecksum != storedChecksum, nil
}

// AddChecksumToGenerated adds checksum comment to generated content
func AddChecksumToGenerated(content []byte, sourceChecksum string) []byte {
	checksumComment := fmt.Sprintf("// apikit:checksum:%s", sourceChecksum)

	// Find where to insert (after "DO NOT EDIT" comment)
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.Contains(line, "DO NOT EDIT") {
			// Insert checksum on next line
			newLines := make([]string, 0, len(lines)+1)
			newLines = append(newLines, lines[:i+1]...)
			newLines = append(newLines, checksumComment)
			newLines = append(newLines, lines[i+1:]...)
			return []byte(strings.Join(newLines, "\n"))
		}
	}

	// If not found, insert at the beginning
	return []byte(checksumComment + "\n" + string(content))
}
