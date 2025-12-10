package ast

import (
	"sync"
)

// CachedParser is a Parser with caching capabilities
type CachedParser struct {
	*Parser
	mu          sync.RWMutex
	parsedFiles map[string]*ParseResult
}

// NewCachedParser creates a new CachedParser instance
func NewCachedParser() *CachedParser {
	return &CachedParser{
		Parser:      New(),
		parsedFiles: make(map[string]*ParseResult),
	}
}

// Parse parses a Go source file with caching
// If the file has been parsed before, returns the cached result
func (cp *CachedParser) Parse(filename string) (*ParseResult, error) {
	// Check cache first (read lock)
	cp.mu.RLock()
	if cached, ok := cp.parsedFiles[filename]; ok {
		cp.mu.RUnlock()
		return cached, nil
	}
	cp.mu.RUnlock()

	// Parse file (no lock needed for parsing)
	result, err := cp.Parser.Parse(filename)
	if err != nil {
		return nil, err
	}

	// Cache result (write lock)
	cp.mu.Lock()
	cp.parsedFiles[filename] = result
	cp.mu.Unlock()

	return result, nil
}

// ClearCache clears all cached parse results
func (cp *CachedParser) ClearCache() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.parsedFiles = make(map[string]*ParseResult)
}

// ClearFile removes a specific file from the cache
func (cp *CachedParser) ClearFile(filename string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	delete(cp.parsedFiles, filename)
}

// IsCached returns true if the file has been parsed and cached
func (cp *CachedParser) IsCached(filename string) bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	_, ok := cp.parsedFiles[filename]
	return ok
}

// CacheSize returns the number of cached files
func (cp *CachedParser) CacheSize() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return len(cp.parsedFiles)
}

