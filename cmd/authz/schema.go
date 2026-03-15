package authz

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Schema defines the structure of the authz code generation.
type Schema struct {
	CSVColumns []string         `yaml:"csv_columns"` //nolint:tagliatelle // matches YAML convention
	Permission PermissionSchema `yaml:"permission"`
	Groups     []GroupSchema    `yaml:"groups"`
}

// PermissionSchema defines the fields of the Permission struct.
type PermissionSchema struct {
	Fields []FieldSchema `yaml:"fields"`
}

// FieldSchema defines a single field in the Permission struct.
type FieldSchema struct {
	Name   string `yaml:"name"`   // Go field name (lowercase in Permission struct)
	Type   string `yaml:"type"`   // "string" or a custom type name like "Action", "Scope"
	Column string `yaml:"column"` // CSV column name that feeds this field
}

// GroupSchema defines how a group (roles, teams, etc.) is parsed and generated.
type GroupSchema struct {
	Name        string `yaml:"name"`         // e.g. "roles", "teams"
	Column      string `yaml:"column"`       // CSV column for this group
	Separator   string `yaml:"separator"`    // value separator (default: "|")
	ScopeColumn string `yaml:"scope_column"` //nolint:tagliatelle // matches YAML convention
}

// TypeName derives the singular Go type name: "roles" -> "Role", "teams" -> "Team".
func (g GroupSchema) TypeName() string {
	return toCamelCase(strings.TrimSuffix(g.Name, "s"))
}

// PluralName returns the CamelCase plural name: "roles" -> "Roles".
func (g GroupSchema) PluralName() string {
	return toCamelCase(g.Name)
}

// GetSeparator returns the separator, defaulting to "|".
func (g GroupSchema) GetSeparator() string {
	if g.Separator == "" {
		return "|"
	}
	return g.Separator
}

// DefaultSchema returns the minimal default schema (resource + action, no groups).
func DefaultSchema() *Schema {
	return &Schema{
		CSVColumns: []string{"resource", "action"},
		Permission: PermissionSchema{
			Fields: []FieldSchema{
				{Name: "resource", Type: "string", Column: "resource"},
				{Name: "action", Type: "Action", Column: "action"},
			},
		},
	}
}

// LoadSchema reads and parses a schema YAML file.
func LoadSchema(path string) (*Schema, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading schema file: %w", err)
	}

	var schema Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("parsing schema file: %w", err)
	}

	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	return &schema, nil
}

// FindSchemaFile looks for an authz.yaml file next to the given CSV path.
func FindSchemaFile(csvPath string) string {
	dir := filepath.Dir(csvPath)
	candidate := filepath.Join(dir, "authz.yaml")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

// Validate checks that the schema is internally consistent.
func (s *Schema) Validate() error {
	if len(s.CSVColumns) == 0 {
		return errors.New("csv_columns must not be empty")
	}

	if len(s.Permission.Fields) == 0 {
		return errors.New("permission.fields must not be empty")
	}

	// Must have exactly one Action field
	actionCount := 0
	for _, f := range s.Permission.Fields {
		if f.Type == "Action" {
			actionCount++
		}
	}
	if actionCount != 1 {
		return fmt.Errorf("permission.fields must have exactly one field with type Action, got %d", actionCount)
	}

	// Must have a resource column
	hasResource := false
	for _, f := range s.Permission.Fields {
		if f.Column == "resource" {
			hasResource = true
			break
		}
	}
	if !hasResource {
		return errors.New("permission.fields must include a field with column \"resource\"")
	}

	// All field columns must exist in csv_columns (excluding "_" skip columns)
	colSet := make(map[string]bool)
	for _, c := range s.CSVColumns {
		if c != "_" {
			colSet[c] = true
		}
	}
	for _, f := range s.Permission.Fields {
		if !colSet[f.Column] {
			return fmt.Errorf("field %q references column %q which is not in csv_columns", f.Name, f.Column)
		}
	}

	// Validate groups
	groupNames := make(map[string]bool)
	for _, g := range s.Groups {
		if g.Name == "" {
			return errors.New("group name must not be empty")
		}
		if groupNames[g.Name] {
			return fmt.Errorf("duplicate group name %q", g.Name)
		}
		groupNames[g.Name] = true

		if g.Column == "" {
			return fmt.Errorf("group %q must have a column", g.Name)
		}
		if !colSet[g.Column] {
			return fmt.Errorf("group %q column %q is not in csv_columns", g.Name, g.Column)
		}

		if g.ScopeColumn != "" {
			found := false
			for _, f := range s.Permission.Fields {
				if f.Column == g.ScopeColumn {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("group %q scope_column %q does not match any permission field column", g.Name, g.ScopeColumn)
			}
		}
	}

	return nil
}

// ActionField returns the field with type "Action".
func (s *Schema) ActionField() *FieldSchema {
	for i := range s.Permission.Fields {
		if s.Permission.Fields[i].Type == "Action" {
			return &s.Permission.Fields[i]
		}
	}
	return nil
}

// CustomTypes returns fields with types other than "string" (these generate Go types).
func (s *Schema) CustomTypes() []FieldSchema {
	var types []FieldSchema
	for _, f := range s.Permission.Fields {
		if f.Type != "string" {
			types = append(types, f)
		}
	}
	return types
}

// ScopeFieldFor returns the permission field referenced by the group's scope_column, or nil.
func (s *Schema) ScopeFieldFor(g GroupSchema) *FieldSchema {
	if g.ScopeColumn == "" {
		return nil
	}
	for i := range s.Permission.Fields {
		if s.Permission.Fields[i].Column == g.ScopeColumn {
			return &s.Permission.Fields[i]
		}
	}
	return nil
}

// ScopeFields returns all unique permission fields referenced as scope_column by any group.
func (s *Schema) ScopeFields() []*FieldSchema {
	seen := make(map[string]bool)
	var fields []*FieldSchema
	for _, g := range s.Groups {
		if g.ScopeColumn != "" && !seen[g.ScopeColumn] {
			seen[g.ScopeColumn] = true
			if f := s.ScopeFieldFor(g); f != nil {
				fields = append(fields, f)
			}
		}
	}
	return fields
}

// HasGroups returns true if any groups are configured.
func (s *Schema) HasGroups() bool {
	return len(s.Groups) > 0
}

// ColumnIndex returns the index of a column in csv_columns, or -1 if not found.
func (s *Schema) ColumnIndex(name string) int {
	for i, c := range s.CSVColumns {
		if c == name {
			return i
		}
	}
	return -1
}
