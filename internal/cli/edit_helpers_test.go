package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// splitBefore (edit.go) — pure string utility
// ---------------------------------------------------------------------------

func TestSplitBefore_WithSeparator(t *testing.T) {
	result := splitBefore("my-config — My Config Name", " — ")
	assert.Equal(t, "my-config", result)
}

func TestSplitBefore_NoSeparator(t *testing.T) {
	result := splitBefore("my-config", " — ")
	assert.Equal(t, "my-config", result)
}

func TestSplitBefore_EmptyString(t *testing.T) {
	result := splitBefore("", " — ")
	assert.Equal(t, "", result)
}

func TestSplitBefore_SeparatorAtStart(t *testing.T) {
	result := splitBefore(" — rest", " — ")
	assert.Equal(t, "", result)
}

func TestSplitBefore_SeparatorAtEnd(t *testing.T) {
	// " — " is at the very end — nothing after separator, but before is the slug.
	result := splitBefore("slug — ", " — ")
	assert.Equal(t, "slug", result)
}

func TestSplitBefore_MultipleSeparators(t *testing.T) {
	// Should split at the FIRST occurrence.
	result := splitBefore("a — b — c", " — ")
	assert.Equal(t, "a", result)
}

func TestSplitBefore_SingleCharSeparator(t *testing.T) {
	result := splitBefore("hello:world", ":")
	assert.Equal(t, "hello", result)
}

func TestSplitBefore_StringShorterThanSeparator(t *testing.T) {
	result := splitBefore("ab", "abcd")
	assert.Equal(t, "ab", result)
}

func TestSplitBefore_IdenticalToSeparator(t *testing.T) {
	result := splitBefore(" — ", " — ")
	assert.Equal(t, "", result)
}
