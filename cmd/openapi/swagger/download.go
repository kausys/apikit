package swagger

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// GitHubReleasesAPI is the GitHub API endpoint for swagger-ui releases
	GitHubReleasesAPI = "https://api.github.com/repos/swagger-api/swagger-ui/releases/latest"
	// DownloadURLTemplate is the template for downloading swagger-ui releases
	DownloadURLTemplate = "https://github.com/swagger-api/swagger-ui/archive/refs/tags/%s.zip"

	// DefaultVersion is the Swagger UI release downloaded when no version is
	// asked for. A pinned default makes `go generate` reproduce the same zip
	// on every machine and every day; "latest" made it whatever was published
	// that morning. Raise it in a change of its own, with its checksum.
	DefaultVersion = "v5.33.0"

	// LatestVersion asks for the newest release, unpinned.
	LatestVersion = "latest"

	// maxArchiveBytes bounds a download: a release archive is a few
	// megabytes, and a response that keeps going is not one.
	maxArchiveBytes = 64 << 20
)

// knownChecksums are the SHA-256 sums of release archives this version of
// apikit has verified. A download of one of them that hashes to anything else
// is refused.
var knownChecksums = map[string]string{
	"v5.33.0": "c7817b8c74f8a63a7fac9ca9e86d5563096bc7130a078bc27328d88bb4844e6c",
}

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
	// Version is the release to download: DefaultVersion when empty,
	// LatestVersion for the newest one
	Version string
	// SHA256 is the checksum the release archive must have. Empty uses the
	// one apikit knows for the version, if any.
	SHA256 string
	// Client makes the requests; nil uses one with a timeout
	Client *http.Client
	// URLTemplate overrides DownloadURLTemplate (tests)
	URLTemplate string
}

func (o DownloadOptions) client() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return &http.Client{Timeout: 2 * time.Minute}
}

// GetLatestVersion fetches the latest Swagger UI version from GitHub.
func GetLatestVersion() (string, error) {
	return latestVersion(&http.Client{Timeout: 30 * time.Second})
}

func latestVersion(client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, GitHubReleasesAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "apikit")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.UnmarshalRead(io.LimitReader(resp.Body, 1<<20), &release); err != nil {
		return "", fmt.Errorf("failed to parse release info: %w", err)
	}

	return release.TagName, nil
}

// Download downloads and prepares Swagger UI for embedding.
func Download(opts DownloadOptions) (string, error) {
	version := opts.Version
	switch version {
	case "":
		version = DefaultVersion
	case LatestVersion:
		latest, err := latestVersion(opts.client())
		if err != nil {
			return "", fmt.Errorf("failed to get latest version: %w", err)
		}
		version = latest
	}

	// Ensure version has 'v' prefix
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	template := opts.URLTemplate
	if template == "" {
		template = DownloadURLTemplate
	}
	downloadURL := fmt.Sprintf(template, version)
	fmt.Printf("Downloading Swagger UI %s from %s\n", version, downloadURL)

	zipData, err := fetch(opts.client(), downloadURL)
	if err != nil {
		return "", err
	}

	if err := verifyChecksum(zipData, version, opts.SHA256); err != nil {
		return "", err
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

	// Write the processed zip, only when it changed: the same release
	// processes to the same bytes, and an unchanged file keeps its time.
	outputPath := filepath.Join(opts.OutputDir, "swagger-ui.zip")
	if current, err := os.ReadFile(filepath.Clean(outputPath)); err == nil && bytes.Equal(current, processedZip) {
		fmt.Printf("Swagger UI %s already in %s\n", version, outputPath)
		return version, nil
	}
	// Embedded into a binary and served to browsers: not a secret.
	if err := os.WriteFile(outputPath, processedZip, 0o644); err != nil { //nolint:gosec // G306: a public asset
		return "", fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Printf("Swagger UI %s saved to %s\n", version, outputPath)
	return version, nil
}

// fetch downloads a release archive, refusing one larger than any release.
func fetch(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxArchiveBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read download: %w", err)
	}
	if len(data) > maxArchiveBytes {
		return nil, fmt.Errorf("download exceeds %d bytes; not a Swagger UI release", maxArchiveBytes)
	}
	return data, nil
}

// verifyChecksum checks an archive against the checksum it was asked to
// have, or the one apikit knows for its version. A version neither names is
// accepted with its checksum printed, so it can be pinned with --sha256.
func verifyChecksum(data []byte, version, want string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if want == "" {
		want = knownChecksums[version]
	}
	if want == "" {
		fmt.Printf("Swagger UI %s has SHA-256 %s; pass --sha256 %s to pin it\n", version, got, got)
		return nil
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("Swagger UI %s archive has SHA-256 %s, want %s: refusing it", version, got, want) //nolint:staticcheck // ST1005: a proper noun
	}
	return nil
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
