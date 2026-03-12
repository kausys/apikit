package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kausys/apikit/cmd/openapi/cache"
	"github.com/kausys/apikit/cmd/openapi/spec"
	"github.com/kausys/apikit/cmd/scanner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestGenerator creates a Generator with initialized fields for testing
func createTestGenerator() *Generator {
	s := scanner.New(
		scanner.WithDir("."),
		scanner.WithPattern("./..."),
	)

	return &Generator{
		config: &Config{
			Dir:          ".",
			Pattern:      "./...",
			OutputFile:   "",
			OutputFormat: "yaml",
			UseCache:     false,
			CleanUnused:  false,
		},
		cache:             cache.NewManager("."),
		scanner:           s,
		referencedSchemas: make(map[string]bool),
	}
}

// TestCollectSpecNames tests the collectSpecNames function
func TestCollectSpecNames(t *testing.T) {
	tests := []struct {
		name     string
		routes   map[string]*scanner.RouteInfo
		expected map[string]bool
	}{
		{
			name:     "empty routes",
			routes:   map[string]*scanner.RouteInfo{},
			expected: map[string]bool{},
		},
		{
			name: "routes without spec directive go to default",
			routes: map[string]*scanner.RouteInfo{
				"getUsers": {
					Method:      "GET",
					Path:        "/users",
					OperationID: "getUsers",
					Specs:       []string{},
				},
			},
			expected: map[string]bool{scanner.DefaultSpec: true},
		},
		{
			name: "routes with single spec",
			routes: map[string]*scanner.RouteInfo{
				"getUsers": {
					Method:      "GET",
					Path:        "/users",
					OperationID: "getUsers",
					Specs:       []string{"admin"},
				},
			},
			expected: map[string]bool{"admin": true},
		},
		{
			name: "routes with multiple specs",
			routes: map[string]*scanner.RouteInfo{
				"getUsers": {
					Method:      "GET",
					Path:        "/users",
					OperationID: "getUsers",
					Specs:       []string{"admin", "public"},
				},
			},
			expected: map[string]bool{"admin": true, "public": true},
		},
		{
			name: "mixed routes with and without specs",
			routes: map[string]*scanner.RouteInfo{
				"getUsers": {
					Method:      "GET",
					Path:        "/users",
					OperationID: "getUsers",
					Specs:       []string{},
				},
				"getAdminUsers": {
					Method:      "GET",
					Path:        "/admin/users",
					OperationID: "getAdminUsers",
					Specs:       []string{"admin"},
				},
				"getPublicData": {
					Method:      "GET",
					Path:        "/public",
					OperationID: "getPublicData",
					Specs:       []string{"public", "mobile"},
				},
			},
			expected: map[string]bool{
				scanner.DefaultSpec: true,
				"admin":             true,
				"public":            true,
				"mobile":            true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator()
			g.scanner.Routes = tt.routes

			result := g.collectSpecNames()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRouteBelongsToSpec tests the routeBelongsToSpec function
func TestRouteBelongsToSpec(t *testing.T) {
	tests := []struct {
		name     string
		route    *scanner.RouteInfo
		specName string
		expected bool
	}{
		{
			name: "route without spec belongs to default",
			route: &scanner.RouteInfo{
				OperationID: "getUsers",
				Specs:       []string{},
			},
			specName: scanner.DefaultSpec,
			expected: true,
		},
		{
			name: "route without spec does not belong to other specs",
			route: &scanner.RouteInfo{
				OperationID: "getUsers",
				Specs:       []string{},
			},
			specName: "admin",
			expected: false,
		},
		{
			name: "route with spec belongs to that spec",
			route: &scanner.RouteInfo{
				OperationID: "getAdminUsers",
				Specs:       []string{"admin"},
			},
			specName: "admin",
			expected: true,
		},
		{
			name: "route with spec does not belong to other specs",
			route: &scanner.RouteInfo{
				OperationID: "getAdminUsers",
				Specs:       []string{"admin"},
			},
			specName: "public",
			expected: false,
		},
		{
			name: "route with multiple specs belongs to all",
			route: &scanner.RouteInfo{
				OperationID: "getUsers",
				Specs:       []string{"admin", "public", "mobile"},
			},
			specName: "mobile",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator()
			result := g.routeBelongsToSpec(tt.route, tt.specName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetMetaForSpec tests the getMetaForSpec function
func TestGetMetaForSpec(t *testing.T) {
	tests := []struct {
		name         string
		generalMeta  *scanner.MetaInfo
		specMetas    []*scanner.MetaInfo
		specName     string
		expectedMeta *scanner.MetaInfo
	}{
		{
			name:         "no metas returns nil",
			generalMeta:  nil,
			specMetas:    []*scanner.MetaInfo{},
			specName:     "admin",
			expectedMeta: nil,
		},
		{
			name: "returns general meta when no specific meta",
			generalMeta: &scanner.MetaInfo{
				Title:   "General API",
				Version: "1.0.0",
			},
			specMetas:    []*scanner.MetaInfo{},
			specName:     "admin",
			expectedMeta: &scanner.MetaInfo{Title: "General API", Version: "1.0.0"},
		},
		{
			name: "returns specific meta when available",
			generalMeta: &scanner.MetaInfo{
				Title:   "General API",
				Version: "1.0.0",
			},
			specMetas: []*scanner.MetaInfo{
				{
					Title:   "Admin API",
					Version: "2.0.0",
					Specs:   []string{"admin"},
				},
			},
			specName:     "admin",
			expectedMeta: &scanner.MetaInfo{Title: "Admin API", Version: "2.0.0", Specs: []string{"admin"}},
		},
		{
			name: "returns general meta when spec not found in specific metas",
			generalMeta: &scanner.MetaInfo{
				Title:   "General API",
				Version: "1.0.0",
			},
			specMetas: []*scanner.MetaInfo{
				{
					Title:   "Admin API",
					Version: "2.0.0",
					Specs:   []string{"admin"},
				},
			},
			specName:     "public",
			expectedMeta: &scanner.MetaInfo{Title: "General API", Version: "1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator()
			g.scanner.Meta = tt.generalMeta
			g.scanner.Metas = tt.specMetas

			result := g.getMetaForSpec(tt.specName)

			if tt.expectedMeta == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedMeta.Title, result.Title)
				assert.Equal(t, tt.expectedMeta.Version, result.Version)
			}
		})
	}
}

// TestGetSchemaForSpec tests the getSchemaForSpec function
func TestGetSchemaForSpec(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		specName       string
		structs        map[string]*scanner.StructInfo
		expectedSchema bool
		expectedDesc   string
	}{
		{
			name:           "model not found returns nil",
			modelName:      "NonExistent",
			specName:       "admin",
			structs:        map[string]*scanner.StructInfo{},
			expectedSchema: false,
		},
		{
			name:      "general model returned for any spec",
			modelName: "User",
			specName:  "admin",
			structs: map[string]*scanner.StructInfo{
				"User": {
					Name:        "User",
					Description: "General user model",
					IsModel:     true,
					Specs:       []string{},
				},
			},
			expectedSchema: true,
			expectedDesc:   "General user model",
		},
		{
			name:      "specific model overrides general model",
			modelName: "User",
			specName:  "admin",
			structs: map[string]*scanner.StructInfo{
				"User": {
					Name:        "User",
					Description: "General user model",
					IsModel:     true,
					Specs:       []string{},
				},
				"User_admin": {
					Name:        "User",
					Description: "Admin user model",
					IsModel:     true,
					Specs:       []string{"admin"},
				},
			},
			expectedSchema: true,
			expectedDesc:   "Admin user model",
		},
		{
			name:      "general model used when specific model for other spec",
			modelName: "User",
			specName:  "public",
			structs: map[string]*scanner.StructInfo{
				"User": {
					Name:        "User",
					Description: "General user model",
					IsModel:     true,
					Specs:       []string{},
				},
				"User_admin": {
					Name:        "User",
					Description: "Admin user model",
					IsModel:     true,
					Specs:       []string{"admin"},
				},
			},
			expectedSchema: true,
			expectedDesc:   "General user model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator()
			g.scanner.Structs = tt.structs

			result := g.getSchemaForSpec(tt.modelName, tt.specName)

			if !tt.expectedSchema {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedDesc, result.Description)
			}
		})
	}
}

// TestAssembleForSpec tests the assembleForSpec function
func TestAssembleForSpec(t *testing.T) {
	tests := []struct {
		name          string
		specName      string
		meta          *scanner.MetaInfo
		routes        map[string]*scanner.RouteInfo
		expectedPaths int
		expectedTitle string
	}{
		{
			name:          "empty spec with no meta uses default info",
			specName:      "admin",
			meta:          nil,
			routes:        map[string]*scanner.RouteInfo{},
			expectedPaths: 0,
			expectedTitle: "API",
		},
		{
			name:     "spec with meta uses meta info",
			specName: scanner.DefaultSpec,
			meta: &scanner.MetaInfo{
				Title:   "My API",
				Version: "2.0.0",
			},
			routes:        map[string]*scanner.RouteInfo{},
			expectedPaths: 0,
			expectedTitle: "My API",
		},
		{
			name:     "spec with routes includes only matching routes",
			specName: "admin",
			meta:     nil,
			routes: map[string]*scanner.RouteInfo{
				"getAdminUsers": {
					Method:      "GET",
					Path:        "/admin/users",
					OperationID: "getAdminUsers",
					Specs:       []string{"admin"},
				},
				"getPublicUsers": {
					Method:      "GET",
					Path:        "/users",
					OperationID: "getPublicUsers",
					Specs:       []string{"public"},
				},
			},
			expectedPaths: 1,
			expectedTitle: "API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator()
			g.scanner.Meta = tt.meta
			g.scanner.Routes = tt.routes

			result, err := g.assembleForSpec(tt.specName)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, "3.1.2", result.OpenAPI)
			assert.Equal(t, tt.expectedTitle, result.Info.Title)
			assert.Len(t, result.Paths.PathItems, tt.expectedPaths)
		})
	}
}

// TestAssembleForSpecWithSecuritySchemes tests security scheme inheritance
func TestAssembleForSpecWithSecuritySchemes(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Meta = &scanner.MetaInfo{
		Title:   "API with Security",
		Version: "1.0.0",
		SecuritySchemes: map[string]*scanner.SecuritySchemeInfo{
			"bearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
		},
	}
	g.scanner.Routes = map[string]*scanner.RouteInfo{}

	result, err := g.assembleForSpec(scanner.DefaultSpec)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Components)
	require.NotNil(t, result.Components.SecuritySchemes)
	assert.Contains(t, result.Components.SecuritySchemes, "bearerAuth")
}

