package health

import "net/http"

// RegisterRoutes attaches health check routes to the provided ServeMux.
// The version string is embedded in the response payload.
//
// Routes registered:
//
//	GET /health   — liveness probe
//	GET /healthz  — alias for liveness probe (kubectl convention)
func RegisterRoutes(mux *http.ServeMux, version string) {
	h := Handler(version)
	mux.Handle("GET /health", h)
	mux.Handle("GET /healthz", h)
}
