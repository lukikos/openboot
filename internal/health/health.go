package health

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// Status represents the health status of the service.
type Status struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	GoVersion string    `json:"go_version"`
	Uptime    string    `json:"uptime"`
}

var startTime = time.Now()

// Handler returns an HTTP handler for the health check endpoint.
// The version string is typically set at build time via -ldflags.
func Handler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		s := Status{
			Status:    "ok",
			Timestamp: time.Now().UTC(),
			Version:   version,
			GoVersion: runtime.Version(),
			Uptime:    time.Since(startTime).Round(time.Second).String(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(s)
	}
}
