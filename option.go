package amigo

// RouterOption configures a Router.
type RouterOption interface {
	applyRouter(*Router)
}

// RouteOption configures a route before it becomes an Operation.
type RouteOption interface {
	applyRoute(*route)
}

// TagsOption configures tags on either a Router or a Route.
type TagsOption interface {
	RouterOption
	RouteOption
}

type prefixOption string

// Prefix sets the common path prefix of a Router.
func Prefix(prefix string) RouterOption {
	return prefixOption(prefix)
}

func (prefix prefixOption) applyRouter(router *Router) {
	router.prefix = string(prefix)
}

type tagsOption []string

// Tags adds OpenAPI tags to a Router or Route.
func Tags(tags ...string) TagsOption {
	return append(tagsOption(nil), tags...)
}

func (tags tagsOption) applyRouter(router *Router) {
	router.tags = append(router.tags, tags...)
}

func (tags tagsOption) applyRoute(route *route) {
	route.tags = append(route.tags, tags...)
}

type statusOption int

// Status sets the successful HTTP status returned by a route. It must be in
// the 200-299 range. Statuses 204 and 205 suppress response encoding and the
// body.
func Status(status int) RouteOption {
	return statusOption(status)
}

func (status statusOption) applyRoute(route *route) {
	route.output.Status = int(status)
}
