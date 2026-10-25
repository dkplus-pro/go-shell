package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dkplus-pro/go-shell/apps/server/internal/metrics"
)

// resolveRoute maps a concrete request path to a low-cardinality route
// label: known infra routes match exactly, pprof collapses to one label,
// everything else reports "unmatched".
func resolveRoute(path string) string {
	switch {
	case path == "/healthz", path == "/readyz", path == "/metrics":
		return path
	case strings.HasPrefix(path, "/debug/pprof"):
		return "/debug/pprof"
	default:
		return "unmatched"
	}
}

// Metrics records request count and latency for every completed request.
func Metrics(m *metrics.HTTPMetrics) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			route := resolveRoute(r.URL.Path)
			m.Requests.WithLabelValues(r.Method, route, strconv.Itoa(rec.status)).Inc()
			m.Duration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		})
	}
}
