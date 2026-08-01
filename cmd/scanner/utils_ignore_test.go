package scanner

import "testing"

// shouldIgnorePath used to be strings.Contains over the raw path, which made an
// --ignore pattern match on any substring — including one that only lined up
// because of where the repo happened to be checked out.
func TestShouldIgnorePathMatchesWholeSegments(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		{"absolute file path", "/src/repo/foo/bar/handler.go", "foo/bar", true},
		{"package import path", "mod/foo/bar", "foo/bar", true},
		{"single segment", "/src/vendor/x.go", "vendor", true},
		{"trailing slash in pattern", "/src/foo/bar/x.go", "foo/bar/", true},
		{"exact tail", "/src/foo/bar", "foo/bar", true},

		{"partial segment prefix", "/src/myfoo/barbecue/x.go", "foo/bar", false},
		{"partial segment suffix", "/src/foox/xbar/x.go", "foo/bar", false},
		{"segments present but not adjacent", "/src/foo/mid/bar/x.go", "foo/bar", false},
		{"substring inside one segment", "/src/foobar/x.go", "foo/bar", false},
		{"unrelated", "mod/other/pkg", "foo/bar", false},
		{"empty pattern is not a wildcard", "/src/anything", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIgnorePath(tt.path, []string{tt.pattern}); got != tt.want {
				t.Errorf("shouldIgnorePath(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

// The nested-module case this was written for: phoenix checks out as
// .../platform/phoenix/phoenix, and the legacy pass ignores the inner copy.
func TestShouldIgnorePathHandlesRepeatedSegment(t *testing.T) {
	inner := "/Users/x/platform/phoenix/phoenix/app-api/kyb/handler.go"
	outer := "/Users/x/platform/phoenix/internal/kyc/handler.go"

	if !shouldIgnorePath(inner, []string{"phoenix/phoenix"}) {
		t.Error("inner tree should be ignored")
	}
	if shouldIgnorePath(outer, []string{"phoenix/phoenix"}) {
		t.Error("outer tree must not be ignored")
	}
	// The package path never contains the doubled segment, which is why the
	// ignore has to work on file paths too.
	if shouldIgnorePath("api/phoenix/app-api/kyb", []string{"phoenix/phoenix"}) {
		t.Error("package path must not match")
	}
}
