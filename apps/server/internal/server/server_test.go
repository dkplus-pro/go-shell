package server_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dkplus-pro/go-shell/apps/server/internal/config"
	"github.com/dkplus-pro/go-shell/apps/server/internal/metrics"
	"github.com/dkplus-pro/go-shell/apps/server/internal/server"
)

func newTestHandler(t *testing.T, readiness ...server.ReadinessChecker) http.Handler {
	t.Helper()
	cfg := config.Config{EnablePprof: true, RequestTimeout: 5 * time.Second}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return server.NewHandler(cfg, logger, metrics.NewRegistry(), readiness...)
}

func do(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHealthz(t *testing.T) {
	rec := do(t, newTestHandler(t), "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("body = %q, want contains ok", rec.Body.String())
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("X-Request-Id header missing")
	}
}

func TestReadyzDefault(t *testing.T) {
	rec := do(t, newTestHandler(t), "/readyz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestReadyzCheckerFailure(t *testing.T) {
	fail := func(context.Context) error { return errors.New("pg down") }
	rec := do(t, newTestHandler(t, fail), "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestMetricsExposesRequestCounter(t *testing.T) {
	h := newTestHandler(t)
	do(t, h, "/healthz")
	rec := do(t, h, "/metrics")
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(body, "http_requests_total") {
		t.Error("http_requests_total missing from /metrics")
	}
	if !strings.Contains(body, `route="/healthz"`) {
		t.Error(`route="/healthz" label missing from /metrics`)
	}
}

func TestUnmatchedRouteLabelled(t *testing.T) {
	h := newTestHandler(t)
	do(t, h, "/definitely-not-a-route")
	if rec := do(t, h, "/definitely-not-a-route"); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := do(t, h, "/metrics").Body.String()
	if !strings.Contains(body, `route="unmatched"`) {
		t.Error(`route="unmatched" label missing from /metrics`)
	}
}

func TestPprofIndex(t *testing.T) {
	rec := do(t, newTestHandler(t), "/debug/pprof/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
