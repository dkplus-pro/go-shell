package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestChainOrderOutermostFirst(t *testing.T) {
	var order []string
	mk := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), mk("a"), mk("b"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("order = %v, want [a b]", order)
	}
}

func TestRequestIDGenerated(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	id := rec.Header().Get(RequestIDHeader)
	if len(id) != 32 {
		t.Fatalf("X-Request-Id = %q, want 32 hex chars", id)
	}
}

func TestRequestIDPropagated(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got != "fixed-id" {
			t.Errorf("context id = %q, want fixed-id", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "fixed-id")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get(RequestIDHeader); got != "fixed-id" {
		t.Errorf("echoed id = %q, want fixed-id", got)
	}
}

func TestRecoveryReturns500(t *testing.T) {
	h := Recovery(quietLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestTimeoutReturns503(t *testing.T) {
	h := Timeout(30 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestResolveRoute(t *testing.T) {
	cases := map[string]string{
		"/healthz":          "/healthz",
		"/readyz":           "/readyz",
		"/metrics":          "/metrics",
		"/debug/pprof/":     "/debug/pprof",
		"/debug/pprof/heap": "/debug/pprof",
		"/api/v1/nope":      "unmatched",
	}
	for path, want := range cases {
		if got := resolveRoute(path); got != want {
			t.Errorf("resolveRoute(%q) = %q, want %q", path, got, want)
		}
	}
}
