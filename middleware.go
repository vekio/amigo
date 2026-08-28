package amigo

import (
	"net/http"
	"slices"
)

// Middleware wraps an HTTP handler with cross-cutting request behavior.
// Middleware declared first is the outermost wrapper and executes first.
type Middleware func(http.Handler) http.Handler

func validateMiddlewares(middlewares []Middleware) {
	for _, middleware := range middlewares {
		if middleware == nil {
			panic("amigo: middleware cannot be nil")
		}
	}
}

func applyMiddlewares(handler http.Handler, middlewares []Middleware) http.Handler {
	for _, middleware := range slices.Backward(middlewares) {
		handler = middleware(handler)
		if handler == nil {
			panic("amigo: middleware returned a nil handler")
		}
	}
	return handler
}
