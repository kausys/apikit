package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFiles lays out a fake source tree under a temp root.
func writeFiles(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("package x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func rel(t *testing.T, root string, files []string) []string {
	t.Helper()
	out := make([]string, 0, len(files))
	for _, f := range files {
		r, err := filepath.Rel(root, f)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, filepath.ToSlash(r))
	}
	return out
}

func TestExpandSourceArgsRecursivePattern(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"auth/staff.go",
		"auth/staff_apikit.go", // generated — skipped
		"auth/staff_test.go",   // test — skipped
		"staff/me.go",
		"wiring/bridge.go",
		"testdata/fixture.go", // testdata — skipped
		".hidden/file.go",     // hidden dir — skipped
		"generate.go",
	)

	files, err := expandSourceArgs(root, []string{"./..."})
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	got := rel(t, root, files)
	want := map[string]bool{"auth/staff.go": true, "staff/me.go": true, "wiring/bridge.go": true, "generate.go": true}
	if len(got) != len(want) {
		t.Fatalf("expected %d files, got %v", len(want), got)
	}
	for _, f := range got {
		if !want[f] {
			t.Fatalf("unexpected file %s in %v", f, got)
		}
	}
}

func TestExpandSourceArgsSubtreePattern(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "auth/staff.go", "auth/inner/deep.go", "staff/me.go")

	files, err := expandSourceArgs(root, []string{"auth/..."})
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	got := rel(t, root, files)
	if len(got) != 2 || got[0] != "auth/inner/deep.go" && got[1] != "auth/inner/deep.go" {
		t.Fatalf("expected auth subtree only, got %v", got)
	}
	for _, f := range got {
		if !strings.HasPrefix(f, "auth/") {
			t.Fatalf("file outside subtree: %s", f)
		}
	}
}

func TestExpandSourceArgsDirectoryNonRecursive(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "auth/staff.go", "auth/inner/deep.go")

	files, err := expandSourceArgs(root, []string{"auth"})
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	got := rel(t, root, files)
	if len(got) != 1 || got[0] != "auth/staff.go" {
		t.Fatalf("expected only auth/staff.go, got %v", got)
	}
}

func TestExpandSourceArgsExplicitFilesUntouched(t *testing.T) {
	root := t.TempDir()
	// Explicit files bypass the generated/test filters — caller knows best.
	writeFiles(t, root, "auth/staff_apikit.go")

	files, err := expandSourceArgs(root, []string{"auth/staff_apikit.go"})
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	got := rel(t, root, files)
	if len(got) != 1 || got[0] != "auth/staff_apikit.go" {
		t.Fatalf("expected explicit file passthrough, got %v", got)
	}
}

func TestExpandSourceArgsMissingFile(t *testing.T) {
	root := t.TempDir()
	if _, err := expandSourceArgs(root, []string{"nope.go"}); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestExpandSourceArgsEmptyPattern(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "auth/staff_apikit.go") // only generated files
	if _, err := expandSourceArgs(root, []string{"./..."}); err == nil {
		t.Fatal("expected error when a pattern matches no source files")
	}
}
