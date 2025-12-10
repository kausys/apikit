package checksum

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// checksumPattern matches the checksum comment in generated files
var checksumPattern = regexp.MustCompile(`// apikit:checksum:([a-f0-9]{64})`)

// CalculateFileChecksum calculates SHA256 checksum of a file
func CalculateFileChecksum(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// ExtractChecksum extracts the checksum from a generated file
func ExtractChecksum(filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
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
