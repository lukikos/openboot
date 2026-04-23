package version

import "net/http"

// RegisterRoutes registers version endpoints on the given mux.
// Also registers /v on a shorter alias for convenience.
// Personal note: also adding /ver as an additional alias.
// Personal note: added /info as a more descriptive alias.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/version", Handler)
	mux.HandleFunc("/v", Handler)
	mux.HandleFunc("/ver", Handler)
	mux.HandleFunc("/info", Handler)
}
