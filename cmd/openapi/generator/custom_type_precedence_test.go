package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A type the config registers renders as configured even when another package
// declares an enum (or a model) of the same short name. The enum lookup falls
// back to the unqualified name, so before this a `kernel.Coin` field — a plain
// string — was published as a $ref to the `Coin` enum of a legacy package the
// same scan happened to cover.
func TestConfiguredTypeBeatsAnEnumOfTheSameShortName(t *testing.T) {
	files := map[string]string{
		"model/coin.go": `package model

// swagger:enum Coin
// example: USD
type Coin string

const (
	CoinUSD Coin = "USD"
	CoinBTC Coin = "BTC"
)
`,
		"kernel/coin.go": `package kernel

// Coin is an open ticker, not the legacy enum.
type Coin string
`,
		"api/handlers.go": `package api

import "testproject/kernel"

// swagger:model WithdrawalRequest
type WithdrawalRequest struct {
	// example: BTC
	Coin kernel.Coin ` + "`json:\"coin\"`" + `
}

// swagger:parameters withdraw
type withdrawParams struct {
	// in:query
	Coin kernel.Coin ` + "`json:\"coin\" query:\"coin\"`" + `
	// in:body
	Body WithdrawalRequest
}

// swagger:route POST /withdrawals withdrawals withdraw
// Responses:
//   200: WithdrawalRequest
func Withdraw() {}
`,
	}
	tmpDir := createTestProject(t, files)

	type rendered struct{ bodyType, queryType, bodyRef, queryRef string }
	generate := func() rendered {
		g := New(WithDir(tmpDir), WithPattern("./..."), WithCache(false), WithEnumRefs(true))
		spec, err := g.Generate()
		require.NoError(t, err)

		model := spec.Components.Schemas["WithdrawalRequest"]
		require.NotNil(t, model)
		prop := model.Properties["coin"]
		require.NotNil(t, prop)

		op := spec.Paths.PathItems["/withdrawals"].Post
		require.NotNil(t, op)
		var query *struct{ typ, ref string }
		for _, p := range op.Parameters {
			if p.Name == "coin" {
				query = &struct{ typ, ref string }{p.Schema.Type.Value(), p.Schema.Ref}
			}
		}
		require.NotNil(t, query, "the query parameter is published")
		return rendered{prop.Type.Value(), query.typ, prop.Ref, query.ref}
	}

	t.Run("unregistered, the enum still wins (the documented fallback)", func(t *testing.T) {
		ClearCustomTypes()
		got := generate()
		assert.Equal(t, "#/components/schemas/Coin", got.bodyRef)
		assert.Equal(t, "#/components/schemas/Coin", got.queryRef)
	})

	t.Run("registered by qualified name, it renders as configured", func(t *testing.T) {
		ClearCustomTypes()
		t.Cleanup(ClearCustomTypes)
		RegisterTypeInfo("kernel.Coin", &TypeInfo{Type: "string", Example: "BTC"})

		got := generate()
		assert.Equal(t, "string", got.bodyType)
		assert.Equal(t, "string", got.queryType)
		assert.Empty(t, got.bodyRef)
		assert.Empty(t, got.queryRef)
	})

	t.Run("the legacy enum is untouched by the registration", func(t *testing.T) {
		ClearCustomTypes()
		t.Cleanup(ClearCustomTypes)
		RegisterTypeInfo("kernel.Coin", &TypeInfo{Type: "string"})

		g := New(WithDir(tmpDir), WithPattern("./..."), WithCache(false), WithEnumRefs(true))
		spec, err := g.Generate()
		require.NoError(t, err)
		// Unreferenced now, so it is only in the components when something
		// else points at it; the scanner still knows it as an enum.
		assert.NotNil(t, g.scanner.Enums["Coin"])
		_ = spec
	})
}