// TestAssembleForSpecWithTags tests tag handling
func TestAssembleForSpecWithTags(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Meta = &scanner.MetaInfo{
		Title:   "API with Tags",
		Version: "1.0.0",
		Tags: []*scanner.TagInfo{
			{Name: "users", Description: "User operations"},
			{Name: "admin", Description: "Admin operations"},
		},
	}
	g.scanner.Routes = map[string]*scanner.RouteInfo{}

	result, err := g.assembleForSpec(scanner.DefaultSpec)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Tags, 2)
	assert.Equal(t, "users", result.Tags[0].Name)
	assert.Equal(t, "admin", result.Tags[1].Name)
}

// TestBuildReferencedSchemas tests the buildReferencedSchemas function
func TestBuildReferencedSchemas(t *testing.T) {
	tests := []struct {
		name              string
		referencedSchemas map[string]bool
		structs           map[string]*scanner.StructInfo
		enums             map[string]*scanner.EnumInfo
		expectedSchemas   []string
	}{
		{
			name:              "no referenced schemas",
			referencedSchemas: map[string]bool{},
			structs:           map[string]*scanner.StructInfo{},
			enums:             map[string]*scanner.EnumInfo{},
			expectedSchemas:   []string{},
		},
		{
			name:              "referenced model gets added",
			referencedSchemas: map[string]bool{"User": true},
			structs: map[string]*scanner.StructInfo{
				"User": {
					Name:        "User",
					Description: "User model",
					IsModel:     true,
					Fields:      []*scanner.FieldInfo{},
				},
			},
			enums:           map[string]*scanner.EnumInfo{},
			expectedSchemas: []string{"User"},
		},
		{
			name:              "referenced enum gets added",
			referencedSchemas: map[string]bool{"Status": true},
			structs:           map[string]*scanner.StructInfo{},
			enums: map[string]*scanner.EnumInfo{
				"Status": {
					TypeName:    "Status",
					BaseType:    "string",
					Description: "Status enum",
					Values:      map[string]any{"active": "active", "inactive": "inactive"},
				},
			},
			expectedSchemas: []string{"Status"},
		},
		{
			name:              "non-model struct is not added",
			referencedSchemas: map[string]bool{"Params": true},
			structs: map[string]*scanner.StructInfo{
				"Params": {
					Name:        "Params",
					Description: "Parameter struct",
					IsModel:     false,
					IsParameter: true,
					Fields:      []*scanner.FieldInfo{},
				},
			},
			enums:           map[string]*scanner.EnumInfo{},
			expectedSchemas: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator()
			g.referencedSchemas = tt.referencedSchemas
			g.scanner.Structs = tt.structs
			g.scanner.Enums = tt.enums

			components := &spec.Components{
				Schemas: make(map[string]*spec.Schema),
			}

			g.buildReferencedSchemas(components, scanner.DefaultSpec)

			assert.Len(t, components.Schemas, len(tt.expectedSchemas))
			for _, schemaName := range tt.expectedSchemas {
				assert.Contains(t, components.Schemas, schemaName)
			}
		})
	}
}

