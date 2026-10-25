// Package metrics owns the Prometheus registry and the canonical HTTP
// metric definitions.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTPMetrics collects per-request counters and latency histograms.
// The route label uses path templates ("/healthz", "/debug/pprof",
// "unmatched") to keep label cardinality bounded.
type HTTPMetrics struct {
	// Requests counts completed requests by method, route and status code.
	Requests *prometheus.CounterVec
	// Duration observes request latency in seconds by method and route.
	Duration *prometheus.HistogramVec
}

// NewRegistry creates a fresh registry with Go runtime and process collectors.
func NewRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	return reg
}

// NewHTTPMetrics registers the HTTP metrics on reg and returns them.
func NewHTTPMetrics(reg *prometheus.Registry) *HTTPMetrics {
	return &HTTPMetrics{
		Requests: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of completed HTTP requests.",
		}, []string{"method", "route", "code"}),
		Duration: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
	}
}
