package swagger

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestVersion(t *testing.T) {
	// Note: This test makes a real network call
	// In a CI environment, you might want to skip this or mock it
	if os.Getenv("SKIP_NETWORK_TESTS") != "" {
		t.Skip("Skipping network test")
	}

	version, err := GetLatestVersion()
	require.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Contains(t, version, "v")
}

func TestProcessSwaggerUI(t *testing.T) {
	// Create a mock swagger-ui zip file
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	files := map[string]string{
		"swagger-ui-5.0.0/dist/index.html":              "<html><head></head><body>Test</body></html>",
		"swagger-ui-5.0.0/dist/swagger-ui.css":          ".swagger-ui {} //# sourceMappingURL=test.map",
		"swagger-ui-5.0.0/dist/swagger-ui-bundle.js":    "window.SwaggerUIBundle = {}; //# sourceMappingURL=bundle.map",
		"swagger-ui-5.0.0/dist/swagger-initializer.js":  "window.onload = function() {};",
		"swagger-ui-5.0.0/dist/test.map":                "source map content",
		"swagger-ui-5.0.0/dist/swagger-ui-es-bundle.js": "es module",
		"swagger-ui-5.0.0/README.md":                    "not included",
	}

	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())

	opts := DownloadOptions{
		CustomCSS:         ".custom {}",
		CustomInitializer: "window.custom = true;",
	}

	result, err := processSwaggerUI(buf.Bytes(), "v5.0.0", opts)
	require.NoError(t, err)

	// Read the result zip
	reader, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	require.NoError(t, err)

	// Check that files are correct
	fileMap := make(map[string]string)
	for _, file := range reader.File {
		rc, err := file.Open()
		require.NoError(t, err)
		var content bytes.Buffer
		_, err = content.ReadFrom(rc)
		require.NoError(t, err)
		_ = rc.Close()
		fileMap[file.Name] = content.String()
	}

	// Check index.html has custom CSS link
	assert.Contains(t, fileMap["index.html"], "custom-styles.css")

	// Check custom initializer was applied
	assert.Equal(t, "window.custom = true;", fileMap["swagger-initializer.js"])

	// Check custom CSS file exists
	assert.Equal(t, ".custom {}", fileMap["custom-styles.css"])

	// Check source maps were removed from content
	assert.NotContains(t, fileMap["swagger-ui.css"], "sourceMappingURL")
	assert.NotContains(t, fileMap["swagger-ui-bundle.js"], "sourceMappingURL")

	// Check .map files were excluded
	_, hasMapFile := fileMap["test.map"]
	assert.False(t, hasMapFile)

	// Check ES module bundle was excluded
	_, hasESBundle := fileMap["swagger-ui-es-bundle.js"]
	assert.False(t, hasESBundle)

	// Check README was not included (not in dist)
	_, hasReadme := fileMap["README.md"]
	assert.False(t, hasReadme)
}

func TestProcessSwaggerUIWithoutCustomizations(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	f, err := w.Create("swagger-ui-5.0.0/dist/index.html")
	require.NoError(t, err)
	_, err = f.Write([]byte("<html><head></head><body>Test</body></html>"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	opts := DownloadOptions{} // No customizations

	result, err := processSwaggerUI(buf.Bytes(), "v5.0.0", opts)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	require.NoError(t, err)

	var hasCustomCSS bool
	for _, file := range reader.File {
		if file.Name == "custom-styles.css" {
			hasCustomCSS = true
		}
	}
	assert.False(t, hasCustomCSS, "Should not have custom CSS when not provided")
}

func TestDownloadWithMockServer(t *testing.T) {
	// Create a mock swagger-ui zip
	var zipBuf bytes.Buffer
	w := zip.NewWriter(&zipBuf)
	f, err := w.Create("swagger-ui-5.0.0/dist/index.html")
	require.NoError(t, err)
	_, err = f.Write([]byte("<html><head></head><body>Mock</body></html>"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipBuf.Bytes())
	}))
	defer server.Close()

	// Create temp directory
	tmpDir := t.TempDir()

	// Note: This test cannot fully test Download() without modifying
	// the DownloadURLTemplate, but we can verify the processing logic works
	_, err = processSwaggerUI(zipBuf.Bytes(), "v5.0.0", DownloadOptions{})
	require.NoError(t, err)

	// Verify temp dir exists
	_, err = os.Stat(tmpDir)
	require.NoError(t, err)
}

func TestDownloadOptions(t *testing.T) {
	opts := DownloadOptions{
		OutputDir:         "/tmp/test",
		CustomCSS:         DefaultCSS,
		CustomInitializer: DefaultInitializer,
		Version:           "5.0.0",
	}

	assert.Equal(t, "/tmp/test", opts.OutputDir)
	assert.Contains(t, opts.CustomCSS, "swagger-ui")
	assert.Contains(t, opts.CustomInitializer, "window.onload")
	assert.Equal(t, "5.0.0", opts.Version)
}
