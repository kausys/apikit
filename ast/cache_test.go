package ast

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCachedParser_Parse(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package test

type User struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	parser := NewCachedParser()

	// First parse - should not be cached
	if parser.IsCached(testFile) {
		t.Error("expected file not to be cached initially")
	}

	result1, err := parser.Parse(testFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should be cached now
	if !parser.IsCached(testFile) {
		t.Error("expected file to be cached after parsing")
	}

	if parser.CacheSize() != 1 {
		t.Errorf("expected cache size 1, got %d", parser.CacheSize())
	}

	// Second parse - should return cached result
	result2, err := parser.Parse(testFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should be the same pointer (cached)
	if result1 != result2 {
		t.Error("expected cached result to be the same pointer")
	}

	// Clear specific file
	parser.ClearFile(testFile)
	if parser.IsCached(testFile) {
		t.Error("expected file not to be cached after clearing")
	}

	// Parse again
	result3, err := parser.Parse(testFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should be a different pointer (re-parsed)
	if result1 == result3 {
		t.Error("expected re-parsed result to be a different pointer")
	}

	// Clear all cache
	parser.ClearCache()
	if parser.CacheSize() != 0 {
		t.Errorf("expected cache size 0 after clearing, got %d", parser.CacheSize())
	}
}

func TestCachedParser_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package test

type User struct {
	ID int
}
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	parser := NewCachedParser()

	// Pre-parse to populate cache
	_, err := parser.Parse(testFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Now parse concurrently - all should get cached result
	const numGoroutines = 10
	results := make(chan *ParseResult, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			result, err := parser.Parse(testFile)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}()
	}

	// Collect results
	var firstResult *ParseResult
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			t.Fatalf("Parse failed: %v", err)
		case result := <-results:
			if firstResult == nil {
				firstResult = result
			}
			// All results should be the same (cached)
			if result != firstResult {
				t.Error("expected all concurrent parses to return the same cached result")
			}
		}
	}

	// Should only have one cached file
	if parser.CacheSize() != 1 {
		t.Errorf("expected cache size 1, got %d", parser.CacheSize())
	}
}
