package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLooksLikeFilePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"relative dot", "./file.json", true},
		{"relative dotdot", "../file.json", true},
		{"absolute", "/tmp/file.json", true},
		{"json suffix", "backup.json", true},
		{"user slug", "alice/dev-setup", false},
		{"plain word", "developer", false},
		{"empty", "", false},
		{"no slash json", "backup.txt", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, looksLikeFilePath(tt.in))
		})
	}
}

func TestLooksLikeUserSlug(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"standard", "alice/dev-setup", true},
		{"underscores", "alice_b/my_setup", true},
		{"digits", "user123/config-2", true},
		{"leading digits in slug", "alice/2-config", true},
		{"leading digit user", "1alice/foo", true},
		{"leading dash", "-alice/foo", false},
		{"three parts", "a/b/c", false},
		{"trailing slash", "alice/", false},
		{"just slash", "/", false},
		{"no slash", "alice", false},
		{"file-like", "./alice/foo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, looksLikeUserSlug(tt.in))
		})
	}
}

func TestResolvePositionalArg_Preset(t *testing.T) {
	src, err := resolvePositionalArg("developer")
	assert.NoError(t, err)
	// "developer" is a built-in preset name.
	assert.Equal(t, sourcePreset, src.kind)
}

func TestResolvePositionalArg_File(t *testing.T) {
	src, err := resolvePositionalArg("./backup.json")
	assert.NoError(t, err)
	assert.Equal(t, sourceFile, src.kind)
	assert.Equal(t, "./backup.json", src.path)
}

func TestResolvePositionalArg_UserSlug(t *testing.T) {
	src, err := resolvePositionalArg("alice/dev-setup")
	assert.NoError(t, err)
	assert.Equal(t, sourceCloud, src.kind)
	assert.Equal(t, "alice/dev-setup", src.userSlug)
}

func TestResolvePositionalArg_Alias(t *testing.T) {
	// A plain word that isn't a preset is treated as a cloud alias —
	// FetchRemoteConfig will attempt alias resolution at run time.
	src, err := resolvePositionalArg("my-custom-alias")
	assert.NoError(t, err)
	assert.Equal(t, sourceCloud, src.kind)
	assert.Equal(t, "my-custom-alias", src.userSlug)
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name     string
		d        time.Duration
		contains string
	}{
		{"under an hour", 30 * time.Minute, "just now"},
		{"hours", 2 * time.Hour, "hours"},
		{"one hour singular", 1 * time.Hour, "1 hour ago"},
		{"days", 3 * 24 * time.Hour, "days"},
		{"one day singular", 24 * time.Hour, "1 day ago"},
		{"months", 45 * 24 * time.Hour, "month"},
		{"years", 400 * 24 * time.Hour, "year"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, relativeTime(tt.d), tt.contains)
		})
	}
}