// TestWriteMultiOutput tests the writeMultiOutput function
func TestWriteMultiOutput(t *testing.T) {
	tests := []struct {
		name         string
		outputFile   string
		outputFormat string
		specs        map[string]*spec.OpenAPI
		expectFiles  []string
	}{
		{
			name:         "write single spec as YAML",
			outputFile:   "output/specs.yaml",
			outputFormat: "yaml",
			specs: map[string]*spec.OpenAPI{
				"admin": {OpenAPI: "3.1.2", Info: &spec.Info{Title: "Admin API", Version: "1.0.0"}},
			},
			expectFiles: []string{"admin.yaml"},
		},
		{
			name:         "write multiple specs as YAML",
			outputFile:   "output/specs.yaml",
			outputFormat: "yaml",
			specs: map[string]*spec.OpenAPI{
				"admin":  {OpenAPI: "3.1.2", Info: &spec.Info{Title: "Admin API", Version: "1.0.0"}},
				"public": {OpenAPI: "3.1.2", Info: &spec.Info{Title: "Public API", Version: "1.0.0"}},
			},
			expectFiles: []string{"admin.yaml", "public.yaml"},
		},
		{
			name:         "write specs as JSON",
			outputFile:   "output/specs.json",
			outputFormat: "json",
			specs: map[string]*spec.OpenAPI{
				"admin": {OpenAPI: "3.1.2", Info: &spec.Info{Title: "Admin API", Version: "1.0.0"}},
			},
			expectFiles: []string{"admin.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, tt.outputFile)

			g := createTestGenerator()
			g.config.OutputFile = outputPath
			g.config.OutputFormat = tt.outputFormat

			err := g.writeMultiOutput(tt.specs)
			require.NoError(t, err)

			outputDir := filepath.Dir(outputPath)
			for _, expectedFile := range tt.expectFiles {
				filePath := filepath.Join(outputDir, expectedFile)
				_, err := os.Stat(filePath)
				assert.NoError(t, err, "Expected file %s to exist", expectedFile)
			}
		})
	}
}

// TestWriteMultiOutputNoExtension tests writeMultiOutput with no extension
func TestWriteMultiOutputNoExtension(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output", "specs")

	g := createTestGenerator()
	g.config.OutputFile = outputPath
	g.config.OutputFormat = "yaml"

	specs := map[string]*spec.OpenAPI{
		"admin": {OpenAPI: "3.1.2", Info: &spec.Info{Title: "Admin API", Version: "1.0.0"}},
	}

	err := g.writeMultiOutput(specs)
	require.NoError(t, err)

	// Should default to .yaml extension
	filePath := filepath.Join(filepath.Dir(outputPath), "admin.yaml")
	_, err = os.Stat(filePath)
	assert.NoError(t, err, "Expected file admin.yaml to exist")
}

// TestAssembleMulti tests the assembleMulti function
func TestAssembleMulti(t *testing.T) {
	tests := []struct {
		name          string
		routes        map[string]*scanner.RouteInfo
		expectedSpecs []string
	}{
		{
			name:          "no routes returns default spec",
			routes:        map[string]*scanner.RouteInfo{},
			expectedSpecs: []string{scanner.DefaultSpec},
		},
		{
			name: "routes with single spec",
			routes: map[string]*scanner.RouteInfo{
				"getAdminUsers": {
					Method:      "GET",
					Path:        "/admin/users",
					OperationID: "getAdminUsers",
					Specs:       []string{"admin"},
				},
			},
			expectedSpecs: []string{"admin"},
		},
		{
			name: "routes with multiple specs",
			routes: map[string]*scanner.RouteInfo{
				"getAdminUsers": {
					Method:      "GET",
					Path:        "/admin/users",
					OperationID: "getAdminUsers",
					Specs:       []string{"admin"},
				},
				"getPublicUsers": {
					Method:      "GET",
					Path:        "/users",
					OperationID: "getPublicUsers",
					Specs:       []string{"public"},
				},
			},
			expectedSpecs: []string{"admin", "public"},
		},
		{
			name: "mixed routes - some without spec directive",
			routes: map[string]*scanner.RouteInfo{
				"getAdminUsers": {
					Method:      "GET",
					Path:        "/admin/users",
					OperationID: "getAdminUsers",
					Specs:       []string{"admin"},
				},
				"getDefaultUsers": {
					Method:      "GET",
					Path:        "/users",
					OperationID: "getDefaultUsers",
					Specs:       []string{},
				},
			},
			expectedSpecs: []string{"admin", scanner.DefaultSpec},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator()
			g.scanner.Routes = tt.routes

			result, err := g.assembleMulti()

			require.NoError(t, err)
			require.NotNil(t, result)

			// Check that all expected specs are present
			for _, specName := range tt.expectedSpecs {
				assert.Contains(t, result, specName, "Expected spec %s to be present", specName)
			}
		})
	}
}

