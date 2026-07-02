// Package handlers contains the HTTP handlers for the demo API. The endpoints
// are intentionally lightweight but exercise the full observability stack:
// health/readiness probes for Kubernetes, a variable-latency "work" endpoint
// to populate the latency histogram, and a fault-injection endpoint to
// demonstrate error-rate alerting.
package handlers

import (
	"encoding/json"
	"log/slog"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"devops-showcase/internal/version"
)

// API wires the handlers together with their dependencies.
type API struct {
	log   *slog.Logger
	env   string
	ready atomic.Bool
}

// New returns an API. The service starts "not ready" until MarkReady is
// called, so the readiness probe fails until initialisation completes.
func New(log *slog.Logger, env string) *API {
	a := &API{log: log, env: env}
	return a
}

// MarkReady flips the readiness flag to true.
func (a *API) MarkReady() { a.ready.Store(true) }

// MarkUnready flips the readiness flag to false (used during shutdown).
func (a *API) MarkUnready() { a.ready.Store(false) }

// Routes registers all handlers on a Go 1.22+ ServeMux. Method-qualified
// patterns give us clean route labels for metrics.
func (a *API) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", a.root)
	mux.HandleFunc("GET /healthz", a.healthz)
	mux.HandleFunc("GET /readyz", a.readyz)
	mux.HandleFunc("GET /api/hello", a.hello)
	mux.HandleFunc("GET /api/work", a.work)
	mux.HandleFunc("GET /api/error", a.errorProne)
	return mux
}

func (a *API) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		a.log.Error("failed to encode response", "error", err)
	}
}

// root returns service metadata.
func (a *API) root(w http.ResponseWriter, _ *http.Request) {
	a.writeJSON(w, http.StatusOK, map[string]any{
		"service":     "devops-showcase",
		"environment": a.env,
		"build":       version.Get(),
		"endpoints": []string{
			"GET /healthz", "GET /readyz", "GET /metrics",
			"GET /api/hello", "GET /api/work", "GET /api/error",
		},
	})
}

// healthz is the liveness probe: it only reports that the process is running.
func (a *API) healthz(w http.ResponseWriter, _ *http.Request) {
	a.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyz is the readiness probe: it reports whether the service should
// receive traffic. Kubernetes removes the pod from Service endpoints when
// this returns non-200.
func (a *API) readyz(w http.ResponseWriter, _ *http.Request) {
	if !a.ready.Load() {
		a.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// hello is a trivial JSON greeting.
func (a *API) hello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "world"
	}
	a.writeJSON(w, http.StatusOK, map[string]string{"message": "hello, " + name})
}

// work simulates a downstream call with variable latency so the request
// duration histogram shows a realistic spread in Grafana.
func (a *API) work(w http.ResponseWriter, _ *http.Request) {
	// 10-260ms of simulated work.
	delay := time.Duration(10+rand.Intn(250)) * time.Millisecond
	time.Sleep(delay)
	a.writeJSON(w, http.StatusOK, map[string]any{
		"result":     "completed",
		"latency_ms": delay.Milliseconds(),
	})
}

// errorProne fails ~30% of the time to demonstrate error-rate metrics and
// Prometheus alerting rules.
func (a *API) errorProne(w http.ResponseWriter, _ *http.Request) {
	if rand.Float64() < 0.3 {
		a.log.Warn("injected failure on /api/error")
		a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "injected failure"})
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
