package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery converts panics into 500 responses with an error log carrying
// the request id. It is a last-resort backstop, not an error handling
// strategy (AGENTS.md rule 5).
func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.ErrorContext(r.Context(), "panic recovered",
						"panic", rec,
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", RequestIDFromContext(r.Context()),
						"stack", string(debug.Stack()),
					)
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