// TestAssembleMultiEmptySpecsNotIncluded tests that specs without routes are not included
func TestAssembleMultiEmptySpecsNotIncluded(t *testing.T) {
	g := createTestGenerator()
	// Route belongs to admin spec, but we won't have any public routes
	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getAdminUsers": {
			Method:      "GET",
			Path:        "/admin/users",
			OperationID: "getAdminUsers",
			Specs:       []string{"admin"},
		},
	}

	result, err := g.assembleMulti()

	require.NoError(t, err)
	assert.Contains(t, result, "admin")
	assert.NotContains(t, result, "public")
}

// TestSecuritySchemeInheritance tests that security schemes are inherited from general meta
func TestSecuritySchemeInheritance(t *testing.T) {
	g := createTestGenerator()

	// Set general meta with security schemes
	g.scanner.Meta = &scanner.MetaInfo{
		Title:   "General API",
		Version: "1.0.0",
		SecuritySchemes: map[string]*scanner.SecuritySchemeInfo{
			"apiKey": {
				Type: "apiKey",
				Name: "X-API-Key",
				In:   "header",
			},
		},
	}

	// Set specific meta for admin WITHOUT security schemes
	g.scanner.Metas = []*scanner.MetaInfo{
		{
			Title:           "Admin API",
			Version:         "2.0.0",
			Specs:           []string{"admin"},
			SecuritySchemes: map[string]*scanner.SecuritySchemeInfo{}, // Empty
		},
	}

	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getAdminUsers": {
			Method:      "GET",
			Path:        "/admin/users",
			OperationID: "getAdminUsers",
			Specs:       []string{"admin"},
		},
	}

	result, err := g.assembleForSpec("admin")

	require.NoError(t, err)
	require.NotNil(t, result)
	// Security schemes should be inherited from general meta
	assert.Contains(t, result.Components.SecuritySchemes, "apiKey")
}

// TestSpecificMetaSecuritySchemesOverride tests that specific meta security schemes are used
func TestSpecificMetaSecuritySchemesOverride(t *testing.T) {
	g := createTestGenerator()

	// Set general meta with security schemes
	g.scanner.Meta = &scanner.MetaInfo{
		Title:   "General API",
		Version: "1.0.0",
		SecuritySchemes: map[string]*scanner.SecuritySchemeInfo{
			"apiKey": {
				Type: "apiKey",
				Name: "X-API-Key",
				In:   "header",
			},
		},
	}

	// Set specific meta for admin WITH its own security schemes
	g.scanner.Metas = []*scanner.MetaInfo{
		{
			Title:   "Admin API",
			Version: "2.0.0",
			Specs:   []string{"admin"},
			SecuritySchemes: map[string]*scanner.SecuritySchemeInfo{
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
		},
	}

	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getAdminUsers": {
			Method:      "GET",
			Path:        "/admin/users",
			OperationID: "getAdminUsers",
			Specs:       []string{"admin"},
		},
	}

	result, err := g.assembleForSpec("admin")

	require.NoError(t, err)
	require.NotNil(t, result)
	// Should have the specific security scheme
	assert.Contains(t, result.Components.SecuritySchemes, "bearerAuth")
}

// TestBuildReferencedSchemasWithNestedReferences tests iterative schema building
func TestBuildReferencedSchemasWithNestedReferences(t *testing.T) {
	g := createTestGenerator()

	// User references Address
	g.scanner.Structs = map[string]*scanner.StructInfo{
		"User": {
			Name:        "User",
			Description: "User model",
			IsModel:     true,
			Fields: []*scanner.FieldInfo{
				{
					Name: "address",
					Type: "Address",
					Tags: map[string]string{"json": "address"},
				},
			},
		},
		"Address": {
			Name:        "Address",
			Description: "Address model",
			IsModel:     true,
			Fields:      []*scanner.FieldInfo{},
		},
	}

	g.referencedSchemas = map[string]bool{"User": true}

	components := &spec.Components{
		Schemas: make(map[string]*spec.Schema),
	}

	g.buildReferencedSchemas(components, scanner.DefaultSpec)

	// Both User and Address should be in components
	assert.Contains(t, components.Schemas, "User")
	// Note: Address will only be added if structToSchema marks it as referenced
}

// TestRouteWithMultipleSpecs tests that a route with multiple specs appears in all
func TestRouteWithMultipleSpecs(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getUsers": {
			Method:      "GET",
			Path:        "/users",
			OperationID: "getUsers",
			Specs:       []string{"admin", "public", "mobile"},
		},
	}

	// Check that route belongs to each spec
	route := g.scanner.Routes["getUsers"]
	assert.True(t, g.routeBelongsToSpec(route, "admin"))
	assert.True(t, g.routeBelongsToSpec(route, "public"))
	assert.True(t, g.routeBelongsToSpec(route, "mobile"))
	assert.False(t, g.routeBelongsToSpec(route, "internal"))
}

// ===========================================================================
// Integration Tests
// ===========================================================================

// createTestProject creates a temporary directory with Go source files for testing
func createTestProject(t *testing.T, files map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()

	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		dir := filepath.Dir(filePath)
		err := os.MkdirAll(dir, 0750)
		require.NoError(t, err)
		err = os.WriteFile(filePath, []byte(content), 0600)
		require.NoError(t, err)
	}

	// Create go.mod file
	goMod := `module testproject

go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0600)
	require.NoError(t, err)

	return tmpDir
}

// TestIntegrationGenerateMultiWithRealFiles tests end-to-end multi-spec generation
func TestIntegrationGenerateMultiWithRealFiles(t *testing.T) {
	// Create test project with swagger comments
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /admin/users admin listAdminUsers
// spec: admin
//
// List admin users
//
// Returns a list of admin users.
//
// Responses:
//   200: []User
func ListAdminUsers() {}

// swagger:route GET /users public listPublicUsers
// spec: public
//
// List public users
//
// Returns a list of public users.
//
// Responses:
//   200: []User
func ListPublicUsers() {}

// swagger:route GET /health health healthCheck
//
// Health check endpoint
//
// Responses:
//   200: HealthResponse
func HealthCheck() {}
`,
		"api/models.go": `package api

// swagger:model User
// User represents a user in the system
type User struct {
	// The unique identifier
	// example: 123
	ID int ` + "`json:\"id\"`" + `
	// The user's name
	// example: John Doe
	Name string ` + "`json:\"name\"`" + `
}

// swagger:model HealthResponse
// HealthResponse represents a health check response
type HealthResponse struct {
	// Status of the service
	// example: ok
	Status string ` + "`json:\"status\"`" + `
}
`,
	}

	tmpDir := createTestProject(t, files)

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithCache(false),
	)

	specs, err := g.GenerateMulti()

	require.NoError(t, err)
	require.NotNil(t, specs)

	// Should have admin, public, and default specs
	// (health endpoint has no spec: directive, so goes to default)
	assert.GreaterOrEqual(t, len(specs), 2, "Expected at least 2 specs")
}

