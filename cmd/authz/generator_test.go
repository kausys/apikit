package authz_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kausys/apikit/cmd/authz"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_DefaultSchema(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "perms.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(`
# resource, action
admins, view and list all admins
users, create new user
users, view and list users
`), 0600))

	outDir := filepath.Join(dir, "out")
	err := authz.Generate(authz.Config{
		InputCSV:    csvPath,
		OutputDir:   outDir,
		PackageName: "mypkg",
	})
	require.NoError(t, err)

	// Only 3 files (no groups)
	assert.FileExists(t, filepath.Join(outDir, "zz_generated_types.go"))
	assert.FileExists(t, filepath.Join(outDir, "zz_generated_resources.go"))
	assert.FileExists(t, filepath.Join(outDir, "zz_generated_actions.go"))
	assert.NoFileExists(t, filepath.Join(outDir, "zz_generated_roles.go"))

	// Types: no Scope, no Role — just Action + Permission with 2 fields
	types := readFile(t, filepath.Join(outDir, "zz_generated_types.go"))
	assert.Contains(t, types, "type Action string")
	assert.Contains(t, types, "type Permission struct")
	assert.Contains(t, types, "resource string")
	assert.Contains(t, types, "action   Action")
	assert.NotContains(t, types, "Scope")
	assert.NotContains(t, types, "Role")
	assert.NotContains(t, types, "kausys/apikit")

	// Resources: no Scope method
	resources := readFile(t, filepath.Join(outDir, "zz_generated_resources.go"))
	assert.Contains(t, resources, "ResourceAdmins")
	assert.Contains(t, resources, "ResourceUsers")
	assert.NotContains(t, resources, "Scope()")
}

