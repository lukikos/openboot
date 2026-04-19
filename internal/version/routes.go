package version

import "net/http"

// RegisterRoutes registers version endpoints on the given mux.
// Also registers /v on a shorter alias for convenience.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/version", Handler)
	mux.HandleFunc("/v", Handler)
}
