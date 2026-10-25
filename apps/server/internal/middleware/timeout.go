package middleware

import (
	"net/http"
	"time"
)

// Timeout bounds request handling time via http.TimeoutHandler, which
// answers 503 "request timeout" when the deadline is exceeded. Applied
// innermost, right before the routed handler.
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, "request timeout")
	}
}