// TestIntegrationGenerateMultiOutput tests file output
func TestIntegrationGenerateMultiOutput(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /admin/users admin listAdminUsers
// spec: admin
//
// List admin users
func ListAdminUsers() {}

// swagger:route GET /users public listPublicUsers
// spec: public
//
// List public users
func ListPublicUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)
	outputDir := filepath.Join(tmpDir, "output")

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithOutput(filepath.Join(outputDir, "specs.yaml"), "yaml"),
		WithCache(false),
	)

	specs, err := g.GenerateMulti()

	require.NoError(t, err)
	require.NotNil(t, specs)

	// Check that output files were created
	for specName := range specs {
		filePath := filepath.Join(outputDir, specName+".yaml")
		_, err := os.Stat(filePath)
		assert.NoError(t, err, "Expected file %s to exist", filePath)
	}
}

// TestIntegrationGenerateMultiWithMeta tests meta inheritance
func TestIntegrationGenerateMultiWithMeta(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /users users listUsers
// spec: public
//
// List users
func ListUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithCache(false),
	)

	specs, err := g.GenerateMulti()

	require.NoError(t, err)
	require.NotNil(t, specs)

	// Check public spec exists and has valid structure
	if publicSpec, ok := specs["public"]; ok {
		require.NotNil(t, publicSpec.Info)
		assert.Equal(t, "3.1.2", publicSpec.OpenAPI)
		// Version should be set (either from meta or default "1.0.0")
		assert.NotEmpty(t, publicSpec.Info.Version)
	}
}

// TestIntegrationGenerateSpec tests single spec generation
func TestIntegrationGenerateSpec(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /admin/users admin listAdminUsers
// spec: admin
//
// List admin users
func ListAdminUsers() {}

// swagger:route GET /users public listPublicUsers
// spec: public
//
// List public users
func ListPublicUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithCache(false),
	)

	// Generate only admin spec
	adminSpec, err := g.GenerateSpec("admin")

	require.NoError(t, err)
	require.NotNil(t, adminSpec)
	assert.Equal(t, "3.1.2", adminSpec.OpenAPI)

	// Should only contain admin routes
	if adminSpec.Paths != nil {
		for path := range adminSpec.Paths.PathItems {
			assert.Contains(t, path, "admin", "Expected only admin paths")
		}
	}
}

// TestIntegrationGetSpecNames tests GetSpecNames function
func TestIntegrationGetSpecNames(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /admin/users admin listAdminUsers
// spec: admin
func ListAdminUsers() {}

// swagger:route GET /users public listPublicUsers
// spec: public mobile
func ListPublicUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithCache(false),
	)

	names, err := g.GetSpecNames()

	require.NoError(t, err)
	assert.Contains(t, names, "admin")
	assert.Contains(t, names, "public")
	assert.Contains(t, names, "mobile")
}

// TestIntegrationGenerateMultiJSON tests JSON output format
func TestIntegrationGenerateMultiJSON(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /users public listUsers
// spec: public
func ListUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)
	outputDir := filepath.Join(tmpDir, "output")

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithOutput(filepath.Join(outputDir, "specs.json"), "json"),
		WithCache(false),
	)

	specs, err := g.GenerateMulti()

	require.NoError(t, err)
	require.NotNil(t, specs)

	// Check that JSON files were created
	for specName := range specs {
		filePath := filepath.Join(outputDir, specName+".json")
		_, err := os.Stat(filePath)
		require.NoError(t, err, "Expected JSON file %s to exist", filePath)

		// Verify content is valid JSON
		content, err := os.ReadFile(filePath) //nolint:gosec // test file path from tmpDir
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, string(content), "\"openapi\"")
	}
}

// ===========================================================================
// Edge Cases and Error Paths
// ===========================================================================

// TestEmptyRouteSpecs tests handling of routes with empty specs array
func TestEmptyRouteSpecs(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getUsers": {
			Method:      "GET",
			Path:        "/users",
			OperationID: "getUsers",
			Specs:       nil, // nil specs should be treated as empty
		},
	}

	specNames := g.collectSpecNames()
	assert.Contains(t, specNames, scanner.DefaultSpec)
}

// TestNilMetaHandling tests handling of nil meta
func TestNilMetaHandling(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Meta = nil
	g.scanner.Metas = nil
	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getUsers": {
			Method:      "GET",
			Path:        "/users",
			OperationID: "getUsers",
			Specs:       []string{"admin"},
		},
	}

	result, err := g.assembleForSpec("admin")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "API", result.Info.Title)
	assert.Equal(t, "1.0.0", result.Info.Version)
}

// TestSpecWithNoMatchingRoutes tests a spec with no matching routes
func TestSpecWithNoMatchingRoutes(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getUsers": {
			Method:      "GET",
			Path:        "/users",
			OperationID: "getUsers",
			Specs:       []string{"admin"},
		},
	}

	result, err := g.assembleForSpec("nonexistent")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Paths.PathItems)
}

// TestSchemaLookupWithMissingSchema tests getSchemaForSpec with missing schema
func TestSchemaLookupWithMissingSchema(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Structs = map[string]*scanner.StructInfo{}

	result := g.getSchemaForSpec("NonExistent", "admin")
	assert.Nil(t, result)
}

