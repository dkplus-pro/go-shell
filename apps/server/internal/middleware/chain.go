// Package middleware provides composable net/http middlewares shared by all
// HTTP routes (infra endpoints today, Connect handlers from Phase 1 on).
package middleware

import "net/http"

// Middleware wraps an http.Handler with additional behaviour.
type Middleware func(http.Handler) http.Handler

// Chain composes middlewares around h. The first middleware in the list is
// the outermost: Chain(h, a, b) serves requests as a(b(h)).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
