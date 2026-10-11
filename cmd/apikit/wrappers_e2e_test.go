package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kausys/apikit/cmd/handler/parser"
)

// TestGeneratedWrappersServeRequests generates the wrappers of a fixture
// module, compiles them against the runtime in this tree, and runs requests
// through them. Reading generated code proves little; a wrapper that does not
// compile, or compiles and binds nothing, fails here.
//
// The fixture covers what used to break: pointer parameters (generated code
// did not compile), defaults of named types (pasted into the code as Go), a
// payload in another file or package (refused), a body tag with options (not
// read), rules on the body's fields (not checked), a body over the limit (cut
// short), and the JSON v2 reading.
func TestGeneratedWrappersServeRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles and runs a module")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go toolchain")
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	copyTree(t, filepath.Join("testdata", "wrappers"), dir)
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/wrappers\n\ngo 1.27\n\n"+
		"require github.com/kausys/apikit v0.0.0\n\n"+
		"replace github.com/kausys/apikit => "+filepath.ToSlash(repoRoot)+"\n")
	if err := os.Rename(filepath.Join(dir, "api", "wrappers_test.go.txt"), filepath.Join(dir, "api", "wrappers_test.go")); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-mod=mod")
	run := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), goBin, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			wrapper, _ := os.ReadFile(filepath.Join(dir, "api", "handler_apikit.go"))
			t.Fatalf("go %s: %v\n%s\n--- handler_apikit.go ---\n%s", strings.Join(args, " "), err, out, wrapper)
		}
	}
	// The fixture must resolve before it is type-checked for generation.
	run("mod", "tidy")

	p := parser.New()
	for _, src := range []string{"handler.go", "types.go"} {
		if _, err := handlerGenerateWithParser(p, filepath.Join(dir, "api", src)); err != nil {
			t.Fatalf("generating %s: %v", src, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "api", "types_apikit.go")); err == nil {
		t.Error("a file without handlers got a wrapper")
	}

	run("mod", "tidy")
	run("vet", "./...")
	run("test", "-count=1", "./...")
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		content, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
