package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder captures the response status code emitted by the handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Logging emits one structured log line per completed request.
func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.InfoContext(r.Context(), "http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", float64(time.Since(start).Microseconds())/1000.0,
				"request_id", RequestIDFromContext(r.Context()),
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}