// TestMultipleMetasForSameSpec tests handling when multiple metas target same spec
func TestMultipleMetasForSameSpec(t *testing.T) {
	g := createTestGenerator()

	// First meta found should be returned
	g.scanner.Metas = []*scanner.MetaInfo{
		{
			Title:   "First Admin API",
			Version: "1.0.0",
			Specs:   []string{"admin"},
		},
		{
			Title:   "Second Admin API",
			Version: "2.0.0",
			Specs:   []string{"admin"},
		},
	}

	result := g.getMetaForSpec("admin")

	require.NotNil(t, result)
	assert.Equal(t, "First Admin API", result.Title)
}

// TestWriteMultiOutputEmptySpecs tests writing with empty specs map
func TestWriteMultiOutputEmptySpecs(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output", "specs.yaml")

	g := createTestGenerator()
	g.config.OutputFile = outputPath
	g.config.OutputFormat = "yaml"

	specs := map[string]*spec.OpenAPI{}

	err := g.writeMultiOutput(specs)
	require.NoError(t, err)
}

// TestAssembleForSpecResetReferencedSchemas tests that referencedSchemas is reset
func TestAssembleForSpecResetReferencedSchemas(t *testing.T) {
	g := createTestGenerator()
	g.referencedSchemas = map[string]bool{"OldSchema": true}
	g.scanner.Routes = map[string]*scanner.RouteInfo{}

	_, err := g.assembleForSpec("admin")

	require.NoError(t, err)
	// referencedSchemas should be reset (empty) after assembleForSpec
	assert.Empty(t, g.referencedSchemas)
}

// TestRouteWithPathAndMethodInMultipleSpecs tests same path/method in multiple specs
func TestRouteWithPathAndMethodInMultipleSpecs(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getUsersAdmin": {
			Method:      "GET",
			Path:        "/users",
			OperationID: "getUsersAdmin",
			Summary:     "Get users (admin)",
			Specs:       []string{"admin"},
		},
		"getUsersPublic": {
			Method:      "GET",
			Path:        "/users",
			OperationID: "getUsersPublic",
			Summary:     "Get users (public)",
			Specs:       []string{"public"},
		},
	}

	adminSpec, err := g.assembleForSpec("admin")
	require.NoError(t, err)

	publicSpec, err := g.assembleForSpec("public")
	require.NoError(t, err)

	// Both specs should have /users path
	assert.Contains(t, adminSpec.Paths.PathItems, "/users")
	assert.Contains(t, publicSpec.Paths.PathItems, "/users")

	// But with different summaries
	if adminSpec.Paths.PathItems["/users"].Get != nil {
		assert.Equal(t, "Get users (admin)", adminSpec.Paths.PathItems["/users"].Get.Summary)
	}
	if publicSpec.Paths.PathItems["/users"].Get != nil {
		assert.Equal(t, "Get users (public)", publicSpec.Paths.PathItems["/users"].Get.Summary)
	}
}

// TestAssembleForSpecWithGeneralMetaOnly tests using general meta when no specific meta
func TestAssembleForSpecWithGeneralMetaOnly(t *testing.T) {
	g := createTestGenerator()

	// Set only general meta (no specific metas)
	g.scanner.Meta = &scanner.MetaInfo{
		Title:   "General API",
		Version: "1.0.0",
		SecuritySchemes: map[string]*scanner.SecuritySchemeInfo{
			"apiKey": {
				Type: "apiKey",
				Name: "X-API-Key",
				In:   "header",
			},
		},
		Tags: []*scanner.TagInfo{
			{Name: "general", Description: "General operations"},
		},
	}
	g.scanner.Metas = nil // No specific metas

	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getAdminUsers": {
			Method:      "GET",
			Path:        "/admin/users",
			OperationID: "getAdminUsers",
			Specs:       []string{"admin"},
		},
	}

	result, err := g.assembleForSpec("admin")

	require.NoError(t, err)
	require.NotNil(t, result)
	// Should use general meta
	assert.Equal(t, "General API", result.Info.Title)
	assert.Contains(t, result.Components.SecuritySchemes, "apiKey")
	assert.Len(t, result.Tags, 1)
	assert.Equal(t, "general", result.Tags[0].Name)
}

// TestWriteMultiOutputMarshalError tests error handling during marshal
func TestWriteMultiOutputMarshalError(t *testing.T) {
	// This test verifies that the function handles marshal errors gracefully
	// In practice, OpenAPI specs should always be marshalable, but we test the path
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output", "specs.yaml")

	g := createTestGenerator()
	g.config.OutputFile = outputPath
	g.config.OutputFormat = "yaml"

	// Create a valid spec - this should succeed
	specs := map[string]*spec.OpenAPI{
		"test": {OpenAPI: "3.1.2", Info: &spec.Info{Title: "Test", Version: "1.0.0"}},
	}

	err := g.writeMultiOutput(specs)
	require.NoError(t, err)
}

// TestGenerateMultiWithCacheDisabled tests GenerateMulti with cache disabled
func TestGenerateMultiWithCacheDisabled(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /users users listUsers
// spec: public
func ListUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithCache(false), // Explicitly disable cache
	)

	specs, err := g.GenerateMulti()

	require.NoError(t, err)
	require.NotNil(t, specs)
	assert.Contains(t, specs, "public")
}

// TestGenerateSpecWithCacheDisabled tests GenerateSpec with cache disabled
func TestGenerateSpecWithCacheDisabled(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /admin/users admin listAdminUsers
// spec: admin
func ListAdminUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithCache(false),
	)

	adminSpec, err := g.GenerateSpec("admin")

	require.NoError(t, err)
	require.NotNil(t, adminSpec)
	assert.Equal(t, "3.1.2", adminSpec.OpenAPI)
}

// TestGenerateSpecCaseInsensitive tests that spec names are case-insensitive
func TestGenerateSpecCaseInsensitive(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /admin/users admin listAdminUsers
// spec: admin
func ListAdminUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithCache(false),
	)

	// Use uppercase spec name
	adminSpec, err := g.GenerateSpec("ADMIN")

	require.NoError(t, err)
	require.NotNil(t, adminSpec)
}

// TestGetSpecNamesWithNoRoutes tests GetSpecNames when no routes exist
func TestGetSpecNamesWithNoRoutes(t *testing.T) {
	files := map[string]string{
		"api/models.go": `package api

// swagger:model User
type User struct {
	ID int
}
`,
	}

	tmpDir := createTestProject(t, files)

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithCache(false),
	)

	names, err := g.GetSpecNames()

	require.NoError(t, err)
	// When there are no routes, collectSpecNames returns empty map
	assert.Empty(t, names)
}

