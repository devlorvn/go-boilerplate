package healthcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Check pings one dependency. Returning an error marks the service DOWN.
type Check func(ctx context.Context) error

// Handler runs every check (2s budget) and answers 200 {"status":"UP"} or
// 503 with the failing dependency names. Used as the k8s/ALB health probe.
func Handler(checks func() map[string]Check) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		failed := map[string]string{}
		for name, check := range checks() {
			if err := check(ctx); err != nil {
				failed[name] = err.Error()
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if len(failed) > 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "DOWN", "failed": failed})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "UP"})
	}
}
