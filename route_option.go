package amigo

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// RouteOption configures one route when it is registered. Values are created by
// WithStatus, WithMaxBodyBytes, WithOperationID, WithSummary, WithDescription,
// WithTags, WithMiddleware, and WithErrorMapping.
type RouteOption func(*route)

// WithStatus sets the endpoint's non-error HTTP status. Statuses from 200
// through 399 are accepted; the default is 200 OK.
func WithStatus(status int) RouteOption {
	if status < http.StatusOK || status > 399 {
		panic(fmt.Sprintf("amigo: response status must be between 200 and 399, got %d", status))
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

// WithOperationID overrides the identifier generated from the HTTP method and
// path. Operation identifiers must be unique across the API.
func WithOperationID(operationID string) RouteOption {
	if strings.TrimSpace(operationID) == "" {
		panic("amigo: operation ID cannot be empty")
	}

	return func(route *route) {
		route.operationID = operationID
	}
}

// WithSummary sets the short human-readable description used for this
// operation in the OpenAPI document.
func WithSummary(summary string) RouteOption {
	if strings.TrimSpace(summary) == "" {
		panic("amigo: operation summary cannot be empty")
	}

	return func(route *route) {
		route.summary = summary
	}
}

// WithDescription sets the detailed human-readable description used for this
// operation in the OpenAPI document.
func WithDescription(description string) RouteOption {
	if strings.TrimSpace(description) == "" {
		panic("amigo: operation description cannot be empty")
	}

	return func(route *route) {
		route.description = description
	}
}

// WithTags groups this operation under one or more OpenAPI tags. Repeated tags
// are ignored while preserving declaration order.
func WithTags(tags ...string) RouteOption {
	if len(tags) == 0 {
		panic("amigo: operation tags cannot be empty")
	}
	for _, tag := range tags {
		if strings.TrimSpace(tag) == "" {
			panic("amigo: operation tag cannot be empty")
		}
	}
	tags = slices.Clone(tags)

	return func(route *route) {
		for _, tag := range tags {
			if !slices.Contains(route.tags, tag) {
				route.tags = append(route.tags, tag)
			}
		}
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
		route.errors = append(route.errors, errorMapping{
			target:       target,
			status:       status,
			publicDetail: publicDetail,
		})
	}
}
