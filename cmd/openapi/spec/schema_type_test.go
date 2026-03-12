package spec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSchemaType_NewSchemaType(t *testing.T) {
	st := NewSchemaType("string")
	assert.Equal(t, "string", st.Value())
	assert.Equal(t, []string{"string"}, st.Values())
	assert.False(t, st.IsEmpty())
	assert.False(t, st.IsZero())
}

func TestSchemaType_NewSchemaTypes(t *testing.T) {
	st := NewSchemaTypes("string", "null")
	assert.Equal(t, "string", st.Value())
	assert.Equal(t, []string{"string", "null"}, st.Values())
	assert.False(t, st.IsEmpty())
}

func TestSchemaType_Empty(t *testing.T) {
	st := SchemaType{}
	assert.Empty(t, st.Value())
	assert.True(t, st.IsEmpty())
	assert.True(t, st.IsZero())
}

func TestSchemaType_NewSchemaType_Empty(t *testing.T) {
	st := NewSchemaType("")
	assert.True(t, st.IsEmpty())
	assert.True(t, st.IsZero())
}

func TestSchemaType_Contains(t *testing.T) {
	st := NewSchemaTypes("string", "null")
	assert.True(t, st.Contains("string"))
	assert.True(t, st.Contains("null"))
	assert.False(t, st.Contains("integer"))
}

func TestSchemaType_WithNull(t *testing.T) {
	st := NewSchemaType("string")
	nullable := st.WithNull()

	assert.Equal(t, []string{"string", "null"}, nullable.Values())
	assert.True(t, nullable.Contains("null"))
	assert.True(t, nullable.Contains("string"))

	// Original should not be modified
	assert.Equal(t, []string{"string"}, st.Values())
	assert.False(t, st.Contains("null"))
}

func TestSchemaType_WithNull_AlreadyNullable(t *testing.T) {
	st := NewSchemaTypes("string", "null")
	nullable := st.WithNull()

	// Should not add another "null"
	assert.Equal(t, []string{"string", "null"}, nullable.Values())
}

func TestSchemaType_Value_NullOnly(t *testing.T) {
	st := NewSchemaType("null")
	assert.Equal(t, "null", st.Value())
}

func TestSchemaType_Value_PreferNonNull(t *testing.T) {
	st := NewSchemaTypes("null", "integer")
	assert.Equal(t, "integer", st.Value())
}

func TestSchemaType_MarshalJSON_Single(t *testing.T) {
	st := NewSchemaType("string")
	data, err := json.Marshal(st)
	require.NoError(t, err)
	assert.Equal(t, `"string"`, string(data))
}

func TestSchemaType_MarshalJSON_Multiple(t *testing.T) {
	st := NewSchemaTypes("string", "null")
	data, err := json.Marshal(st)
	require.NoError(t, err)
	assert.Equal(t, `["string","null"]`, string(data))
}

func TestSchemaType_MarshalJSON_Empty(t *testing.T) {
	st := SchemaType{}
	data, err := json.Marshal(st)
	require.NoError(t, err)
	assert.Equal(t, "null", string(data))
}

func TestSchemaType_UnmarshalJSON_Single(t *testing.T) {
	var st SchemaType
	err := json.Unmarshal([]byte(`"string"`), &st)
	require.NoError(t, err)
	assert.Equal(t, []string{"string"}, st.Values())
}

func TestSchemaType_UnmarshalJSON_Multiple(t *testing.T) {
	var st SchemaType
	err := json.Unmarshal([]byte(`["string","null"]`), &st)
	require.NoError(t, err)
	assert.Equal(t, []string{"string", "null"}, st.Values())
}

func TestSchemaType_UnmarshalJSON_Null(t *testing.T) {
	var st SchemaType
	err := json.Unmarshal([]byte(`null`), &st)
	require.NoError(t, err)
	assert.True(t, st.IsEmpty())
}

func TestSchemaType_MarshalYAML_Single(t *testing.T) {
	st := NewSchemaType("integer")
	data, err := yaml.Marshal(st)
	require.NoError(t, err)
	assert.Equal(t, "integer\n", string(data))
}

func TestSchemaType_MarshalYAML_Multiple(t *testing.T) {
	st := NewSchemaTypes("string", "null")
	data, err := yaml.Marshal(st)
	require.NoError(t, err)
	assert.Equal(t, "- string\n- \"null\"\n", string(data))
}

func TestSchemaType_UnmarshalYAML_Single(t *testing.T) {
	var st SchemaType
	err := yaml.Unmarshal([]byte("string"), &st)
	require.NoError(t, err)
	assert.Equal(t, []string{"string"}, st.Values())
}

func TestSchemaType_UnmarshalYAML_Multiple(t *testing.T) {
	var st SchemaType
	err := yaml.Unmarshal([]byte("- string\n- \"null\"\n"), &st)
	require.NoError(t, err)
	assert.Equal(t, []string{"string", "null"}, st.Values())
}

func TestSchemaType_JSON_Roundtrip(t *testing.T) {
	tests := []struct {
		name string
		st   SchemaType
	}{
		{"single", NewSchemaType("string")},
		{"multiple", NewSchemaTypes("string", "null")},
		{"empty", SchemaType{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.st)
			require.NoError(t, err)

			var result SchemaType
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			assert.Equal(t, tt.st.Values(), result.Values())
		})
	}
}

func TestSchemaType_InSchema_JSON(t *testing.T) {
	schema := Schema{
		Type:        NewSchemaTypes("string", "null"),
		Description: "A nullable string",
	}

	data, err := json.Marshal(schema)
	require.NoError(t, err)

	var result Schema
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, []string{"string", "null"}, result.Type.Values())
	assert.Equal(t, "A nullable string", result.Description)
}

func TestSchemaType_InSchema_OmitEmpty(t *testing.T) {
	schema := Schema{
		Description: "No type set",
	}

	data, err := json.Marshal(schema)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"type"`)
}
