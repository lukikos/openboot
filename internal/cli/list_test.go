package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunList_NotAuthenticated(t *testing.T) {
	setupTestAuth(t, false)
	t.Setenv("OPENBOOT_API_URL", "http://localhost:9999")

	err := runList()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRunList_ShowsConfigs(t *testing.T) {
	setupTestAuth(t, true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/configs", r.URL.Path)
		assert.Equal(t, "Bearer obt_test_token_123", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(map[string]any{
			"configs": []map[string]any{
				{"slug": "my-setup", "name": "My Mac Setup", "visibility": "public"},
				{"slug": "work-mac", "name": "Work Machine", "visibility": "private"},
			},
		})
	}))
	defer server.Close()

	t.Setenv("OPENBOOT_API_URL", server.URL)

	err := runList()
	assert.NoError(t, err)
}

func TestRunList_Empty(t *testing.T) {
	setupTestAuth(t, true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"configs": []any{}})
	}))
	defer server.Close()

	t.Setenv("OPENBOOT_API_URL", server.URL)

	err := runList()
	assert.NoError(t, err)
}

func TestRunList_ServerError(t *testing.T) {
	setupTestAuth(t, true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	t.Setenv("OPENBOOT_API_URL", server.URL)

	// fetchUserConfigs treats non-200 as nil (non-fatal), so runList shows empty list
	err := runList()
	assert.NoError(t, err)
}

func TestRunList_LinkedConfigMarked(t *testing.T) {
	tmpDir := setupTestAuth(t, true)
	writeSyncSource(t, tmpDir, "my-setup")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"configs": []map[string]any{
				{"slug": "my-setup", "name": "My Mac Setup", "visibility": "unlisted"},
				{"slug": "work-mac", "name": "Work Machine", "visibility": "unlisted"},
			},
		})
	}))
	defer server.Close()

	t.Setenv("OPENBOOT_API_URL", server.URL)

	err := runList()
	assert.NoError(t, err)
}

func TestSlugInList(t *testing.T) {
	configs := []remoteConfigSummary{
		{Slug: "my-setup"},
		{Slug: "work-mac"},
	}

	assert.True(t, slugInList(configs, "my-setup"))
	assert.True(t, slugInList(configs, "work-mac"))
	assert.False(t, slugInList(configs, "nonexistent"))
}

func TestSlugInList_CaseInsensitive(t *testing.T) {
	configs := []remoteConfigSummary{{Slug: "My-Setup"}}

	assert.True(t, slugInList(configs, "my-setup"))
	assert.True(t, slugInList(configs, "MY-SETUP"))
}

func TestListCmd_CommandStructure(t *testing.T) {
	assert.Equal(t, "list", listCmd.Use)
	assert.NotEmpty(t, listCmd.Short)
	assert.NotEmpty(t, listCmd.Long)
	assert.NotNil(t, listCmd.RunE)
}
