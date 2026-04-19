package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openboot/openboot/internal/health"
)

func TestHandler_OK(t *testing.T) {
	h := health.Handler("1.0.0-test")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	h(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var s health.Status
	if err := json.NewDecoder(rr.Body).Decode(&s); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if s.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", s.Status)
	}

	if s.Version != "1.0.0-test" {
		t.Errorf("expected version '1.0.0-test', got %q", s.Version)
	}

	if s.GoVersion == "" {
		t.Error("expected non-empty go_version")
	}

	// Also verify the Content-Type header is set correctly
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := health.Handler("1.0.0")

	// Also checking PATCH since it's another mutating method that should be rejected
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req := httptest.NewRequest(method, "/health", nil)
		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: expected 405, got %d", method, rr.Code)
		}
	}
}
