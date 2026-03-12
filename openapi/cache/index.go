package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Index represents the cache index that tracks all cached files.
type Index struct {
	// Version of the cache format
	Version string `json:"version"`
	// CreatedAt is when the cache was first created
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is when the cache was last updated
	UpdatedAt time.Time `json:"updatedAt"`
	// Files maps source file paths to their cache entries
	Files map[string]*FileEntry `json:"files"`
}

// FileEntry represents a cached entry for a single source file.
type FileEntry struct {
	// Checksum is the SHA256 hash of the source file
	Checksum string `json:"checksum"`
	// ParsedAt is when the file was last parsed
	ParsedAt time.Time `json:"parsedAt"`
	// Schemas exported by this file (schema names)
	Schemas []string `json:"schemas,omitempty"`
	// Routes exported by this file (operation IDs)
	Routes []string `json:"routes,omitempty"`
	// Parameters exported by this file
	Parameters []string `json:"parameters,omitempty"`
}

// NewIndex creates a new empty cache index.
func NewIndex() *Index {
	return &Index{
		Version:   "1.0.0",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Files:     make(map[string]*FileEntry),
	}
}

// Load reads the index from disk.
func (idx *Index) Load(path string) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache yet, use empty index
		}
		return err
	}

	return json.Unmarshal(data, idx)
}

// Save writes the index to disk.
func (idx *Index) Save(path string) error {
	idx.UpdatedAt = time.Now()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// NeedsUpdate checks if a source file needs to be re-parsed.
func (idx *Index) NeedsUpdate(relPath, checksum string) bool {
	entry, ok := idx.Files[relPath]
	if !ok {
		return true // Not in cache
	}
	return entry.Checksum != checksum
}

// SetEntry updates or creates a cache entry for a file.
func (idx *Index) SetEntry(relPath string, entry *FileEntry) {
	idx.Files[relPath] = entry
}

// GetEntry returns the cache entry for a file.
func (idx *Index) GetEntry(relPath string) (*FileEntry, bool) {
	entry, ok := idx.Files[relPath]
	return entry, ok
}

// RemoveEntry removes a cache entry.
func (idx *Index) RemoveEntry(relPath string) {
	delete(idx.Files, relPath)
}
