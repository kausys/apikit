package generator

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kausys/apikit/openapi/spec"

	"gopkg.in/yaml.v3"
)

// ConfigFile represents the .openapi.yaml configuration file.
type ConfigFile struct {
	CustomTypes map[string]TypeConfig  `yaml:"customTypes"`
	Schemas     map[string]spec.Schema `yaml:"schemas"`
}

// TypeConfig represents a custom type configuration in the config file.
type TypeConfig struct {
	Type    string `yaml:"type"`
	Format  string `yaml:"format"`
	Example any    `yaml:"example"`
	Default any    `yaml:"default"`
}

var (
	externalSchemas   = make(map[string]*spec.Schema)
	externalSchemasMu sync.RWMutex
)

// RegisterSchema registers an external schema by name.
func RegisterSchema(name string, schema *spec.Schema) {
	externalSchemasMu.Lock()
	defer externalSchemasMu.Unlock()
	externalSchemas[name] = schema
}

// GetExternalSchema returns the external schema for the given name.
func GetExternalSchema(name string) *spec.Schema {
	externalSchemasMu.RLock()
	defer externalSchemasMu.RUnlock()
	return externalSchemas[name]
}

// GetAllExternalSchemas returns a copy of all external schemas.
func GetAllExternalSchemas() map[string]*spec.Schema {
	externalSchemasMu.RLock()
	defer externalSchemasMu.RUnlock()
	result := make(map[string]*spec.Schema, len(externalSchemas))
	maps.Copy(result, externalSchemas)
	return result
}

// ClearExternalSchemas removes all registered external schemas.
func ClearExternalSchemas() {
	externalSchemasMu.Lock()
	defer externalSchemasMu.Unlock()
	externalSchemas = make(map[string]*spec.Schema)
}

// LoadConfigFile loads custom types and schemas from .openapi.yaml in the given directory.
// It searches for .openapi.yaml, .openapi.yml, or openapi.config.yaml.
func LoadConfigFile(dir string) error {
	configNames := []string{
		".openapi.yaml",
		".openapi.yml",
		"openapi.config.yaml",
		"openapi.config.yml",
	}

	var configPath string
	for _, name := range configNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return nil // No config file, not an error
	}

	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return err
	}

	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	// Register custom types
	for typeName, typeConfig := range config.CustomTypes {
		info := &TypeInfo{
			Type:    typeConfig.Type,
			Format:  typeConfig.Format,
			Example: typeConfig.Example,
			Default: typeConfig.Default,
		}
		RegisterTypeInfo(typeName, info)

		// If fully-qualified path (e.g., "github.com/shopspring/decimal.Decimal"),
		// also register under the short name ("decimal.Decimal") as fallback.
		if strings.Contains(typeName, "/") {
			parts := strings.Split(typeName, "/")
			shortName := parts[len(parts)-1]
			if GetCustomType(shortName) == nil {
				RegisterTypeInfo(shortName, info)
			}
		}
	}

	// Register external schemas
	for name, schema := range config.Schemas {
		s := schema // copy to avoid pointer to loop variable
		RegisterSchema(name, &s)
	}

	return nil
}
