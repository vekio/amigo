package amigo

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// Router groups routes below a shared prefix and middleware chain. A router may
// create nested groups; it always remains owned by the API that created it.
type Router struct {
	api         *API
	parent      *Router
	prefix      string
	middlewares []Middleware
}

func newRouter(api *API, parent *Router, prefix string, middlewares []Middleware) *Router {
	return &Router{
		api:         api,
		parent:      parent,
		prefix:      prefix,
		middlewares: slices.Clone(middlewares),
	}
}

// GET registers a typed GET endpoint below the router prefix. An empty path
// addresses the prefix itself.
func (router *Router) GET[In, Out any](path string, endpoint EndpointFunc[In, Out], options ...RouteOption) {
	router.registerEndpoint(http.MethodGet, path, endpoint, options...)
}

// POST registers a typed POST endpoint below the router prefix. An empty path
// addresses the prefix itself.
func (router *Router) POST[In, Out any](path string, endpoint EndpointFunc[In, Out], options ...RouteOption) {
	router.registerEndpoint(http.MethodPost, path, endpoint, options...)
}

// PUT registers a typed PUT endpoint below the router prefix. An empty path
// addresses the prefix itself.
func (router *Router) PUT[In, Out any](path string, endpoint EndpointFunc[In, Out], options ...RouteOption) {
	router.registerEndpoint(http.MethodPut, path, endpoint, options...)
}

// PATCH registers a typed PATCH endpoint below the router prefix. An empty path
// addresses the prefix itself.
func (router *Router) PATCH[In, Out any](path string, endpoint EndpointFunc[In, Out], options ...RouteOption) {
	router.registerEndpoint(http.MethodPatch, path, endpoint, options...)
}

// DELETE registers a typed DELETE endpoint below the router prefix. An empty
// path addresses the prefix itself.
func (router *Router) DELETE[In, Out any](path string, endpoint EndpointFunc[In, Out], options ...RouteOption) {
	router.registerEndpoint(http.MethodDelete, path, endpoint, options...)
}

// RAW registers an endpoint below the router prefix without typed request
// binding or response encoding.
func (router *Router) RAW(method string, path string, endpoint RawEndpointFunc, options ...RouteOption) {
	if endpoint == nil {
		panic("amigo: raw endpoint cannot be nil")
	}
	route := router.buildRoute(method, path, options...)
	handler := applyMiddlewares(rawEndpointHandler(router.api.logger, route, endpoint), route.middlewares)
	router.api.mux.Handle(route.pattern(), handler)
}

func (router *Router) registerEndpoint[In, Out any](
	method string,
	path string,
	endpoint EndpointFunc[In, Out],
	options ...RouteOption,
) {
	if endpoint == nil {
		panic("amigo: endpoint cannot be nil")
	}

	endpointRoute := router.buildRoute(method, path, options...)
	input := buildInputMetadata[In](endpointRoute.path, router.api.validators)
	output := buildOutputMetadata[Out]()
	handler := endpointHandler(router.api.logger, endpointRoute, input, output, endpoint)

	router.api.mux.Handle(
		endpointRoute.pattern(),
		applyMiddlewares(handler, endpointRoute.middlewares),
	)
}

// Group creates a child router. Parent middleware is applied before middleware
// declared by the child and by individual routes.
func (router *Router) Group(prefix string, middlewares ...Middleware) *Router {
	checkGroupPrefix(prefix)
	validateMiddlewares(middlewares)

	fullPrefix := joinPath(router.prefix, prefix)
	return newRouter(router.api, router, fullPrefix, middlewares)
}

func (router *Router) buildRoute(method string, path string, options ...RouteOption) route {
	fullPath := router.routePath(path)
	route := newRoute(method, fullPath, options...)
	route.middlewares = append(router.inheritedMiddlewares(), route.middlewares...)
	return route
}

func (router *Router) inheritedMiddlewares() []Middleware {
	lineage := []*Router{}
	for current := router; current != nil; current = current.parent {
		lineage = append(lineage, current)
	}

	middlewares := []Middleware{}
	for _, current := range slices.Backward(lineage) {
		middlewares = append(middlewares, current.middlewares...)
	}
	return middlewares
}

func (router *Router) routePath(path string) string {
	if path == "" {
		if router.prefix == "" {
			return "/"
		}
		return router.prefix
	}
	if !strings.HasPrefix(path, "/") {
		panic(fmt.Sprintf("amigo: route path %q must start with a slash", path))
	}
	return joinPath(router.prefix, path)
}

func joinPath(prefix string, path string) string {
	if prefix == "" {
		return path
	}
	return prefix + path
}
