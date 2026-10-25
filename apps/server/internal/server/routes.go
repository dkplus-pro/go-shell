package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/dkplus-pro/go-shell/apps/server/internal/config"
	"github.com/dkplus-pro/go-shell/apps/server/internal/metrics"
	"github.com/dkplus-pro/go-shell/apps/server/internal/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ReadinessChecker reports whether the server can serve traffic. Phase 0
// has no external dependencies to probe; Phase 2+ registers a PG ping.
type ReadinessChecker func(ctx context.Context) error

// NewHandler assembles the middleware chain and the route table.
//
// Middleware order (outermost first) per docs/plans/go-backend-v1.md §5:
//
//	recovery → requestID → logging → metrics → timeout
//
// Reserved slots, intentionally empty in Phase 0:
//
//	TODO(phase-2): authn (JWT validation) — after timeout.
//	TODO(phase-3): rbac (permission check) — after authn.
func NewHandler(cfg config.Config, logger *slog.Logger, reg *prometheus.Registry, readiness ...ReadinessChecker) http.Handler {
	httpMetrics := metrics.NewHTTPMetrics(reg)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.Handle("/readyz", handleReadiness(readiness))
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	if cfg.EnablePprof {
		mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	}

	return middleware.Chain(mux,
		middleware.Recovery(logger),
		middleware.RequestID,
		middleware.Logging(logger),
		middleware.Metrics(httpMetrics),
		middleware.Timeout(cfg.RequestTimeout),
	)
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleReadiness(checkers []ReadinessChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		for _, check := range checkers {
			if err := check(ctx); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
}

func writeJSON(w http.ResponseWriter, code int, body map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