func TestGenerate_WithSchema(t *testing.T) {
	dir := t.TempDir()

	schemaPath := filepath.Join(dir, "authz.yaml")
	require.NoError(t, os.WriteFile(schemaPath, []byte(`
csv_columns: [scope, resource, action, roles]
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
`), 0600))

	csvPath := filepath.Join(dir, "perms.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(`
# scope, resource, action, roles
global, admins, view and list all admins, admin
tenant, users, create new user, admin|editor
`), 0600))

	outDir := filepath.Join(dir, "out")
	err := authz.Generate(authz.Config{
		InputCSV:    csvPath,
		OutputDir:   outDir,
		PackageName: "authz",
		SchemaFile:  schemaPath,
	})
	require.NoError(t, err)

	// All 4 files
	for _, name := range []string{"types", "resources", "actions", "roles"} {
		assert.FileExists(t, filepath.Join(outDir, "zz_generated_"+name+".go"))
	}

	// Types: has Scope, Role, Permission with 3 fields
	types := readFile(t, filepath.Join(outDir, "zz_generated_types.go"))
	assert.Contains(t, types, "type Scope string")
	assert.Contains(t, types, `ScopeGlobal Scope = "global"`)
	assert.Contains(t, types, `ScopeTenant Scope = "tenant"`)
	assert.Contains(t, types, "type Role struct")
	assert.Contains(t, types, "scope Scope")

	// Roles: has DefaultRolePolicies with scope switch
	roles := readFile(t, filepath.Join(outDir, "zz_generated_roles.go"))
	assert.Contains(t, roles, "RoleGlobalAdmin")
	assert.Contains(t, roles, "RoleTenantAdmin")
	assert.Contains(t, roles, "RoleTenantEditor")
	assert.Contains(t, roles, "func DefaultRolePolicies(scope Scope)")
	assert.Contains(t, roles, "case ScopeGlobal:")
	assert.Contains(t, roles, "case ScopeTenant:")
}

func TestGenerate_AutoDetectSchema(t *testing.T) {
	dir := t.TempDir()

	// Place authz.yaml next to CSV
	require.NoError(t, os.WriteFile(filepath.Join(dir, "authz.yaml"), []byte(`
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
`), 0600))

	csvPath := filepath.Join(dir, "perms.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(`
admins, view admins, admin
`), 0600))

	outDir := filepath.Join(dir, "out")
	err := authz.Generate(authz.Config{
		InputCSV:    csvPath,
		OutputDir:   outDir,
		PackageName: "pkg",
	})
	require.NoError(t, err)

	// Should have roles file (auto-detected schema has roles group)
	assert.FileExists(t, filepath.Join(outDir, "zz_generated_roles.go"))

	roles := readFile(t, filepath.Join(outDir, "zz_generated_roles.go"))
	assert.Contains(t, roles, "RoleAdmin")
	// No scope → DefaultRolePolicies without parameter
	assert.Contains(t, roles, "func DefaultRolePolicies() map[string][]Permission")
}

func TestGenerate_NoGroups(t *testing.T) {
	dir := t.TempDir()

	schemaPath := filepath.Join(dir, "authz.yaml")
	require.NoError(t, os.WriteFile(schemaPath, []byte(`
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

	csvPath := filepath.Join(dir, "perms.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(`
admins, view admins
users, create user
`), 0600))

	outDir := filepath.Join(dir, "out")
	err := authz.Generate(authz.Config{
		InputCSV:    csvPath,
		OutputDir:   outDir,
		PackageName: "pkg",
		SchemaFile:  schemaPath,
	})
	require.NoError(t, err)

	// Only base files, no group files
	assert.FileExists(t, filepath.Join(outDir, "zz_generated_types.go"))
	assert.FileExists(t, filepath.Join(outDir, "zz_generated_resources.go"))
	assert.FileExists(t, filepath.Join(outDir, "zz_generated_actions.go"))
	assert.NoFileExists(t, filepath.Join(outDir, "zz_generated_roles.go"))

	types := readFile(t, filepath.Join(outDir, "zz_generated_types.go"))
	assert.NotContains(t, types, "Role")
	assert.NotContains(t, types, "Team")
}

func TestGenerate_TwoGroups(t *testing.T) {
	dir := t.TempDir()

	schemaPath := filepath.Join(dir, "authz.yaml")
	require.NoError(t, os.WriteFile(schemaPath, []byte(`
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
    separator: "|"
`), 0600))

	csvPath := filepath.Join(dir, "perms.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(`
# scope, resource, action, roles, teams
global, admins, view admins, admin, backend|frontend
tenant, users, create user, admin|editor, backend
`), 0600))

	outDir := filepath.Join(dir, "out")
	err := authz.Generate(authz.Config{
		InputCSV:    csvPath,
		OutputDir:   outDir,
		PackageName: "authz",
		SchemaFile:  schemaPath,
	})
	require.NoError(t, err)

	// Should have both group files
	assert.FileExists(t, filepath.Join(outDir, "zz_generated_roles.go"))
	assert.FileExists(t, filepath.Join(outDir, "zz_generated_teams.go"))

	// Types: has both Role and Team structs
	types := readFile(t, filepath.Join(outDir, "zz_generated_types.go"))
	assert.Contains(t, types, "type Role struct")
	assert.Contains(t, types, "scope Scope")
	assert.Contains(t, types, "type Team struct")

	// Roles: scope-based
	roles := readFile(t, filepath.Join(outDir, "zz_generated_roles.go"))
	assert.Contains(t, roles, "RoleGlobalAdmin")
	assert.Contains(t, roles, "RoleTenantAdmin")
	assert.Contains(t, roles, "RoleTenantEditor")
	assert.Contains(t, roles, "func DefaultRolePolicies(scope Scope)")
	assert.Contains(t, roles, "func AllRoles() []Role")

	// Teams: no scope
	teams := readFile(t, filepath.Join(outDir, "zz_generated_teams.go"))
	assert.Contains(t, teams, "TeamBackend")
	assert.Contains(t, teams, "TeamFrontend")
	assert.Contains(t, teams, "func AllTeams() []Team")
	assert.Contains(t, teams, "func DefaultTeamPolicies() map[string][]Permission")
}

func TestGenerate_CustomSeparator(t *testing.T) {
	dir := t.TempDir()

	schemaPath := filepath.Join(dir, "authz.yaml")
	require.NoError(t, os.WriteFile(schemaPath, []byte(`
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
    separator: ";"
`), 0600))

	csvPath := filepath.Join(dir, "perms.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(`
admins, view admins, admin;editor
`), 0600))

	outDir := filepath.Join(dir, "out")
	err := authz.Generate(authz.Config{
		InputCSV:    csvPath,
		OutputDir:   outDir,
		PackageName: "pkg",
		SchemaFile:  schemaPath,
	})
	require.NoError(t, err)

	roles := readFile(t, filepath.Join(outDir, "zz_generated_roles.go"))
	assert.Contains(t, roles, "RoleAdmin")
	assert.Contains(t, roles, "RoleEditor")
}

func TestGenerate_InvalidCSVPath(t *testing.T) {
	err := authz.Generate(authz.Config{
		InputCSV:    "/nonexistent/file.csv",
		OutputDir:   t.TempDir(),
		PackageName: "pkg",
	})
	assert.Error(t, err)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // test helper with trusted paths
	require.NoError(t, err)
	return string(data)
}
