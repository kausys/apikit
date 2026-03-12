package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// CacheDir is the directory where cache files are stored
	CacheDir = ".openapi"
	// IndexFile is the cache index file name
	IndexFile = "index.json"
	// SchemasDir stores cached schema definitions
	SchemasDir = "schemas"
	// RoutesDir stores cached route definitions
	RoutesDir = "routes"
)

// Manager handles all cache operations.
type Manager struct {
	baseDir string
	index   *Index
}

// NewManager creates a new cache manager for the given base directory.
func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir: baseDir,
		index:   NewIndex(),
	}
}

// CachePath returns the full path to the cache directory.
func (m *Manager) CachePath() string {
	return filepath.Join(m.baseDir, CacheDir)
}

// IndexPath returns the full path to the index file.
func (m *Manager) IndexPath() string {
	return filepath.Join(m.CachePath(), IndexFile)
}

// Init initializes the cache directory structure.
func (m *Manager) Init() error {
	dirs := []string{
		m.CachePath(),
		filepath.Join(m.CachePath(), SchemasDir),
		filepath.Join(m.CachePath(), RoutesDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	return nil
}

// Load reads the cache index from disk.
func (m *Manager) Load() error {
	return m.index.Load(m.IndexPath())
}

// Save writes the cache index to disk.
func (m *Manager) Save() error {
	return m.index.Save(m.IndexPath())
}

// Clean removes all cache files.
func (m *Manager) Clean() error {
	return os.RemoveAll(m.CachePath())
}

// NeedsUpdate checks if a source file needs to be re-parsed.
func (m *Manager) NeedsUpdate(filePath string) (bool, error) {
	checksum, err := CalculateChecksum(filePath)
	if err != nil {
		return true, err
	}

	relPath, err := filepath.Rel(m.baseDir, filePath)
	if err != nil {
		relPath = filePath
	}

	return m.index.NeedsUpdate(relPath, checksum), nil
}

// UpdateEntry updates the cache entry for a source file.
// filePath can be either absolute or relative to baseDir.
func (m *Manager) UpdateEntry(filePath string, schemas, routes, parameters []string) error {
	// Convert to absolute path for checksum calculation
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(m.baseDir, filePath)
	}

	checksum, err := CalculateChecksum(absPath)
	if err != nil {
		return err
	}

	// Store as relative path
	relPath, err := filepath.Rel(m.baseDir, absPath)
	if err != nil {
		relPath = filePath
	}

	m.index.SetEntry(relPath, &FileEntry{
		Checksum:   checksum,
		ParsedAt:   time.Now(),
		Schemas:    schemas,
		Routes:     routes,
		Parameters: parameters,
	})

	return nil
}

// GetEntry returns the cache entry for a file.
func (m *Manager) GetEntry(filePath string) (*FileEntry, bool) {
	relPath, err := filepath.Rel(m.baseDir, filePath)
	if err != nil {
		relPath = filePath
	}
	return m.index.GetEntry(relPath)
}

// CacheStats holds statistics about the cache.
type CacheStats struct {
	FileCount      int
	SchemaCount    int
	RouteCount     int
	ParameterCount int
}

// Stats returns statistics about the current cache state.
func (m *Manager) Stats() CacheStats {
	stats := CacheStats{}

	for _, entry := range m.index.Files {
		stats.FileCount++
		stats.SchemaCount += len(entry.Schemas)
		stats.RouteCount += len(entry.Routes)
		stats.ParameterCount += len(entry.Parameters)
	}

	return stats
}
