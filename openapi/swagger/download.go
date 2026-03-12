package swagger

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// GitHubReleasesAPI is the GitHub API endpoint for swagger-ui releases
	GitHubReleasesAPI = "https://api.github.com/repos/swagger-api/swagger-ui/releases/latest"
	// DownloadURLTemplate is the template for downloading swagger-ui releases
	DownloadURLTemplate = "https://github.com/swagger-api/swagger-ui/archive/refs/tags/%s.zip"
)

// GitHubRelease represents the GitHub API response for a release.
type GitHubRelease struct {
	TagName string `json:"tag_name"` //nolint:tagliatelle // GitHub API uses snake_case
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"` //nolint:tagliatelle // GitHub API uses snake_case
}

// DownloadOptions configures the Swagger UI download.
type DownloadOptions struct {
	// OutputDir is the directory where swagger-ui.zip will be saved
	OutputDir string
	// CustomCSS is optional custom CSS to inject
	CustomCSS string
	// CustomInitializer is optional custom JavaScript for swagger-initializer.js
	CustomInitializer string
	// Version is the specific version to download (empty for latest)
	Version string
}

// GetLatestVersion fetches the latest Swagger UI version from GitHub.
func GetLatestVersion() (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, GitHubReleasesAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "openapi-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse release info: %w", err)
	}

	return release.TagName, nil
}

// Download downloads and prepares Swagger UI for embedding.
func Download(opts DownloadOptions) (string, error) {
	version := opts.Version
	if version == "" {
		var err error
		version, err = GetLatestVersion()
		if err != nil {
			return "", fmt.Errorf("failed to get latest version: %w", err)
		}
	}

	// Ensure version has 'v' prefix
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	downloadURL := fmt.Sprintf(DownloadURLTemplate, version)
	fmt.Printf("Downloading Swagger UI %s from %s\n", version, downloadURL)

	// Download the zip file
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read download: %w", err)
	}

	// Process the zip file
	processedZip, err := processSwaggerUI(zipData, version, opts)
	if err != nil {
		return "", fmt.Errorf("failed to process swagger-ui: %w", err)
	}

	// Ensure output directory exists
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}
	if err := os.MkdirAll(opts.OutputDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write the processed zip
	outputPath := filepath.Join(opts.OutputDir, "swagger-ui.zip")
	if err := os.WriteFile(outputPath, processedZip, 0600); err != nil {
		return "", fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Printf("Swagger UI %s saved to %s\n", version, outputPath)
	return version, nil
}

// processSwaggerUI extracts dist folder, applies customizations, and repackages.
func processSwaggerUI(zipData []byte, version string, opts DownloadOptions) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}

	// Version without 'v' prefix for folder name
	versionNoV := strings.TrimPrefix(version, "v")
	distPrefix := fmt.Sprintf("swagger-ui-%s/dist/", versionNoV)

	var outputBuf bytes.Buffer
	writer := zip.NewWriter(&outputBuf)

	sourceMappingRegex := regexp.MustCompile(`//# sourceMappingURL=.*`)

	for _, file := range reader.File {
		// Only process files from the dist folder
		if !strings.HasPrefix(file.Name, distPrefix) {
			continue
		}

		// Skip source map files
		if strings.HasSuffix(file.Name, ".map") {
			continue
		}
		// Skip ES module bundle files
		if strings.Contains(file.Name, "swagger-ui-es-") {
			continue
		}

		// Get the relative path (remove dist prefix)
		relativePath := strings.TrimPrefix(file.Name, distPrefix)
		if relativePath == "" {
			continue
		}
		// (continued in next file section due to length)
		outputFile, err := writer.Create(relativePath)
		if err != nil {
			return nil, err
		}

		rc, err := file.Open()
		if err != nil {
			return nil, err
		}

		content, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}

		// Process JavaScript and CSS files - remove source mapping comments
		if strings.HasSuffix(relativePath, ".js") || strings.HasSuffix(relativePath, ".css") {
			content = sourceMappingRegex.ReplaceAll(content, []byte{})
		}

		// Handle swagger-initializer.js customization
		if relativePath == "swagger-initializer.js" && opts.CustomInitializer != "" {
			content = []byte(opts.CustomInitializer)
		}

		// Handle index.html customization for custom CSS
		if relativePath == "index.html" && opts.CustomCSS != "" {
			htmlContent := string(content)
			cssLink := `<link rel="stylesheet" type="text/css" href="custom-styles.css">`
			htmlContent = strings.Replace(htmlContent, "</head>", cssLink+"\n</head>", 1)
			content = []byte(htmlContent)
		}

		if _, err := outputFile.Write(content); err != nil {
			return nil, err
		}
	}

	// Add custom CSS file if provided
	if opts.CustomCSS != "" {
		cssFile, err := writer.Create("custom-styles.css")
		if err != nil {
			return nil, err
		}
		if _, err := cssFile.Write([]byte(opts.CustomCSS)); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return outputBuf.Bytes(), nil
}
