package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
}

func fieldNames(s *Struct) []string {
	names := make([]string, 0, len(s.Fields))
	for _, f := range s.Fields {
		names = append(names, f.Name)
	}
	return names
}

// A payload declared in another file of the handler's package resolves. It
// used to be refused: the lookup only saw the handler's own file.
func TestPayloadInAnotherFileOfThePackage(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod": "module example.com/app\n\ngo 1.27\n",
		"api/types.go": `package api

type GetRequest struct {
	ID string ` + "`path:\"id\"`" + `
}
`,
		"api/handler.go": `package api

import "context"

// apikit:handler
func Get(ctx context.Context, req GetRequest) (string, error) { return "", nil }
`,
	})

	result, err := New().ParseFile(filepath.Join(root, "api", "handler.go"))

	require.NoError(t, err)
	require.Empty(t, result.Errors)
	require.Len(t, result.Handlers, 1)
	assert.Equal(t, []string{"ID"}, fieldNames(result.Handlers[0].Struct))
	assert.Empty(t, result.Handlers[0].ParamImport)
}

// A payload from another package of the module resolves, and the handler
// records where it came from so the wrapper can import it.
func TestPayloadInAnotherPackageOfTheModule(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod": "module example.com/app\n\ngo 1.27\n",
		"dto/dto.go": `package dto

type CreateRequest struct {
	Body CreateBody ` + "`json:\"body\"`" + `
}

type CreateBody struct {
	Name string ` + "`json:\"name\"`" + `
}
`,
		"api/handler.go": `package api

import (
	"context"

	"example.com/app/dto"
)

// apikit:handler
func Create(ctx context.Context, req dto.CreateRequest) (string, error) { return "", nil }
`,
	})

	result, err := New().ParseFile(filepath.Join(root, "api", "handler.go"))

	require.NoError(t, err)
	require.Empty(t, result.Errors)
	require.Len(t, result.Handlers, 1)
	h := result.Handlers[0]
	assert.Equal(t, "example.com/app/dto", h.ParamImport)
	assert.Equal(t, "dto", h.ParamImportAlias)
	require.NotNil(t, h.Struct.Fields[0].NestedStruct, "a field type resolves in the payload's own package")
	assert.Equal(t, []string{"Name"}, fieldNames(h.Struct.Fields[0].NestedStruct))
}

// Two packages declaring a struct with the same name each resolve their own.
// Structs used to be cached by bare name across the run, so the second
// package's Payload resolved to the first package's.
func TestSameStructNameInTwoPackages(t *testing.T) {
	root := t.TempDir()
	handler := func(pkg string) string {
		return `package ` + pkg + `

import "context"

type Request struct {
	Body Payload ` + "`json:\"body\"`" + `
}

// apikit:handler
func Do(ctx context.Context, req Request) (string, error) { return "", nil }
`
	}
	writeFiles(t, root, map[string]string{
		"go.mod":       "module example.com/app\n\ngo 1.27\n",
		"a/handler.go": handler("a"),
		"a/body.go":    "package a\n\ntype Payload struct {\n\tFromA string\n}\n",
		"b/handler.go": handler("b"),
		"b/body.go":    "package b\n\ntype Payload struct {\n\tFromB string\n}\n",
	})
	p := New()

	ra, err := p.ParseFile(filepath.Join(root, "a", "handler.go"))
	require.NoError(t, err)
	rb, err := p.ParseFile(filepath.Join(root, "b", "handler.go"))
	require.NoError(t, err)

	require.Len(t, ra.Handlers, 1)
	require.Len(t, rb.Handlers, 1)
	assert.Equal(t, []string{"FromA"}, fieldNames(ra.Handlers[0].Struct.Fields[0].NestedStruct))
	assert.Equal(t, []string{"FromB"}, fieldNames(rb.Handlers[0].Struct.Fields[0].NestedStruct))
}
