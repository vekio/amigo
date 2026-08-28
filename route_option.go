package amigo

import (
	"fmt"
	"net/http"
)

// RouteOption configures one route when it is registered. Values are created by
// WithStatus, WithMaxBodyBytes, WithMiddleware, and WithErrorMapping.
type RouteOption func(*route)

// WithStatus sets the endpoint's successful HTTP status. Only 2xx statuses are
// accepted; the default is 200 OK.
func WithStatus(status int) RouteOption {
	if status < http.StatusOK || status > 299 {
		panic(fmt.Sprintf("amigo: success status must be between 200 and 299, got %d", status))
	}

	return func(route *route) {
		route.status = status
	}
}

// WithMaxBodyBytes limits the request body before it is decoded. Zero disables
// the limit; routes default to one MiB.
func WithMaxBodyBytes(limit int64) RouteOption {
	if limit < 0 {
		panic("amigo: maximum body size cannot be negative")
	}

	return func(route *route) {
		route.maxBodyBytes = limit
	}
}

// WithMiddleware adds middleware to a route in declaration order. Router
// middleware runs before route middleware.
func WithMiddleware(middlewares ...Middleware) RouteOption {
	validateMiddlewares(middlewares)

	return func(route *route) {
		route.middlewares = append(route.middlewares, middlewares...)
	}
}

// WithErrorMapping translates errors matching target through [errors.Is] into
// an RFC 9457 response. publicDetail must be safe to expose to HTTP clients;
// the original error is never copied into the response automatically.
func WithErrorMapping(target error, status int, publicDetail string) RouteOption {
	if target == nil {
		panic("amigo: mapped error cannot be nil")
	}
	if status < http.StatusBadRequest || status > 599 {
		panic(fmt.Sprintf("amigo: error status must be between 400 and 599, got %d", status))
	}

	return func(route *route) {
		route.errorMappings = append(route.errorMappings, errorMapping{
			target:       target,
			status:       status,
			publicDetail: publicDetail,
		})
	}
}
