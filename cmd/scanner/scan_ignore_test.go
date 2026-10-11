package scanner

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An --ignore wide enough to exclude every file used to succeed with exit 0 and
// write nothing, leaving whatever the spec files already held on disk. The run
// read as a no-op that had nothing to do.
func TestScanFailsWhenIgnoreExcludesEveryFile(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api
// swagger:route GET /users users listUsers
func ListUsers() {}
`,
	}
	tmpDir := createTestProject(t, files)

	s := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithIgnorePaths("api"),
	)

	err := s.Scan()

	require.Error(t, err, "an ignore that excludes everything must not look like a clean run")
	assert.Contains(t, err.Error(), "excluded every file")
	assert.Contains(t, err.Error(), "0 scanned")
	assert.Empty(t, s.Routes)
}

// A pattern is matched against file paths relative to the scanned directory,
// so the directories above it — where the repository happens to be checked out
// — cannot match. Against the absolute path, a pattern naming the checkout's
// own directory excluded every file on one machine and none on another.
func TestIgnoreDoesNotMatchAboveTheScannedDirectory(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api
// swagger:route GET /users users listUsers
func ListUsers() {}
`,
	}
	tmpDir := createTestProject(t, files)

	// Lines up with .../<tmpdir>/api/handlers.go only through the checkout.
	pattern := filepath.Join(filepath.Base(tmpDir), "api")

	s := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithIgnorePaths(pattern),
	)

	require.NoError(t, s.Scan())
	assert.Contains(t, s.Routes, "listUsers")
}

// No ignore patterns and no directives is a legitimate scan of a tree that has
// nothing to declare — it must stay a success, or the guard above would fail
// every consumer whose pattern happens to match only plain Go code.
func TestScanSucceedsWithNoDirectivesAndNoIgnore(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

func ListUsers() {}
`,
	}
	tmpDir := createTestProject(t, files)

	s := New(WithDir(tmpDir), WithPattern("./..."))

	require.NoError(t, s.Scan())
	assert.Empty(t, s.Routes)
}

// A partially applied ignore is still a real scan: what is left must generate,
// and the excluded half must not raise the every-file error.
func TestScanSucceedsWhenIgnoreExcludesSomeFiles(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api
// swagger:route GET /users users listUsers
func ListUsers() {}
`,
		"vendor/lib/file.go": `package lib
// swagger:route GET /other other listOther
func ListOther() {}
`,
	}
	tmpDir := createTestProject(t, files)

	s := New(WithDir(tmpDir), WithPattern("./..."), WithIgnorePaths("vendor"))

	require.NoError(t, s.Scan())
	assert.Contains(t, s.Routes, "listUsers")
	assert.NotContains(t, s.Routes, "listOther")
}

// --strict reports packages that do not type-check because their routes would be
// dropped. A package the caller excluded has no routes to drop, so it must not
// be reported — and that has to hold for a pattern matched against the file path.
func TestStrictIgnoresBrokenPackageExcludedByFilePath(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api
// swagger:route GET /users users listUsers
func ListUsers() {}
`,
		"legacy/broken.go": `package legacy

func Broken() { undefinedSymbol() }
`,
	}
	tmpDir := createTestProject(t, files)

	// The file path relative to the scanned directory: legacy/broken.go.
	pattern := "legacy/broken.go"

	s := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithStrict(true),
		WithIgnorePaths(pattern),
	)

	err := s.Scan()

	require.NoError(t, err, "a broken package the caller excluded is not a strict failure")
	assert.Contains(t, s.Routes, "listUsers")
}

// The same package, not excluded, still fails under --strict: the guard above
// must not have widened into "ignore every compile error".
func TestStrictStillReportsBrokenPackageThatIsNotIgnored(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api
// swagger:route GET /users users listUsers
func ListUsers() {}
`,
		"legacy/broken.go": `package legacy

func Broken() { undefinedSymbol() }
`,
	}
	tmpDir := createTestProject(t, files)

	s := New(WithDir(tmpDir), WithPattern("./..."), WithStrict(true))

	err := s.Scan()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "do not type-check")
}
