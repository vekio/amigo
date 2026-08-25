package amigo

import (
	"fmt"
	"net/http"
	"slices"
)

// Router groups routes, middleware, and snapshots of included routers. Configure
// it from one goroutine. Including it in another Router or API copies its current
// state, so it can be changed and reused afterwards without affecting that copy.
type Router struct {
	prefix string
	tags   []string

	routes          []route
	staticMounts    []staticMount
	middleware      []Middleware
	includedRouters map[*Router]struct{}
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

// Include adds a snapshot of a child Router.
func (router *Router) Include(child *Router) {
	if child == nil {
		panic("amigo: cannot include a nil router")
	}
	if child == router || child.contains(router) {
		panic("amigo: router inclusion cycle detected")
	}

	for _, childRoute := range child.routes {
		snapshot := childRoute
		snapshot.path = joinPath(child.prefix, snapshot.path)
		snapshot.tags = slices.Concat(child.tags, snapshot.tags)
		snapshot.middleware = slices.Concat(child.middleware, snapshot.middleware)
		router.routes = append(router.routes, snapshot)
	}
	for _, childMount := range child.staticMounts {
		snapshot := childMount
		snapshot.path = joinPath(child.prefix, snapshot.path)
		snapshot.middleware = slices.Concat(child.middleware, snapshot.middleware)
		router.staticMounts = append(router.staticMounts, snapshot)
	}

	if router.includedRouters == nil {
		router.includedRouters = make(map[*Router]struct{})
	}
	router.includedRouters[child] = struct{}{}
	for included := range child.includedRouters {
		router.includedRouters[included] = struct{}{}
	}
}

// Use adds middleware to every route in the router, including router snapshots.
func (router *Router) Use(middleware ...Middleware) {
	for index, current := range middleware {
		if current == nil {
			panic(fmt.Sprintf("amigo: middleware at index %d is nil", index))
		}
		router.middleware = append(router.middleware, current)
	}
}

func (router *Router) addRoute(route route) {
	router.routes = append(router.routes, route)
}

func (router *Router) contains(target *Router) bool {
	if router == target {
		return true
	}
	_, exists := router.includedRouters[target]
	return exists
}
