package search

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openbootdotdev/openboot/internal/config"
)

// newTestServer creates an httptest.Server that serves a fixed response for
// any request path. It returns both the server and a cleanup function.
func newTestServer(t *testing.T, statusCode int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// setAPIURL points the package at the given test server and restores the
// original environment value when the test ends.
func setAPIURL(t *testing.T, serverURL string) {
	t.Helper()
	t.Setenv("OPENBOOT_API_URL", serverURL)
}

// marshalResponse is a helper to build a JSON searchResponse body.
func marshalResponse(t *testing.T, results []searchResult) string {
	t.Helper()
	b, err := json.Marshal(searchResponse{Results: results})
	require.NoError(t, err)
	return string(b)
}

// --- queryAPI tests ---

func TestQueryAPI_200_ValidJSON(t *testing.T) {
	results := []searchResult{
		{Name: "git", Desc: "Distributed version control", Type: "formula"},
		{Name: "iterm2", Desc: "Terminal emulator", Type: "cask"},
		{Name: "typescript", Desc: "TypeScript compiler", Type: "npm"},
	}
	body := marshalResponse(t, results)
	srv := newTestServer(t, http.StatusOK, body)
	setAPIURL(t, srv.URL)

	pkgs, err := queryAPI("homebrew", "git")

	require.NoError(t, err)
	require.Len(t, pkgs, 3)

	assert.Equal(t, "git", pkgs[0].Name)
	assert.Equal(t, "Distributed version control", pkgs[0].Description)
	assert.False(t, pkgs[0].IsCask, "formula should not be a cask")
	assert.False(t, pkgs[0].IsNpm, "formula should not be npm")

	assert.Equal(t, "iterm2", pkgs[1].Name)
	assert.True(t, pkgs[1].IsCask, "type=cask should set IsCask")
	assert.False(t, pkgs[1].IsNpm)

	assert.Equal(t, "typescript", pkgs[2].Name)
	assert.True(t, pkgs[2].IsNpm, "type=npm should set IsNpm")
	assert.False(t, pkgs[2].IsCask)
}

func TestQueryAPI_200_EmptyResults(t *testing.T) {
	body := marshalResponse(t, []searchResult{})
	srv := newTestServer(t, http.StatusOK, body)
	setAPIURL(t, srv.URL)

	pkgs, err := queryAPI("homebrew", "nonexistent-package-xyz")

	require.NoError(t, err)
	assert.Nil(t, pkgs, "empty results should return nil slice")
}

func TestQueryAPI_429_RateLimited(t *testing.T) {
	// httputil.Do retries once on 429; serve 429 for every request so the
	// retry also gets 429 and the function returns a RateLimitError.
	// The wrapped error message is "Rate limited. Please wait N seconds and try again."
	srv := newTestServer(t, http.StatusTooManyRequests, "rate limited")
	setAPIURL(t, srv.URL)

	pkgs, err := queryAPI("homebrew", "git")

	require.Error(t, err, "429 should produce a non-nil error")
	assert.Nil(t, pkgs)
	// httputil.Do converts a persistent 429 into a RateLimitError whose message
	// contains "Rate limited" — the outer wrapper adds the endpoint prefix.
	assert.Contains(t, err.Error(), "Rate limited", "error message should indicate rate limiting")
}

func TestQueryAPI_500_ServerError(t *testing.T) {
	srv := newTestServer(t, http.StatusInternalServerError, "internal server error")
	setAPIURL(t, srv.URL)

	pkgs, err := queryAPI("homebrew", "git")

	require.Error(t, err, "500 should produce a non-nil error")
	assert.Nil(t, pkgs)
	assert.Contains(t, err.Error(), "500", "error message should mention 500")
}

func TestQueryAPI_InvalidJSON(t *testing.T) {
	srv := newTestServer(t, http.StatusOK, `{not valid json}`)
	setAPIURL(t, srv.URL)

	pkgs, err := queryAPI("homebrew", "git")

	require.Error(t, err, "invalid JSON should produce a non-nil error")
	assert.Nil(t, pkgs)
}

func TestQueryAPI_NetworkError(t *testing.T) {
	// Start a server, capture its URL, then close it before the request is made.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // close immediately — connection will be refused
	t.Setenv("OPENBOOT_API_URL", url)

	pkgs, err := queryAPI("homebrew", "git")

	require.Error(t, err, "closed server should produce a non-nil error")
	assert.Nil(t, pkgs)
}

// --- SearchOnline tests ---

func TestSearchOnline_EmptyQuery(t *testing.T) {
	// No server needed; empty query returns early.
	pkgs, err := SearchOnline("")

	require.NoError(t, err)
	assert.Nil(t, pkgs)
}

func TestSearchOnline_CombinesBrewAndNpm(t *testing.T) {
	// Both "homebrew" and "npm" endpoints share the same mock server.
	// The server returns one result regardless of path.
	result := searchResult{Name: "ripgrep", Desc: "Fast grep", Type: "formula"}
	body := marshalResponse(t, []searchResult{result})
	srv := newTestServer(t, http.StatusOK, body)
	setAPIURL(t, srv.URL)

	pkgs, err := SearchOnline("ripgrep")

	require.NoError(t, err)
	// Two requests (homebrew + npm), each returns one result → 2 total.
	assert.Len(t, pkgs, 2)
}

func TestSearchOnline_BothEndpointsError_ReturnsError(t *testing.T) {
	srv := newTestServer(t, http.StatusInternalServerError, "error")
	setAPIURL(t, srv.URL)

	pkgs, err := SearchOnline("git")

	// Both endpoints fail; no results → firstErr propagated.
	require.Error(t, err)
	assert.Empty(t, pkgs)
	assert.Contains(t, err.Error(), "500")
}

func TestSearchOnline_PartialSuccess_ReturnsResults(t *testing.T) {
	// Serve valid JSON for any path that contains "homebrew", error for npm.
	// Because both share the same URL, we distinguish by path.
	brewResult := searchResult{Name: "wget", Desc: "HTTP client", Type: "formula"}
	brewBody := marshalResponse(t, []searchResult{brewResult})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "homebrew") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(brewBody))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	setAPIURL(t, srv.URL)

	pkgs, err := SearchOnline("wget")

	// One endpoint succeeds → results returned, error suppressed.
	require.NoError(t, err, "partial success should not propagate error when results exist")
	require.Len(t, pkgs, 1)
	assert.Equal(t, "wget", pkgs[0].Name)
}

