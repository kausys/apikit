package generator

import (
	"testing"

	"github.com/kausys/apikit/cmd/openapi/spec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A parameter's type resolves as a body field's does, and a slice is an
// array. Parameters used to be read as a primitive whatever they were: a
// typed identifier the config describes came out as a bare string (with an
// "unknown type" warning), a []string as one string carrying the slice's
// maxItems as a maxLength, and a slice of an enum as one enum value.
func TestParameterTypesResolveLikeFields(t *testing.T) {
	RegisterSchema("id.ItemID", &spec.Schema{Type: spec.NewSchemaType("string"), Pattern: "^itm_[0-9a-z]{26}$"})
	t.Cleanup(ClearExternalSchemas)

	dir := createTestProject(t, map[string]string{
		"id/id.go": `package id

type ItemID string
`,
		"api/api.go": `package api

import (
	"context"

	"testproject/id"
)

// swagger:enum
type Status string

const (
	StatusOpen   Status = "open"
	StatusClosed Status = "closed"
)

type GetRequest struct {
	ItemID id.ItemID ` + "`path:\"item_id\"`" + `
	Tags   []string  ` + "`query:\"tag\" validate:\"max=3,dive,min=2\"`" + `
	States []Status  ` + "`query:\"state\"`" + `
	Limit  int       ` + "`query:\"limit\" validate:\"max=50\"`" + `
}

// swagger:route GET /items/{item_id} items getItem
//
// Responses:
//   200: description: ok
func Get(ctx context.Context, req GetRequest) (string, error) { return "", nil }
`,
	})
	g := New(WithDir(dir), WithPattern("./..."), WithStrict(true))

	doc, err := g.Generate()
	require.NoError(t, err)

	params := map[string]*spec.Schema{}
	for _, p := range doc.Paths.PathItems["/items/{item_id}"].Get.Parameters {
		params[p.Name] = p.Schema
	}
	is := assert.New(t)

	is.Equal("#/components/schemas/id.ItemID", params["item_id"].Ref, "a schema the config names is referenced")
	is.Contains(doc.Components.Schemas, "id.ItemID", "and is in the document")

	require.NotNil(t, params["tag"].Items)
	is.Equal("array", params["tag"].Type.Value())
	is.Equal(uint64(3), *params["tag"].MaxItems, "the slice's rule bounds the count")
	is.Equal(uint64(2), params["tag"].Items.MinLength, "the rule after dive bounds each value")

	require.NotNil(t, params["state"].Items)
	is.Equal("array", params["state"].Type.Value())
	is.Equal([]any{"closed", "open"}, params["state"].Items.Enum)

	is.InDelta(50, *params["limit"].Maximum, 0, "a scalar keeps its rules")
}
