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

// The shape this was found in: a pattern that only ever lines up with the
// absolute file path, never with the import path. The consumer's checkout was
// .../platform/phoenix/phoenix, so `--ignore phoenix/phoenix` matched every file
// of the inner tree while `testproject/...`-style import paths never contain the
// doubled segment.
func TestScanFailsWhenIgnoreMatchesOnlyAbsolutePaths(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api
// swagger:route GET /users users listUsers
func ListUsers() {}
`,
	}
	tmpDir := createTestProject(t, files)

	// Matches .../<tmpdir>/api/handlers.go, never the import path testproject/api.
	pattern := filepath.Join(filepath.Base(tmpDir), "api")

	s := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithIgnorePaths(pattern),
	)

	err := s.Scan()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "excluded every file")
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
// be reported — and that has to hold for a pattern matched against the file path,
// which is the only kind the phoenix case had.
func TestStrictIgnoresBrokenPackageExcludedByAbsolutePath(t *testing.T) {
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

	// Absolute-path-only: the import path is testproject/legacy.
	pattern := filepath.Join(filepath.Base(tmpDir), "legacy")

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
