package amigo

import (
	"fmt"
	"net/http"
)

// Router groups routes, middleware, and child routers. Configure it from one
// goroutine before the API is built; serving requests does not mutate it.
type Router struct {
	prefix string
	tags   []string

	routes       []*route
	routers      []*Router
	staticMounts []staticMount
	middleware   []Middleware
	frozen       bool
}

// NewRouter creates a group of related routes.
func NewRouter(options ...RouterOption) *Router {
	router := &Router{}
	for _, option := range options {
		option.applyRouter(router)
	}
	return router
}

// GET adds a typed GET handler to the router.
func (router *Router) GET[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	register(router, http.MethodGet, path, handler, options...)
}

// POST adds a typed POST handler to the router.
func (router *Router) POST[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	register(router, http.MethodPost, path, handler, options...)
}

// PUT adds a typed PUT handler to the router.
func (router *Router) PUT[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	register(router, http.MethodPut, path, handler, options...)
}

// PATCH adds a typed PATCH handler to the router.
func (router *Router) PATCH[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	register(router, http.MethodPatch, path, handler, options...)
}

// DELETE adds a typed DELETE handler to the router.
func (router *Router) DELETE[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) {
	register(router, http.MethodDelete, path, handler, options...)
}

// Include adds a child Router.
func (router *Router) Include(child *Router) {
	router.assertMutable()
	if child == nil {
		panic("amigo: cannot include a nil router")
	}
	router.routers = append(router.routers, child)
}

// Use adds middleware to every route in the router and its children.
func (router *Router) Use(middleware ...Middleware) {
	router.assertMutable()
	for index, current := range middleware {
		if current == nil {
			panic(fmt.Sprintf("amigo: middleware at index %d is nil", index))
		}
		router.middleware = append(router.middleware, current)
	}
}

func (router *Router) addRoute(route *route) {
	router.assertMutable()
	router.routes = append(router.routes, route)
}

func (router *Router) assertMutable() {
	if router.frozen {
		panic("amigo: cannot modify a router after its API has been built")
	}
}
