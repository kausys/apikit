// Package cache provides incremental build caching for OpenAPI spec generation.
// It stores parsed results in .openapi/ directory to speed up subsequent builds.
package cache

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CalculateChecksum calculates SHA256 checksum of a file.
func CalculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

// CalculateContentChecksum calculates SHA256 checksum of content bytes.
func CalculateContentChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("sha256:%x", hash)
}
