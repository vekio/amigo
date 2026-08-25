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
	operations   []Operation
	mux          *http.ServeMux
	errorHandler ErrorHandler
	validators   validatorRegistry
	maxBodyBytes int64
	buildOnce    sync.Once
	built        bool
	buildErr     error
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
// It must be called before Compile, Handler, Run, or the first request.
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

// Compile builds the routes once and returns the final HTTP handler.
func (api *API) Compile() (http.Handler, error) {
	api.buildOnce.Do(func() {
		api.built = true
		api.mux, api.operations, api.buildErr = api.compile()
	})
	if api.buildErr != nil {
		return nil, api.buildErr
	}
	return http.HandlerFunc(api.mux.ServeHTTP), nil
}

// Handler builds the routes once and returns the final HTTP handler. It panics
// if the API configuration is invalid. Use Compile to handle that error.
func (api *API) Handler() http.Handler {
	handler, err := api.Compile()
	if err != nil {
		panic(err)
	}
	return handler
}

// Run builds the routes and starts an HTTP server.
func (api *API) Run(address string) error {
	handler, err := api.Compile()
	if err != nil {
		return err
	}
	return http.ListenAndServe(address, handler)
}

// ServeHTTP makes API an http.Handler.
func (api *API) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if _, err := api.Compile(); err != nil {
		panic(err)
	}
	api.mux.ServeHTTP(w, req)
}

// GET adds a typed GET handler to the root router.
func (api *API) GET[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	api.assertMutable("register a route")
	register(api.root, http.MethodGet, path, handler, options...)
}

// POST adds a typed POST handler to the root router.
func (api *API) POST[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	api.assertMutable("register a route")
	register(api.root, http.MethodPost, path, handler, options...)
}

// PUT adds a typed PUT handler to the root router.
func (api *API) PUT[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	api.assertMutable("register a route")
	register(api.root, http.MethodPut, path, handler, options...)
}

// PATCH adds a typed PATCH handler to the root router.
func (api *API) PATCH[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	api.assertMutable("register a route")
	register(api.root, http.MethodPatch, path, handler, options...)
}

// DELETE adds a typed DELETE handler to the root router.
func (api *API) DELETE[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	api.assertMutable("register a route")
	register(api.root, http.MethodDelete, path, handler, options...)
}

// Include adds a snapshot of a Router to the API.
func (api *API) Include(router *Router) {
	api.assertMutable("include a router")
	api.root.Include(router)
}

// Use adds middleware to every route in the API.
func (api *API) Use(middleware ...Middleware) {
	api.assertMutable("register middleware")
	api.root.Use(middleware...)
}

// Operations builds the API and returns a snapshot of its operations.
func (api *API) Operations() []Operation {
	if _, err := api.Compile(); err != nil {
		panic(err)
	}

	operations := make([]Operation, len(api.operations))
	for index := range api.operations {
		operations[index] = api.operations[index].clone()
	}
	return operations
}

// SetMaxBodyBytes sets the maximum size of a JSON request body. Zero disables
// the limit. It must be called before the API is built.
func (api *API) SetMaxBodyBytes(limit int64) {
	api.assertMutable("set the maximum request body size")
	if limit < 0 {
		panic("amigo: maximum request body size cannot be negative")
	}
	api.maxBodyBytes = limit
}

func (api *API) assertMutable(action string) {
	if api.built {
		panic("amigo: cannot " + action + " after the API has been built")
	}
}
