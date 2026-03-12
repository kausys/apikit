package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveTypeAlias_Simple(t *testing.T) {
	s := New()
	s.TypeAliases["model.FeeType"] = "workspace.FeeType"

	result := s.ResolveTypeAlias("model.FeeType")
	assert.Equal(t, "workspace.FeeType", result)
}

func TestResolveTypeAlias_Chained(t *testing.T) {
	s := New()
	s.TypeAliases["A"] = "B"
	s.TypeAliases["B"] = "C"

	result := s.ResolveTypeAlias("A")
	assert.Equal(t, "C", result)
}

func TestResolveTypeAlias_CycleDetection(t *testing.T) {
	s := New()
	// Create a cycle: A -> B -> C -> A
	s.TypeAliases["A"] = "B"
	s.TypeAliases["B"] = "C"
	s.TypeAliases["C"] = "A"

	// Should not hang, should return one of the cycle members
	result := s.ResolveTypeAlias("A")
	assert.Contains(t, []string{"A", "B", "C"}, result)
}

func TestResolveTypeAlias_SelfCycle(t *testing.T) {
	s := New()
	s.TypeAliases["A"] = "A"

	result := s.ResolveTypeAlias("A")
	assert.Equal(t, "A", result)
}

func TestResolveTypeAlias_NoAlias(t *testing.T) {
	s := New()

	result := s.ResolveTypeAlias("SomeType")
	assert.Equal(t, "SomeType", result)
}

func TestResolveTypeAlias_ShortNameResolution(t *testing.T) {
	s := New()
	s.TypeAliases["FeeType"] = "workspace.FeeType"

	result := s.ResolveTypeAlias("model.FeeType")
	assert.Equal(t, "workspace.FeeType", result)
}

func TestResolveTypeAlias_ShortNameCycle(t *testing.T) {
	s := New()
	// Cycle via short name resolution: pkg.X -> Y -> pkg2.X (short "X" -> Y again)
	s.TypeAliases["X"] = "Y"
	s.TypeAliases["Y"] = "X"

	result := s.ResolveTypeAlias("pkg.X")
	assert.Contains(t, []string{"X", "Y"}, result)
}
