package health

import "net/http"

// RegisterRoutes attaches health check routes to the provided ServeMux.
// The version string is embedded in the response payload.
//
// Routes registered:
//
//	GET /health  — liveness probe
func RegisterRoutes(mux *http.ServeMux, version string) {
	mux.Handle("GET /health", Handler(version))
}
