package spec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ==================== Paths Tests ====================

func TestPathsMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		paths    *Paths
		expected string
	}{
		{
			name:     "empty paths",
			paths:    &Paths{},
			expected: "{}",
		},
		{
			name: "with path items",
			paths: &Paths{
				PathItems: map[string]*PathItem{
					"/users": {Summary: "Users endpoint"},
				},
			},
			expected: `{"/users":{"summary":"Users endpoint"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.paths)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestPathsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // number of path items
	}{
		{
			name:     "empty object",
			input:    "{}",
			expected: 0,
		},
		{
			name:     "with path items",
			input:    `{"/users":{"summary":"Users"}}`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paths Paths
			err := json.Unmarshal([]byte(tt.input), &paths)
			require.NoError(t, err)
			assert.Len(t, paths.PathItems, tt.expected)
		})
	}
}

func TestPathsMarshalYAML(t *testing.T) {
	paths := &Paths{
		PathItems: map[string]*PathItem{
			"/users": {Summary: "Users endpoint"},
		},
	}

	data, err := yaml.Marshal(paths)
	require.NoError(t, err)
	assert.Contains(t, string(data), "/users")
}

func TestPathsUnmarshalYAML(t *testing.T) {
	input := `/users:
  summary: Users endpoint
`
	var paths Paths
	err := yaml.Unmarshal([]byte(input), &paths)
	require.NoError(t, err)
	assert.Len(t, paths.PathItems, 1)
	assert.Contains(t, paths.PathItems, "/users")
}

// ==================== Responses Tests ====================

func TestResponsesMarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		responses *Responses
		expected  string
	}{
		{
			name:      "empty responses",
			responses: &Responses{},
			expected:  "{}",
		},
		{
			name: "with status codes",
			responses: &Responses{
				StatusCodes: map[string]*Response{
					"200": {Description: "OK"},
				},
			},
			expected: `{"200":{"description":"OK"}}`,
		},
		{
			name: "with default",
			responses: &Responses{
				Default: &Response{Description: "Default response"},
			},
			expected: `{"default":{"description":"Default response"}}`,
		},
		{
			name: "with both",
			responses: &Responses{
				Default: &Response{Description: "Default"},
				StatusCodes: map[string]*Response{
					"200": {Description: "OK"},
				},
			},
			expected: `{"200":{"description":"OK"},"default":{"description":"Default"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.responses)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestResponsesUnmarshalJSON(t *testing.T) {
	input := `{"200":{"description":"OK"},"default":{"description":"Default"}}`

	var responses Responses
	err := json.Unmarshal([]byte(input), &responses)

	require.NoError(t, err)
	assert.Len(t, responses.StatusCodes, 1)
	assert.NotNil(t, responses.Default)
	assert.Equal(t, "Default", responses.Default.Description)
}

func TestResponsesMarshalYAML(t *testing.T) {
	responses := &Responses{
		Default: &Response{Description: "Default"},
		StatusCodes: map[string]*Response{
			"200": {Description: "OK"},
		},
	}

	data, err := yaml.Marshal(responses)
	require.NoError(t, err)
	assert.Contains(t, string(data), "200")
	assert.Contains(t, string(data), "default")
}

func TestResponsesUnmarshalYAML(t *testing.T) {
	input := `"200":
  description: OK
default:
  description: Default
`
	var responses Responses
	err := yaml.Unmarshal([]byte(input), &responses)
	require.NoError(t, err)
	assert.Len(t, responses.StatusCodes, 1)
	assert.NotNil(t, responses.Default)
}

// ==================== SecurityRequirement Tests ====================

func TestSecurityRequirementMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		security *SecurityRequirement
		expected string
	}{
		{
			name:     "empty security",
			security: &SecurityRequirement{},
			expected: "{}",
		},
		{
			name: "with requirements",
			security: &SecurityRequirement{
				Requirements: map[string][]string{
					"bearerAuth": {},
				},
			},
			expected: `{"bearerAuth":[]}`,
		},
		{
			name: "with scopes",
			security: &SecurityRequirement{
				Requirements: map[string][]string{
					"oauth2": {"read", "write"},
				},
			},
			expected: `{"oauth2":["read","write"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.security)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestSecurityRequirementUnmarshalJSON(t *testing.T) {
	input := `{"oauth2":["read","write"]}`

	var security SecurityRequirement
	err := json.Unmarshal([]byte(input), &security)

	require.NoError(t, err)
	assert.Contains(t, security.Requirements, "oauth2")
	assert.Equal(t, []string{"read", "write"}, security.Requirements["oauth2"])
}

func TestSecurityRequirementMarshalYAML(t *testing.T) {
	security := &SecurityRequirement{
		Requirements: map[string][]string{
			"bearerAuth": {},
		},
	}

	data, err := yaml.Marshal(security)
	require.NoError(t, err)
	assert.Contains(t, string(data), "bearerAuth")
}

func TestSecurityRequirementUnmarshalYAML(t *testing.T) {
	input := `bearerAuth: []
`
	var security SecurityRequirement
	err := yaml.Unmarshal([]byte(input), &security)
	require.NoError(t, err)
	assert.Contains(t, security.Requirements, "bearerAuth")
}

// ==================== Edge Cases ====================

func TestPathsUnmarshalJSONEmptyObject(t *testing.T) {
	var paths Paths
	err := json.Unmarshal([]byte("{}"), &paths)
	require.NoError(t, err)
	assert.Empty(t, paths.PathItems)
}

func TestResponsesUnmarshalJSONEmpty(t *testing.T) {
	var responses Responses
	err := json.Unmarshal([]byte("{}"), &responses)
	require.NoError(t, err)
	assert.Empty(t, responses.StatusCodes)
}

func TestSecurityRequirementUnmarshalJSONEmpty(t *testing.T) {
	var security SecurityRequirement
	err := json.Unmarshal([]byte("{}"), &security)
	require.NoError(t, err)
	assert.Empty(t, security.Requirements)
}

// ==================== Nil Receiver Tests ====================

func TestPathsNilReceiver(t *testing.T) {
	var paths *Paths
	data, err := paths.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data))
}

func TestResponsesNilReceiver(t *testing.T) {
	var responses *Responses
	data, err := responses.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data))
}

func TestSecurityRequirementNilReceiver(t *testing.T) {
	var security *SecurityRequirement
	data, err := security.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data))
}

// ==================== Callback Tests ====================

func TestCallbackMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		callback *Callback
		expected string
	}{
		{
			name:     "empty callback",
			callback: &Callback{},
			expected: "{}",
		},
		{
			name: "with path items",
			callback: &Callback{
				PathItems: map[string]*PathItem{
					"{$request.body#/callbackUrl}": {Summary: "Callback"},
				},
			},
			expected: `{"{$request.body#/callbackUrl}":{"summary":"Callback"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.callback)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestCallbackUnmarshalJSON(t *testing.T) {
	input := `{"{$request.body#/callbackUrl}":{"summary":"Callback"}}`

	var callback Callback
	err := json.Unmarshal([]byte(input), &callback)

	require.NoError(t, err)
	assert.Len(t, callback.PathItems, 1)
}

func TestCallbackMarshalYAML(t *testing.T) {
	callback := &Callback{
		PathItems: map[string]*PathItem{
			"{$request.body#/callbackUrl}": {Summary: "Callback"},
		},
	}

	data, err := yaml.Marshal(callback)
	require.NoError(t, err)
	assert.Contains(t, string(data), "callbackUrl")
}

func TestCallbackUnmarshalYAML(t *testing.T) {
	input := `"{$request.body#/callbackUrl}":
  summary: Callback
`
	var callback Callback
	err := yaml.Unmarshal([]byte(input), &callback)
	require.NoError(t, err)
	assert.Len(t, callback.PathItems, 1)
}

func TestCallbackNilReceiver(t *testing.T) {
	var callback *Callback
	data, err := callback.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data))
}

func TestCallbackUnmarshalJSONEmpty(t *testing.T) {
	var callback Callback
	err := json.Unmarshal([]byte("{}"), &callback)
	require.NoError(t, err)
	assert.Empty(t, callback.PathItems)
}
