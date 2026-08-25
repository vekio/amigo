package amigo

import (
	"net/http"
	"sync"
)

// API is a small typed wrapper around http.ServeMux. Create one with New and
// configure it from one goroutine before building it. Once built, an API can
// serve requests concurrently but can no longer be configured.
type API struct {
	root         *Router
	operations   []*Operation
	mux          *http.ServeMux
	errorHandler ErrorHandler
	validators   validatorRegistry
	buildOnce    sync.Once
	built        bool
	buildFailure any
}

// New creates an empty API.
func New() *API {
	return &API{
		root:         NewRouter(),
		errorHandler: DefaultErrorHandler,
		validators:   make(validatorRegistry),
	}
}

// SetErrorHandler replaces the handler used for request errors.
// It must be called before Handler, Run, or the first request.
// Passing nil restores DefaultErrorHandler.
func (api *API) SetErrorHandler(handler ErrorHandler) {
	if api.built {
		panic("amigo: cannot set the error handler after the API has been built")
	}
	if handler == nil {
		handler = DefaultErrorHandler
	}
	api.errorHandler = handler
}

// Handler builds the routes once and returns the final HTTP handler.
func (api *API) Handler() http.Handler {
	api.build()
	return http.HandlerFunc(api.mux.ServeHTTP)
}

// Run builds the routes and starts an HTTP server.
func (api *API) Run(address string) error {
	return http.ListenAndServe(address, api.Handler())
}

// ServeHTTP makes API an http.Handler.
func (api *API) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	api.build()
	api.mux.ServeHTTP(w, req)
}

// GET adds a typed GET handler to the root router.
func (api *API) GET[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	register(api.root, http.MethodGet, path, handler, options...)
}

// POST adds a typed POST handler to the root router.
func (api *API) POST[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	register(api.root, http.MethodPost, path, handler, options...)
}

// PUT adds a typed PUT handler to the root router.
func (api *API) PUT[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	register(api.root, http.MethodPut, path, handler, options...)
}

// PATCH adds a typed PATCH handler to the root router.
func (api *API) PATCH[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	register(api.root, http.MethodPatch, path, handler, options...)
}

// DELETE adds a typed DELETE handler to the root router.
func (api *API) DELETE[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	register(api.root, http.MethodDelete, path, handler, options...)
}

// Include adds a Router to the root router.
func (api *API) Include(router *Router) {
	api.root.Include(router)
}

// Use adds middleware to every route in the API.
func (api *API) Use(middleware ...Middleware) {
	api.root.Use(middleware...)
}

// Operations builds the API and returns a snapshot of its operations.
func (api *API) Operations() []Operation {
	api.build()

	operations := make([]Operation, len(api.operations))
	for index, operation := range api.operations {
		operations[index] = operation.clone()
	}
	return operations
}
