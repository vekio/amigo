package amigo

import (
	"net/http"
	"slices"
)

// Middleware wraps an HTTP handler using the request and response primitives
// from net/http.
type Middleware func(http.ResponseWriter, *http.Request, http.Handler)

func applyMiddleware(handler http.Handler, middleware []Middleware) http.Handler {
	for _, current := range slices.Backward(middleware) {
		next := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			current(w, req, next)
		})
	}
	return handler
}
