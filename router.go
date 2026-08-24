package amigo

type Router struct {
	prefix string
	tags   []string

	routes     []*Route
	routers    []*Router
	middleware []Middleware
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
func (router *Router) GET[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return get(router, path, handler, options...)
}

// POST adds a typed POST handler to the router.
func (router *Router) POST[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return post(router, path, handler, options...)
}

// PUT adds a typed PUT handler to the router.
func (router *Router) PUT[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return put(router, path, handler, options...)
}

// PATCH adds a typed PATCH handler to the router.
func (router *Router) PATCH[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return patch(router, path, handler, options...)
}

// DELETE adds a typed DELETE handler to the router.
func (router *Router) DELETE[In, Out any](path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return deleteRoute(router, path, handler, options...)
}

// Include adds a child Router.
func (router *Router) Include(child *Router) {
	router.routers = append(router.routers, child)
}

// Use adds middleware to every route in the router and its children.
func (router *Router) Use(middleware ...Middleware) {
	router.middleware = append(router.middleware, middleware...)
}

func (router *Router) addRoute(route *Route) {
	router.routes = append(router.routes, route)
}