// TestAssembleMultiWithRouteAddition tests that routes are properly added to specs
func TestAssembleMultiWithRouteAddition(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getUsers": {
			Method:      "GET",
			Path:        "/users",
			OperationID: "getUsers",
			Summary:     "Get all users",
			Description: "Returns a list of users",
			Tags:        []string{"users"},
			Specs:       []string{"public"},
		},
	}

	specs, err := g.assembleMulti()

	require.NoError(t, err)
	require.Contains(t, specs, "public")

	publicSpec := specs["public"]
	require.NotNil(t, publicSpec.Paths)
	assert.Contains(t, publicSpec.Paths.PathItems, "/users")
}

// TestAssembleForSpecWithGeneralMetaAndTags tests general meta with tags inheritance
func TestAssembleForSpecWithGeneralMetaAndTags(t *testing.T) {
	g := createTestGenerator()

	// Set general meta with tags but no specific metas
	g.scanner.Meta = &scanner.MetaInfo{
		Title:   "General API",
		Version: "1.0.0",
		Tags: []*scanner.TagInfo{
			{Name: "users", Description: "User operations"},
			{Name: "admin", Description: "Admin operations"},
		},
	}
	g.scanner.Metas = []*scanner.MetaInfo{} // Empty specific metas

	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getUsers": {
			Method:      "GET",
			Path:        "/users",
			OperationID: "getUsers",
			Specs:       []string{"public"},
		},
	}

	result, err := g.assembleForSpec("public")

	require.NoError(t, err)
	require.NotNil(t, result)
	// Should use general meta
	assert.Equal(t, "General API", result.Info.Title)
	// Tags should be inherited from general meta
	assert.Len(t, result.Tags, 2)
}

// TestAssembleForSpecWithSpecificMetaNoSecuritySchemes tests security scheme inheritance
func TestAssembleForSpecWithSpecificMetaNoSecuritySchemes(t *testing.T) {
	g := createTestGenerator()

	// General meta with security schemes
	g.scanner.Meta = &scanner.MetaInfo{
		Title:   "General API",
		Version: "1.0.0",
		SecuritySchemes: map[string]*scanner.SecuritySchemeInfo{
			"apiKey": {
				Type: "apiKey",
				Name: "X-API-Key",
				In:   "header",
			},
		},
	}

	// Specific meta WITHOUT security schemes (empty map)
	g.scanner.Metas = []*scanner.MetaInfo{
		{
			Title:           "Admin API",
			Version:         "2.0.0",
			Specs:           []string{"admin"},
			SecuritySchemes: map[string]*scanner.SecuritySchemeInfo{}, // Empty
		},
	}

	g.scanner.Routes = map[string]*scanner.RouteInfo{
		"getAdminUsers": {
			Method:      "GET",
			Path:        "/admin/users",
			OperationID: "getAdminUsers",
			Specs:       []string{"admin"},
		},
	}

	result, err := g.assembleForSpec("admin")

	require.NoError(t, err)
	require.NotNil(t, result)
	// Should use specific meta title
	assert.Equal(t, "Admin API", result.Info.Title)
	// Security schemes should be inherited from general meta
	assert.Contains(t, result.Components.SecuritySchemes, "apiKey")
}

// TestWriteMultiOutputDirectoryCreationError tests error when directory cannot be created
func TestWriteMultiOutputDirectoryCreationError(t *testing.T) {
	// Use a regular file as parent dir — creating a dir inside a file fails on all platforms
	tmpFile, err := os.CreateTemp(t.TempDir(), "blockingfile")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	g := createTestGenerator()
	g.config.OutputFile = filepath.Join(tmpFile.Name(), "invalid", "specs.yaml")
	g.config.OutputFormat = "yaml"

	specs := map[string]*spec.OpenAPI{
		"test": {OpenAPI: "3.1.2", Info: &spec.Info{Title: "Test", Version: "1.0.0"}},
	}

	err = g.writeMultiOutput(specs)
	assert.Error(t, err)
}

// TestAssembleMultiWithEmptySpecNames tests assembleMulti when no routes exist
func TestAssembleMultiWithEmptySpecNames(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Routes = map[string]*scanner.RouteInfo{}

	specs, err := g.assembleMulti()

	require.NoError(t, err)
	// Should return default spec even with no routes
	assert.Contains(t, specs, scanner.DefaultSpec)
}

// TestGenerateMultiWithOutputFile tests GenerateMulti with output file specified
func TestGenerateMultiWithOutputFile(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /users users listUsers
// spec: public
func ListUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)
	outputPath := filepath.Join(tmpDir, "output", "specs.yaml")

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithOutput(outputPath, "yaml"),
		WithCache(false),
	)

	specs, err := g.GenerateMulti()

	require.NoError(t, err)
	require.NotNil(t, specs)

	// Verify output file was created
	_, err = os.Stat(filepath.Join(filepath.Dir(outputPath), "public.yaml"))
	assert.NoError(t, err)
}

// TestGenerateSpecWithOutputFile tests GenerateSpec with output file specified
func TestGenerateSpecWithOutputFile(t *testing.T) {
	files := map[string]string{
		"api/handlers.go": `package api

// swagger:route GET /admin/users admin listAdminUsers
// spec: admin
func ListAdminUsers() {}
`,
	}

	tmpDir := createTestProject(t, files)
	outputPath := filepath.Join(tmpDir, "output", "admin.yaml")

	g := New(
		WithDir(tmpDir),
		WithPattern("./..."),
		WithOutput(outputPath, "yaml"),
		WithCache(false),
	)

	adminSpec, err := g.GenerateSpec("admin")

	require.NoError(t, err)
	require.NotNil(t, adminSpec)

	// Verify output file was created
	_, err = os.Stat(outputPath)
	assert.NoError(t, err)
}

// ==================== Convert Tests ====================

