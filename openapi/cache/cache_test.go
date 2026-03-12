package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kausys/apikit/openapi/spec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Checksum Tests ====================

func TestCalculateChecksum(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	require.NoError(t, os.WriteFile(testFile, content, 0600))

	checksum, err := CalculateChecksum(testFile)

	require.NoError(t, err)
	assert.Contains(t, checksum, "sha256:")
	assert.Len(t, checksum, 71) // "sha256:" + 64 hex chars
}

func TestCalculateChecksumFileNotFound(t *testing.T) {
	_, err := CalculateChecksum("/nonexistent/file.txt")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "opening file")
}

func TestCalculateContentChecksum(t *testing.T) {
	content := []byte("hello world")

	checksum := CalculateContentChecksum(content)

	assert.Contains(t, checksum, "sha256:")
	assert.Len(t, checksum, 71)
}

func TestCalculateContentChecksumDeterministic(t *testing.T) {
	content := []byte("same content")

	checksum1 := CalculateContentChecksum(content)
	checksum2 := CalculateContentChecksum(content)

	assert.Equal(t, checksum1, checksum2)
}

// ==================== Index Tests ====================

func TestNewIndex(t *testing.T) {
	idx := NewIndex()

	assert.Equal(t, "1.0.0", idx.Version)
	assert.NotNil(t, idx.Files)
	assert.NotZero(t, idx.CreatedAt)
	assert.NotZero(t, idx.UpdatedAt)
}

func TestIndexSetAndGetEntry(t *testing.T) {
	idx := NewIndex()
	entry := &FileEntry{
		Checksum: "sha256:abc123",
		Schemas:  []string{"User"},
		Routes:   []string{"getUsers"},
	}

	idx.SetEntry("api/models.go", entry)
	retrieved, ok := idx.GetEntry("api/models.go")

	assert.True(t, ok)
	assert.Equal(t, entry, retrieved)
}

func TestIndexGetEntryNotFound(t *testing.T) {
	idx := NewIndex()

	_, ok := idx.GetEntry("nonexistent.go")

	assert.False(t, ok)
}

func TestIndexRemoveEntry(t *testing.T) {
	idx := NewIndex()
	idx.SetEntry("test.go", &FileEntry{Checksum: "abc"})

	idx.RemoveEntry("test.go")
	_, ok := idx.GetEntry("test.go")

	assert.False(t, ok)
}

func TestIndexNeedsUpdate(t *testing.T) {
	tests := []struct {
		name        string
		setupEntry  *FileEntry
		checksum    string
		needsUpdate bool
	}{
		{
			name:        "not in cache",
			setupEntry:  nil,
			checksum:    "sha256:new",
			needsUpdate: true,
		},
		{
			name:        "checksum matches",
			setupEntry:  &FileEntry{Checksum: "sha256:abc"},
			checksum:    "sha256:abc",
			needsUpdate: false,
		},
		{
			name:        "checksum differs",
			setupEntry:  &FileEntry{Checksum: "sha256:old"},
			checksum:    "sha256:new",
			needsUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewIndex()
			if tt.setupEntry != nil {
				idx.SetEntry("test.go", tt.setupEntry)
			}

			result := idx.NeedsUpdate("test.go", tt.checksum)

			assert.Equal(t, tt.needsUpdate, result)
		})
	}
}

func TestIndexSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "cache", "index.json")

	// Create and save
	idx := NewIndex()
	idx.SetEntry("api/handlers.go", &FileEntry{
		Checksum: "sha256:test",
		Schemas:  []string{"User"},
	})
	require.NoError(t, idx.Save(indexPath))

	// Load into new index
	loadedIdx := NewIndex()
	require.NoError(t, loadedIdx.Load(indexPath))

	assert.Equal(t, idx.Version, loadedIdx.Version)
	entry, ok := loadedIdx.GetEntry("api/handlers.go")
	assert.True(t, ok)
	assert.Equal(t, "sha256:test", entry.Checksum)
}

func TestIndexLoadNonExistent(t *testing.T) {
	idx := NewIndex()

	err := idx.Load("/nonexistent/index.json")

	assert.NoError(t, err) // Returns nil for non-existent file
}

// ==================== Manager Tests ====================

func TestNewManager(t *testing.T) {
	base := filepath.Join(t.TempDir(), "dir")
	m := NewManager(base)

	assert.NotNil(t, m)
	assert.Equal(t, filepath.Join(base, ".openapi"), m.CachePath())
	assert.Equal(t, filepath.Join(base, ".openapi", "index.json"), m.IndexPath())
}

func TestManagerInit(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	err := m.Init()

	require.NoError(t, err)
	assert.DirExists(t, m.CachePath())
	assert.DirExists(t, filepath.Join(m.CachePath(), SchemasDir))
	assert.DirExists(t, filepath.Join(m.CachePath(), RoutesDir))
}

func TestManagerSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())

	// Add entry and save
	testFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package main"), 0600))
	require.NoError(t, m.UpdateEntry(testFile, []string{"User"}, []string{"getUsers"}, nil))
	require.NoError(t, m.Save())

	// Create new manager and load
	m2 := NewManager(tmpDir)
	require.NoError(t, m2.Load())

	entry, ok := m2.GetEntry(testFile)
	assert.True(t, ok)
	assert.Contains(t, entry.Checksum, "sha256:")
}

func TestManagerClean(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())
	assert.DirExists(t, m.CachePath())

	err := m.Clean()

	require.NoError(t, err)
	assert.NoDirExists(t, m.CachePath())
}

func TestManagerNeedsUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())

	testFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package main"), 0600))

	// First check - not in cache
	needsUpdate, err := m.NeedsUpdate(testFile)
	require.NoError(t, err)
	assert.True(t, needsUpdate)

	// Add to cache
	require.NoError(t, m.UpdateEntry(testFile, nil, nil, nil))

	// Second check - in cache, unchanged
	needsUpdate, err = m.NeedsUpdate(testFile)
	require.NoError(t, err)
	assert.False(t, needsUpdate)

	// Modify file
	require.NoError(t, os.WriteFile(testFile, []byte("package changed"), 0600))

	// Third check - changed
	needsUpdate, err = m.NeedsUpdate(testFile)
	require.NoError(t, err)
	assert.True(t, needsUpdate)
}

func TestManagerNeedsUpdateFileNotFound(t *testing.T) {
	m := NewManager(t.TempDir())

	_, err := m.NeedsUpdate(filepath.Join(t.TempDir(), "nonexistent", "file.go"))

	assert.Error(t, err)
}

func TestManagerStats(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())

	testFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package main"), 0600))
	require.NoError(t, m.UpdateEntry(testFile, []string{"User", "Post"}, []string{"getUsers"}, []string{"PageParams"}))

	stats := m.Stats()

	assert.Equal(t, 1, stats.FileCount)
	assert.Equal(t, 2, stats.SchemaCount)
	assert.Equal(t, 1, stats.RouteCount)
	assert.Equal(t, 1, stats.ParameterCount)
}

// ==================== Storage Tests ====================

func TestManagerSaveAndLoadSchema(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())

	schema := &spec.Schema{
		Type:        spec.NewSchemaType("object"),
		Description: "A user object",
	}

	err := m.SaveSchema("User", schema)
	require.NoError(t, err)

	loaded, err := m.LoadSchema("User")
	require.NoError(t, err)
	assert.Equal(t, "object", loaded.Type.Value())
	assert.Equal(t, "A user object", loaded.Description)
}

func TestManagerLoadSchemaNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())

	_, err := m.LoadSchema("NonExistent")

	assert.Error(t, err)
}

func TestManagerLoadAllSchemas(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())

	require.NoError(t, m.SaveSchema("User", &spec.Schema{Type: spec.NewSchemaType("object")}))
	require.NoError(t, m.SaveSchema("Post", &spec.Schema{Type: spec.NewSchemaType("object")}))

	schemas, err := m.LoadAllSchemas()

	require.NoError(t, err)
	assert.Len(t, schemas, 2)
	assert.Contains(t, schemas, "User")
	assert.Contains(t, schemas, "Post")
}

func TestManagerSaveAndLoadRoute(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())

	operation := &spec.Operation{
		Summary:     "Get users",
		OperationID: "getUsers",
	}

	err := m.SaveRoute("getUsers", operation)
	require.NoError(t, err)

	loaded, err := m.LoadRoute("getUsers")
	require.NoError(t, err)
	assert.Equal(t, "Get users", loaded.Summary)
	assert.Equal(t, "getUsers", loaded.OperationID)
}

func TestManagerLoadAllRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())

	require.NoError(t, m.SaveRoute("getUsers", &spec.Operation{OperationID: "getUsers"}))
	require.NoError(t, m.SaveRoute("createUser", &spec.Operation{OperationID: "createUser"}))

	routes, err := m.LoadAllRoutes()

	require.NoError(t, err)
	assert.Len(t, routes, 2)
	assert.Contains(t, routes, "getUsers")
	assert.Contains(t, routes, "createUser")
}

func TestManagerDeleteSchema(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())
	require.NoError(t, m.SaveSchema("User", &spec.Schema{Type: spec.NewSchemaType("object")}))

	err := m.DeleteSchema("User")
	require.NoError(t, err)

	_, err = m.LoadSchema("User")
	assert.Error(t, err)
}

func TestManagerDeleteRoute(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	require.NoError(t, m.Init())
	require.NoError(t, m.SaveRoute("getUsers", &spec.Operation{OperationID: "getUsers"}))

	err := m.DeleteRoute("getUsers")
	require.NoError(t, err)

	_, err = m.LoadRoute("getUsers")
	assert.Error(t, err)
}

func TestManagerLoadAllSchemasEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	// Don't init - directory doesn't exist

	schemas, err := m.LoadAllSchemas()

	require.NoError(t, err)
	assert.Empty(t, schemas)
}

func TestManagerLoadAllRoutesEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	// Don't init - directory doesn't exist

	routes, err := m.LoadAllRoutes()

	require.NoError(t, err)
	assert.Empty(t, routes)
}
