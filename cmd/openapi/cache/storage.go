package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kausys/apikit/cmd/openapi/spec"
)

// SaveSchema saves a schema to the cache.
// The schema is stored as JSON with unresolved $refs.
func (m *Manager) SaveSchema(name string, schema *spec.Schema) error {
	return m.saveJSON(SchemasDir, name, schema)
}

// LoadSchema loads a schema from the cache.
func (m *Manager) LoadSchema(name string) (*spec.Schema, error) {
	var schema spec.Schema
	if err := m.loadJSON(SchemasDir, name, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// LoadAllSchemas loads all cached schemas.
func (m *Manager) LoadAllSchemas() (map[string]*spec.Schema, error) {
	names, err := m.listCached(SchemasDir)
	if err != nil {
		return nil, err
	}

	schemas := make(map[string]*spec.Schema, len(names))
	for _, name := range names {
		schema, err := m.LoadSchema(name)
		if err != nil {
			return nil, fmt.Errorf("loading schema %s: %w", name, err)
		}
		schemas[name] = schema
	}

	return schemas, nil
}

// SaveRoute saves a route/operation to the cache.
func (m *Manager) SaveRoute(operationID string, operation *spec.Operation) error {
	return m.saveJSON(RoutesDir, operationID, operation)
}

// LoadRoute loads a route/operation from the cache.
func (m *Manager) LoadRoute(operationID string) (*spec.Operation, error) {
	var operation spec.Operation
	if err := m.loadJSON(RoutesDir, operationID, &operation); err != nil {
		return nil, err
	}
	return &operation, nil
}

// LoadAllRoutes loads all cached routes.
func (m *Manager) LoadAllRoutes() (map[string]*spec.Operation, error) {
	names, err := m.listCached(RoutesDir)
	if err != nil {
		return nil, err
	}

	routes := make(map[string]*spec.Operation, len(names))
	for _, name := range names {
		route, err := m.LoadRoute(name)
		if err != nil {
			return nil, fmt.Errorf("loading route %s: %w", name, err)
		}
		routes[name] = route
	}

	return routes, nil
}

// saveJSON saves data as JSON to the specified subdirectory.
func (m *Manager) saveJSON(subdir, name string, data any) error {
	dir := filepath.Join(m.CachePath(), subdir)
	filePath := filepath.Join(dir, name+".json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", name, err)
	}

	return os.WriteFile(filePath, jsonData, 0600)
}

// loadJSON loads JSON data from the specified subdirectory.
func (m *Manager) loadJSON(subdir, name string, dest any) error {
	filePath := filepath.Join(m.CachePath(), subdir, name+".json")

	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// listCached returns all cached item names in a subdirectory.
func (m *Manager) listCached(subdir string) ([]string, error) {
	dir := filepath.Join(m.CachePath(), subdir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			name := entry.Name()[:len(entry.Name())-5] // Remove .json
			names = append(names, name)
		}
	}

	return names, nil
}

// DeleteSchema removes a schema from cache.
func (m *Manager) DeleteSchema(name string) error {
	return os.Remove(filepath.Join(m.CachePath(), SchemasDir, name+".json"))
}

// DeleteRoute removes a route from cache.
func (m *Manager) DeleteRoute(operationID string) error {
	return os.Remove(filepath.Join(m.CachePath(), RoutesDir, operationID+".json"))
}
