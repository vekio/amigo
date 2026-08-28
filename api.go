// Package amigo provides a small, typed layer over net/http.
//
// Endpoints use struct inputs and outputs so request binding and response
// encoding stay outside application handlers. Raw endpoints remain available
// when direct access to net/http is required.
package amigo

import (
	"log/slog"
	"net/http"
)

// API owns the HTTP route tree and its underlying [http.ServeMux].
// Routes should be registered during application startup.
type API struct {
	mux        *http.ServeMux
	root       *Router
	logger     *slog.Logger
	validators validatorRegistry
}

// New creates an HTTP application with a root router.
func New(options ...APIOption) *API {
	api := &API{
		mux:        http.NewServeMux(),
		logger:     slog.Default(),
		validators: newValidatorRegistry(),
	}
	for _, option := range options {
		if option == nil {
			panic("amigo: API option cannot be nil")
		}
		option(api)
	}
	api.root = newRouter(api, nil, "", nil)
	return api
}

// Group creates a child router for prefix. Its middleware is inherited by all
// routes and nested groups registered below it.
func (app *API) Group(prefix string, middlewares ...Middleware) *Router {
	return app.root.Group(prefix, middlewares...)
}

// ServeHTTP implements http.Handler.
func (app *API) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	handler, pattern := app.mux.Handler(request)
	if pattern == "" {
		fallback := inspectMuxFallback(handler, request)
		if fallback.status == http.StatusNotFound || fallback.status == http.StatusMethodNotAllowed {
			writeRoutingProblem(w, request, fallback)
			return
		}
	}
	app.mux.ServeHTTP(w, request)
}

// GET registers a typed GET endpoint on the root router.
func (app *API) GET[In, Out any](path string, endpoint EndpointFunc[In, Out], options ...RouteOption) {
	app.root.GET(path, endpoint, options...)
}

// POST registers a typed POST endpoint on the root router.
func (app *API) POST[In, Out any](path string, endpoint EndpointFunc[In, Out], options ...RouteOption) {
	app.root.POST(path, endpoint, options...)
}

// PUT registers a typed PUT endpoint on the root router.
func (app *API) PUT[In, Out any](path string, endpoint EndpointFunc[In, Out], options ...RouteOption) {
	app.root.PUT(path, endpoint, options...)
}

// PATCH registers a typed PATCH endpoint on the root router.
func (app *API) PATCH[In, Out any](path string, endpoint EndpointFunc[In, Out], options ...RouteOption) {
	app.root.PATCH(path, endpoint, options...)
}

// DELETE registers a typed DELETE endpoint on the root router.
func (app *API) DELETE[In, Out any](path string, endpoint EndpointFunc[In, Out], options ...RouteOption) {
	app.root.DELETE(path, endpoint, options...)
}

// RAW registers an endpoint on the root router without typed request binding
// or response encoding. The endpoint owns the complete HTTP response.
func (app *API) RAW(method string, path string, endpoint RawEndpointFunc, options ...RouteOption) {
	app.root.RAW(method, path, endpoint, options...)
}