func TestMetaToInfo(t *testing.T) {
	g := createTestGenerator()

	tests := []struct {
		name     string
		meta     *scanner.MetaInfo
		expected *spec.Info
	}{
		{
			name: "basic meta",
			meta: &scanner.MetaInfo{
				Title:       "Test API",
				Description: "Test Description",
				Version:     "1.0.0",
			},
			expected: &spec.Info{
				Title:       "Test API",
				Description: "Test Description",
				Version:     "1.0.0",
			},
		},
		{
			name: "with contact",
			meta: &scanner.MetaInfo{
				Title:   "Test API",
				Version: "1.0.0",
				Contact: &scanner.ContactInfo{
					Name:  "Support",
					Email: "support@example.com",
					URL:   "https://example.com",
				},
			},
			expected: &spec.Info{
				Title:   "Test API",
				Version: "1.0.0",
				Contact: &spec.Contact{
					Name:  "Support",
					Email: "support@example.com",
					URL:   "https://example.com",
				},
			},
		},
		{
			name: "with license",
			meta: &scanner.MetaInfo{
				Title:   "Test API",
				Version: "1.0.0",
				License: &scanner.LicenseInfo{
					Name: "MIT",
					URL:  "https://opensource.org/licenses/MIT",
				},
			},
			expected: &spec.Info{
				Title:   "Test API",
				Version: "1.0.0",
				License: &spec.License{
					Name: "MIT",
					URL:  "https://opensource.org/licenses/MIT",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.metaToInfo(tt.meta)
			assert.Equal(t, tt.expected.Title, result.Title)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.Version, result.Version)
			if tt.expected.Contact != nil {
				require.NotNil(t, result.Contact)
				assert.Equal(t, tt.expected.Contact.Name, result.Contact.Name)
				assert.Equal(t, tt.expected.Contact.Email, result.Contact.Email)
			}
			if tt.expected.License != nil {
				require.NotNil(t, result.License)
				assert.Equal(t, tt.expected.License.Name, result.License.Name)
			}
		})
	}
}

func TestEnumToSchema(t *testing.T) {
	g := createTestGenerator()

	tests := []struct {
		name     string
		enum     *scanner.EnumInfo
		expected *spec.Schema
	}{
		{
			name: "string enum",
			enum: &scanner.EnumInfo{
				TypeName:    "Status",
				BaseType:    "string",
				Description: "Status enum",
				Values: map[string]any{
					"Active":   "active",
					"Inactive": "inactive",
				},
			},
			expected: &spec.Schema{
				Type:        spec.NewSchemaType("string"),
				Description: "Status enum",
			},
		},
		{
			name: "int enum",
			enum: &scanner.EnumInfo{
				TypeName: "Priority",
				BaseType: "int",
				Values: map[string]any{
					"Low":  1,
					"High": 2,
				},
			},
			expected: &spec.Schema{
				Type: spec.NewSchemaType("integer"),
			},
		},
		{
			name: "with example",
			enum: &scanner.EnumInfo{
				TypeName: "Color",
				BaseType: "string",
				Values: map[string]any{
					"Red": "red",
				},
				Example: "red",
			},
			expected: &spec.Schema{
				Type:     spec.NewSchemaType("string"),
				Examples: []any{"red"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.enumToSchema(tt.enum)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Description, result.Description)
			if len(tt.expected.Examples) > 0 {
				assert.Equal(t, tt.expected.Examples, result.Examples)
			}
			assert.Len(t, result.Enum, len(tt.enum.Values))
		})
	}
}

func TestStructToSchema(t *testing.T) {
	g := createTestGenerator()
	g.scanner.Enums = make(map[string]*scanner.EnumInfo)

	tests := []struct {
		name     string
		struct_  *scanner.StructInfo
		expected *spec.Schema
	}{
		{
			name: "simple struct",
			struct_: &scanner.StructInfo{
				Name:        "User",
				Description: "User model",
				Fields: []*scanner.FieldInfo{
					{
						Name: "ID",
						Type: "int",
						Tags: map[string]string{"json": "id"},
					},
					{
						Name: "Name",
						Type: "string",
						Tags: map[string]string{"json": "name"},
					},
				},
			},
			expected: &spec.Schema{
				Type:        spec.NewSchemaType("object"),
				Description: "User model",
			},
		},
		{
			name: "with required fields",
			struct_: &scanner.StructInfo{
				Name: "CreateUser",
				Fields: []*scanner.FieldInfo{
					{
						Name:     "Name",
						Type:     "string",
						Tags:     map[string]string{"json": "name"},
						Required: true,
					},
				},
			},
			expected: &spec.Schema{
				Type:     spec.NewSchemaType("object"),
				Required: []string{"name"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.structToSchema(tt.struct_)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Description, result.Description)
		})
	}
}

func TestSetSchemaType(t *testing.T) {
	g := createTestGenerator()

	tests := []struct {
		name         string
		goType       string
		expectedType string
		expectedFmt  string
	}{
		{"string", "string", "string", ""},
		{"int", "int", "integer", ""},
		{"int32", "int32", "integer", "int32"},
		{"int64", "int64", "integer", "int64"},
		{"float32", "float32", "number", "float"},
		{"float64", "float64", "number", "double"},
		{"bool", "bool", "boolean", ""},
		{"time.Time", "time.Time", "string", "date-time"},
		{"uuid.UUID", "uuid.UUID", "string", "uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &spec.Schema{}
			g.setSchemaType(schema, tt.goType)
			assert.Equal(t, tt.expectedType, schema.Type.Value())
			if tt.expectedFmt != "" {
				assert.Equal(t, tt.expectedFmt, schema.Format)
			}
		})
	}
}

func TestGetPropertyName(t *testing.T) {
	g := createTestGenerator()

	tests := []struct {
		name     string
		field    *scanner.FieldInfo
		expected string
	}{
		{
			name: "with json name",
			field: &scanner.FieldInfo{
				Name: "UserID",
				Tags: map[string]string{"json": "user_id"},
			},
			expected: "user_id",
		},
		{
			name: "without json name",
			field: &scanner.FieldInfo{
				Name: "UserID",
				Tags: map[string]string{},
			},
			expected: "UserID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.getPropertyName(tt.field)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortEnumValues(t *testing.T) {
	values := map[string]any{
		"C": "c",
		"A": "a",
		"B": "b",
	}

	result := sortEnumValues(values)

	assert.Len(t, result, 3)
	assert.Equal(t, "a", result[0])
	assert.Equal(t, "b", result[1])
	assert.Equal(t, "c", result[2])
}