// --- IsCask / IsNpm flag tests (via queryAPI) ---

func TestQueryAPI_CaskAndNpmFlags(t *testing.T) {
	tests := []struct {
		name       string
		resultType string
		wantCask   bool
		wantNpm    bool
	}{
		{"formula", "formula", false, false},
		{"cask", "cask", true, false},
		{"npm", "npm", false, true},
		{"unknown type", "unknown", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := searchResult{Name: "pkg", Desc: "desc", Type: tc.resultType}
			body := marshalResponse(t, []searchResult{result})
			srv := newTestServer(t, http.StatusOK, body)
			setAPIURL(t, srv.URL)

			pkgs, err := queryAPI("homebrew", "pkg")
			require.NoError(t, err)
			require.Len(t, pkgs, 1)

			got := pkgs[0]
			assert.Equal(t, tc.wantCask, got.IsCask, "IsCask mismatch for type=%s", tc.resultType)
			assert.Equal(t, tc.wantNpm, got.IsNpm, "IsNpm mismatch for type=%s", tc.resultType)
		})
	}
}

// --- Package field mapping ---

func TestQueryAPI_PackageFieldMapping(t *testing.T) {
	result := searchResult{Name: "fzf", Desc: "Fuzzy finder", Type: "formula"}
	body := marshalResponse(t, []searchResult{result})
	srv := newTestServer(t, http.StatusOK, body)
	setAPIURL(t, srv.URL)

	pkgs, err := queryAPI("homebrew", "fzf")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	assert.Equal(t, "fzf", pkg.Name)
	assert.Equal(t, "Fuzzy finder", pkg.Description)

	// Verify config.Package type is returned correctly.
	var _ config.Package = pkg
}
