// Package healthcheck serves liveness and readiness probes for k8s / ECS / ALB.
//
//   - /livez  answers 200 while the process can serve HTTP. It never checks
//     dependencies: a database outage must not make the orchestrator restart
//     every replica.
//   - /readyz answers 200 only when every opened dependency responds and the
//     service is not shutting down. Load balancers use it to route traffic.
package healthcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// Check pings one dependency. Returning an error marks the service not ready.
type Check func(ctx context.Context) error

const checkTimeout = 2 * time.Second

type Probe struct {
	checks   func() map[string]Check
	draining atomic.Bool
}

func NewProbe(checks func() map[string]Check) *Probe {
	return &Probe{checks: checks}
}

// SetDraining makes /readyz fail so load balancers stop sending new traffic
// before the servers stop.
func (p *Probe) SetDraining() { p.draining.Store(true) }

func (p *Probe) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "UP"})
}

func (p *Probe) Ready(w http.ResponseWriter, r *http.Request) {
	if p.draining.Load() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "DRAINING"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), checkTimeout)
	defer cancel()

	failed := map[string]string{}
	for name, check := range p.checks() {
		if err := check(ctx); err != nil {
			failed[name] = err.Error()
		}
	}
	if len(failed) > 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "DOWN", "failed": failed})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "UP"})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
