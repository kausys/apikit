package sdkgen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SDKGenConfig represents the .sdkgen.yaml configuration file.
type SDKGenConfig struct {
	Provider ProviderConfig     `yaml:"provider"`
	Spec     SpecConfig         `yaml:"spec"`
	Output   OutputConfig       `yaml:"output"`
	Config   ConfigFieldsConfig `yaml:"config"`
	Services ServicesConfig     `yaml:"services"`
	Models   ModelsConfig       `yaml:"models"`
}

// ProviderConfig holds provider identification.
type ProviderConfig struct {
	Name        string `yaml:"name"`        // Package name (e.g., "pokemon")
	DisplayName string `yaml:"displayName"` // For logs, comments (e.g., "Pokemon")
}

// SpecConfig holds OpenAPI spec file location.
type SpecConfig struct {
	Path string `yaml:"path"` // Path to the OpenAPI YAML spec
}

// OutputConfig holds module path for the generated SDK.
type OutputConfig struct {
	ModulePath string `yaml:"modulePath"` // Go module import path for the SDK (e.g., "api/pkg/sdk/pokemon")
}

// ConfigFieldsConfig holds config generation settings.
type ConfigFieldsConfig struct {
	Prefix string             `yaml:"prefix"` // gookit config key prefix (e.g., "pokemon")
	Fields []ConfigFieldEntry `yaml:"fields"` // Config fields
}

// ConfigFieldEntry is a single config field.
type ConfigFieldEntry struct {
	Name    string `yaml:"name"`    // Go field name (e.g., "APIUrl")
	Key     string `yaml:"key"`     // gookit config key suffix (e.g., "apiUrl")
	Type    string `yaml:"type"`    // Go type: "string", "bool", "int", "duration"
	Default string `yaml:"default"` // Default value expression (optional)
}

// ServicesConfig holds service generation configuration.
type ServicesConfig struct {
	ResponseWrapper string                     `yaml:"responseWrapper"` // gjson path to unwrap (empty = use root)
	ParamsStyle     string                     `yaml:"paramsStyle"`     // "inline" (default) or "struct"
	Operations      map[string]OperationConfig `yaml:"operations"`
}

// OperationConfig holds per-operation overrides.
type OperationConfig struct {
	ParamsStyle string `yaml:"paramsStyle"` // "inline" or "struct" — overrides services.params_style
}

// ModelsConfig holds model generation configuration.
type ModelsConfig struct {
	CustomTypes map[string]CustomTypeConfig `yaml:"customTypes"`
}

// CustomTypeConfig maps an OpenAPI format to a Go type.
type CustomTypeConfig struct {
	GoType string `yaml:"goType"` // Full Go type (e.g., "decimal.Decimal")
	Import string `yaml:"import"` // Import path (e.g., "github.com/shopspring/decimal")
}

// LoadSDKGenConfig reads and parses a .sdkgen.yaml file.
func LoadSDKGenConfig(path string) (*SDKGenConfig, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg SDKGenConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// validate checks the config for required fields.
func (c *SDKGenConfig) validate() error {
	if c.Provider.Name == "" {
		return errors.New("provider.name is required")
	}
	if c.Provider.DisplayName == "" {
		c.Provider.DisplayName = toPascalCase(c.Provider.Name)
	}
	if c.Spec.Path == "" {
		return errors.New("spec.path is required")
	}
	if c.Output.ModulePath == "" {
		return errors.New("output.module_path is required")
	}
	if c.Config.Prefix == "" {
		c.Config.Prefix = c.Provider.Name
	}
	if c.Services.ParamsStyle != "" && c.Services.ParamsStyle != "inline" && c.Services.ParamsStyle != "struct" {
		return errors.New("services.params_style must be 'inline' or 'struct'")
	}
	for opID, opCfg := range c.Services.Operations {
		if opCfg.ParamsStyle != "" && opCfg.ParamsStyle != "inline" && opCfg.ParamsStyle != "struct" {
			return fmt.Errorf("services.operations.%s.params_style must be 'inline' or 'struct'", opID)
		}
	}
	return nil
}
