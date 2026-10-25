package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type contextKey int

const requestIDKey contextKey = iota

// RequestIDHeader is the inbound/outbound header carrying the request id.
const RequestIDHeader = "X-Request-Id"

// RequestIDFromContext returns the request id attached by RequestID, if any.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// RequestID propagates an existing X-Request-Id or generates a random one,
// stores it in the request context and echoes it on the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

// newRequestID never panics: on the (unreachable) crypto/rand failure it
// returns an empty id, which merely disables correlation for this request.
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}
