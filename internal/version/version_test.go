package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openboot/openboot/internal/version"
)

func TestGet(t *testing.T) {
	info := version.Get()

	if info.GoVersion == "" {
		t.Error("expected non-empty GoVersion")
	}
	if info.OS == "" {
		t.Error("expected non-empty OS")
	}
	if info.Arch == "" {
		t.Error("expected non-empty Arch")
	}
}

func TestHandler_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	version.Handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var info version.Info
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if info.GoVersion == "" {
		t.Error("expected non-empty GoVersion in response")
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/version", nil)
		w := httptest.NewRecorder()

		version.Handler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("method %s: expected 405, got %d", method, w.Code)
		}
	}
}
