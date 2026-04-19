package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- GetCategories ----

func TestGetCategories_ReturnsNonEmpty(t *testing.T) {
	cats := GetCategories()
	assert.NotEmpty(t, cats, "GetCategories must return at least one category from the embedded YAML")
}

func TestGetCategories_ContainsPackages(t *testing.T) {
	cats := GetCategories()
	total := 0
	for _, cat := range cats {
		total += len(cat.Packages)
	}
	assert.Greater(t, total, 0, "expected packages inside categories")
}

func TestGetCategories_ReturnsCopy(t *testing.T) {
	// Mutating the returned slice must not affect a subsequent call.
	cats1 := GetCategories()
	require.NotEmpty(t, cats1)
	cats1[0].Name = "MUTATED"

	cats2 := GetCategories()
	require.NotEmpty(t, cats2)
	assert.NotEqual(t, "MUTATED", cats2[0].Name, "GetCategories must return a deep copy")
}

func TestGetCategories_EachCategoryHasName(t *testing.T) {
	for _, cat := range GetCategories() {
		assert.NotEmpty(t, cat.Name, "every category must have a non-empty name")
	}
}

// ---- GetAllPackageNames ----

func TestGetAllPackageNames_ReturnsNonEmpty(t *testing.T) {
	names := GetAllPackageNames()
	assert.NotEmpty(t, names)
}

func TestGetAllPackageNames_ContainsKnownPackages(t *testing.T) {
	names := GetAllPackageNames()
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	// curl and wget are in the Essential category of the embedded YAML.
	assert.True(t, nameSet["curl"], "curl should be in the package catalog")
	assert.True(t, nameSet["wget"], "wget should be in the package catalog")
}

func TestGetAllPackageNames_NoDuplicates(t *testing.T) {
	names := GetAllPackageNames()
	seen := make(map[string]int)
	for _, n := range names {
		seen[n]++
	}
	for name, count := range seen {
		assert.Equal(t, 1, count, "package %q appears %d times; expected exactly once", name, count)
	}
}

// ---- CatalogDescriptionMap ----

func TestCatalogDescriptionMap_ReturnsNonNil(t *testing.T) {
	m := CatalogDescriptionMap()
	assert.NotNil(t, m)
}

func TestCatalogDescriptionMap_ContainsDescriptions(t *testing.T) {
	m := CatalogDescriptionMap()
	// The embedded YAML has descriptions for many packages. At least one should exist.
	assert.Greater(t, len(m), 0, "CatalogDescriptionMap must return at least one entry")
}

func TestCatalogDescriptionMap_ValuesAreNonEmpty(t *testing.T) {
	m := CatalogDescriptionMap()
	for name, desc := range m {
		assert.NotEmpty(t, desc, "description for %q should not be empty", name)
	}
}

func TestCatalogDescriptionMap_ReturnsCopy(t *testing.T) {
	// Mutating the returned map must not affect a subsequent call.
	m1 := CatalogDescriptionMap()
	for k := range m1 {
		m1[k] = "MUTATED"
		break
	}
	m2 := CatalogDescriptionMap()
	for _, v := range m2 {
		assert.NotEqual(t, "MUTATED", v)
	}
}
