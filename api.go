package amigo

import (
	"net/http"
	"slices"
	"sync"
)

// API is a small typed wrapper around http.ServeMux.
type API struct {
	root         *Router
	operations   []*Operation
	mux          *http.ServeMux
	errorHandler ErrorHandler
	buildOnce    sync.Once
}

// New creates an empty API.
func New() *API {
	return &API{
		root:         NewRouter(),
		mux:          http.NewServeMux(),
		errorHandler: DefaultErrorHandler,
	}
}

// SetErrorHandler replaces the handler used for request errors.
// It must be called before Handler, Run, or the first request.
// Passing nil restores DefaultErrorHandler.
func (api *API) SetErrorHandler(handler ErrorHandler) {
	if handler == nil {
		handler = DefaultErrorHandler
	}
	api.errorHandler = handler
}

// Handler builds the routes once and returns the final HTTP handler.
func (api *API) Handler() http.Handler {
	api.build()
	return api.mux
}

// Run builds the routes and starts an HTTP server.
func (api *API) Run(address string) error {
	return http.ListenAndServe(address, api.Handler())
}

// ServeHTTP makes API an http.Handler.
func (api *API) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	api.Handler().ServeHTTP(w, req)
}

// GET adds a typed GET handler to the root router.
func (api *API) GET[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return get(api.root, path, handler, options...)
}

// POST adds a typed POST handler to the root router.
func (api *API) POST[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return post(api.root, path, handler, options...)
}

// PUT adds a typed PUT handler to the root router.
func (api *API) PUT[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return put(api.root, path, handler, options...)
}

// PATCH adds a typed PATCH handler to the root router.
func (api *API) PATCH[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return patch(api.root, path, handler, options...)
}

// DELETE adds a typed DELETE handler to the root router.
func (api *API) DELETE[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return deleteRoute(api.root, path, handler, options...)
}

// Include adds a Router to the root router.
func (api *API) Include(router *Router) {
	api.root.Include(router)
}

func (api *API) build() {
	api.buildOnce.Do(func() {
		api.buildRouter(api.root, routerContext{})
	})
}

type routerContext struct {
	prefix     string
	tags       []string
	middleware []Middleware
}

func (api *API) buildRouter(router *Router, parent routerContext) {
	current := routerContext{
		prefix:     joinPath(parent.prefix, router.prefix),
		tags:       slices.Concat(parent.tags, router.tags),
		middleware: slices.Concat(parent.middleware, router.middleware),
	}

	for _, route := range router.routes {
		handler := route.handlerFactory(api.errorHandler)
		operation := &Operation{
			Method:  route.Method,
			Path:    joinPath(current.prefix, route.Path),
			Tags:    slices.Concat(current.tags, route.Tags),
			Input:   route.Input,
			Output:  route.Output,
			handler: applyMiddleware(handler, current.middleware),
		}

		api.operations = append(api.operations, operation)
		api.mux.Handle(operation.Method+" "+operation.Path, operation.handler)
	}

	for _, child := range router.routers {
		api.buildRouter(child, current)
	}
}
