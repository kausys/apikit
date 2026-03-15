package authz_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kausys/apikit/cmd/authz"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSchema_GroupsArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "authz.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
csv_columns: [scope, resource, action, roles, teams]
permission:
  fields:
    - name: resource
      type: string
      column: resource
    - name: action
      type: Action
      column: action
    - name: scope
      type: Scope
      column: scope
groups:
  - name: roles
    column: roles
    separator: "|"
    scope_column: scope
  - name: teams
    column: teams
`), 0600))

	schema, err := authz.LoadSchema(path)
	require.NoError(t, err)

	assert.Len(t, schema.Groups, 2)
	assert.Equal(t, "roles", schema.Groups[0].Name)
	assert.Equal(t, "roles", schema.Groups[0].Column)
	assert.Equal(t, "|", schema.Groups[0].Separator)
	assert.Equal(t, "scope", schema.Groups[0].ScopeColumn)
	assert.Equal(t, "teams", schema.Groups[1].Name)
	assert.Equal(t, "teams", schema.Groups[1].Column)
	assert.Empty(t, schema.Groups[1].Separator)
	assert.Empty(t, schema.Groups[1].ScopeColumn)
}

func TestLoadSchema_EmptyGroups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "authz.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
csv_columns: [resource, action]
permission:
  fields:
    - name: resource
      type: string
      column: resource
    - name: action
      type: Action
      column: action
groups: []
`), 0600))

	schema, err := authz.LoadSchema(path)
	require.NoError(t, err)
	assert.Empty(t, schema.Groups)
	assert.False(t, schema.HasGroups())
}

func TestGroupSchema_TypeName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"roles", "Role"},
		{"teams", "Team"},
		{"profiles", "Profile"},
		{"staff", "Staff"},
	}
	for _, tt := range tests {
		g := authz.GroupSchema{Name: tt.name, Column: tt.name}
		assert.Equal(t, tt.expected, g.TypeName(), "TypeName for %q", tt.name)
	}
}

func TestGroupSchema_PluralName(t *testing.T) {
	g := authz.GroupSchema{Name: "roles", Column: "roles"}
	assert.Equal(t, "Roles", g.PluralName())

	g2 := authz.GroupSchema{Name: "teams", Column: "teams"}
	assert.Equal(t, "Teams", g2.PluralName())
}

func TestGroupSchema_GetSeparator(t *testing.T) {
	g := authz.GroupSchema{Name: "roles", Column: "roles"}
	assert.Equal(t, "|", g.GetSeparator(), "default separator")

	g2 := authz.GroupSchema{Name: "roles", Column: "roles", Separator: ";"}
	assert.Equal(t, ";", g2.GetSeparator(), "custom separator")
}

func TestSchema_Validate_DuplicateGroupName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "authz.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
csv_columns: [resource, action, roles]
permission:
  fields:
    - name: resource
      type: string
      column: resource
    - name: action
      type: Action
      column: action
groups:
  - name: roles
    column: roles
  - name: roles
    column: roles
`), 0600))

	_, err := authz.LoadSchema(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate group name")
}

func TestSchema_Validate_GroupColumnNotInCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "authz.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
csv_columns: [resource, action]
permission:
  fields:
    - name: resource
      type: string
      column: resource
    - name: action
      type: Action
      column: action
groups:
  - name: roles
    column: roles
`), 0600))

	_, err := authz.LoadSchema(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not in csv_columns")
}

func TestSchema_Validate_GroupScopeColumnInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "authz.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
csv_columns: [resource, action, roles]
permission:
  fields:
    - name: resource
      type: string
      column: resource
    - name: action
      type: Action
      column: action
groups:
  - name: roles
    column: roles
    scope_column: scope
`), 0600))

	_, err := authz.LoadSchema(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scope_column")
}

func TestSchema_Validate_GroupEmptyName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "authz.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
csv_columns: [resource, action, roles]
permission:
  fields:
    - name: resource
      type: string
      column: resource
    - name: action
      type: Action
      column: action
groups:
  - name: ""
    column: roles
`), 0600))

	_, err := authz.LoadSchema(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group name must not be empty")
}
