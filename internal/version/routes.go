package version

import "net/http"

// RegisterRoutes registers version endpoints on the given mux.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/version", Handler)
}
