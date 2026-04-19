package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- SetClientVersion ----

func TestSetClientVersion_CanBeCalledWithoutPanic(t *testing.T) {
	// Capture the original value and restore it after the test.
	original := clientVersion
	t.Cleanup(func() { clientVersion = original })

	assert.NotPanics(t, func() { SetClientVersion("1.2.3") })
	assert.Equal(t, "1.2.3", clientVersion)
}

func TestSetClientVersion_UpdatesPackageVar(t *testing.T) {
	original := clientVersion
	t.Cleanup(func() { clientVersion = original })

	SetClientVersion("0.99.0")
	assert.Equal(t, "0.99.0", clientVersion)

	SetClientVersion("dev")
	assert.Equal(t, "dev", clientVersion)
}

// ---- GetScreenRecordingPackages ----

func TestGetScreenRecordingPackages_ReturnsNonEmpty(t *testing.T) {
	pkgs := GetScreenRecordingPackages()
	assert.NotNil(t, pkgs)
	assert.Greater(t, len(pkgs), 0, "expected at least one screen-recording package")
}

func TestGetScreenRecordingPackages_ContainsStrings(t *testing.T) {
	pkgs := GetScreenRecordingPackages()
	for _, p := range pkgs {
		assert.NotEmpty(t, p, "package name should not be empty")
	}
}

// ---- LoadRemoteConfigFromFile ----

func TestLoadRemoteConfigFromFile_ValidRemoteConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := []byte(`{
		"username": "alice",
		"slug": "setup",
		"packages": ["git", "curl"],
		"casks": ["firefox"],
		"taps": ["homebrew/core"],
		"npm": ["typescript"]
	}`)
	require.NoError(t, os.WriteFile(path, data, 0600))

	rc, err := LoadRemoteConfigFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, rc)
	assert.Equal(t, "alice", rc.Username)
	assert.Equal(t, "setup", rc.Slug)
	assert.Len(t, rc.Packages, 2)
	assert.Equal(t, "git", rc.Packages[0].Name)
	assert.Len(t, rc.Casks, 1)
	assert.Len(t, rc.Taps, 1)
	assert.Len(t, rc.Npm, 1)
}

func TestLoadRemoteConfigFromFile_ValidRemoteConfigWithDotfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := []byte(`{
		"packages": ["git"],
		"dotfiles_repo": "https://github.com/alice/dotfiles"
	}`)
	require.NoError(t, os.WriteFile(path, data, 0600))

	rc, err := LoadRemoteConfigFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, rc)
	assert.Equal(t, "https://github.com/alice/dotfiles", rc.DotfilesRepo)
}

func TestLoadRemoteConfigFromFile_NonExistentPath(t *testing.T) {
	_, err := LoadRemoteConfigFromFile("/nonexistent/path/config.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config file")
}

func TestLoadRemoteConfigFromFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not json at all"), 0600))

	_, err := LoadRemoteConfigFromFile(path)
	require.Error(t, err)
}

func TestLoadRemoteConfigFromFile_InvalidPackageName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.json")

	data := []byte(`{
		"packages": ["bad name with spaces"]
	}`)
	require.NoError(t, os.WriteFile(path, data, 0600))

	_, err := LoadRemoteConfigFromFile(path)
	require.Error(t, err)
}

func TestLoadRemoteConfigFromFile_SnapshotFormat(t *testing.T) {
	// Snapshot files are auto-detected via the "captured_at" field.
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")

	snapshot := map[string]interface{}{
		"captured_at": "2024-01-01T00:00:00Z",
		"packages": map[string]interface{}{
			"formulae": []string{"git", "curl"},
			"casks":    []string{"firefox"},
			"taps":     []string{"homebrew/core"},
			"npm":      []string{"typescript"},
		},
		"shell": map[string]interface{}{
			"oh_my_zsh": true,
			"theme":     "robbyrussell",
			"plugins":   []string{"git"},
		},
		"macos_prefs": []interface{}{},
	}
	data, err := json.Marshal(snapshot)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))

	rc, err := LoadRemoteConfigFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, rc)
	assert.Len(t, rc.Packages, 2)
	assert.Len(t, rc.Casks, 1)
	require.NotNil(t, rc.Shell)
	assert.True(t, rc.Shell.OhMyZsh)
	assert.Equal(t, "robbyrussell", rc.Shell.Theme)
}

func TestLoadRemoteConfigFromFile_SnapshotFormat_NoShell(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")

	snapshot := map[string]interface{}{
		"captured_at": "2024-01-01T00:00:00Z",
		"packages": map[string]interface{}{
			"formulae": []string{"git"},
			"casks":    []string{},
			"taps":     []string{},
			"npm":      []string{},
		},
		"shell": map[string]interface{}{
			"oh_my_zsh": false,
			"theme":     "",
			"plugins":   []string{},
		},
		"macos_prefs": []interface{}{},
	}
	data, err := json.Marshal(snapshot)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))

	rc, err := LoadRemoteConfigFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, rc)
	// oh_my_zsh = false → Shell field should be nil.
	assert.Nil(t, rc.Shell)
}

func TestLoadRemoteConfigFromFile_ObjectArrayFormat(t *testing.T) {
	// The server sometimes returns a typed-object array for packages.
	dir := t.TempDir()
	path := filepath.Join(dir, "typed.json")

	data := []byte(`{
		"packages": [
			{"name": "git", "type": "formula"},
			{"name": "firefox", "type": "cask"},
			{"name": "homebrew/cask-fonts", "type": "tap"},
			{"name": "typescript", "type": "npm"}
		]
	}`)
	require.NoError(t, os.WriteFile(path, data, 0600))

	rc, err := LoadRemoteConfigFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, rc)
	assert.Len(t, rc.Packages, 1)
	assert.Equal(t, "git", rc.Packages[0].Name)
	assert.Len(t, rc.Casks, 1)
	assert.Equal(t, "firefox", rc.Casks[0].Name)
	assert.Len(t, rc.Taps, 1)
	assert.Len(t, rc.Npm, 1)
}

// ---- getAPIBase ----

func TestGetAPIBase_DefaultsToProductionURL(t *testing.T) {
	t.Setenv("OPENBOOT_API_URL", "")
	base := getAPIBase()
	assert.Equal(t, "https://openboot.dev", base)
}

func TestGetAPIBase_AcceptsHTTPS(t *testing.T) {
	t.Setenv("OPENBOOT_API_URL", "https://staging.openboot.dev")
	base := getAPIBase()
	assert.Equal(t, "https://staging.openboot.dev", base)
}

func TestGetAPIBase_AcceptsLocalhost(t *testing.T) {
	t.Setenv("OPENBOOT_API_URL", "http://localhost:3000")
	base := getAPIBase()
	assert.Equal(t, "http://localhost:3000", base)
}

func TestGetAPIBase_RejectsInsecureNonLocalhost(t *testing.T) {
	t.Setenv("OPENBOOT_API_URL", "http://evil.com/steal")
	base := getAPIBase()
	// Insecure non-localhost URL must be rejected; fallback to production.
	assert.Equal(t, "https://openboot.dev", base)
}
