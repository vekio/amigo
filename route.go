package amigo

const defaultMaxBodyBytes int64 = 1 << 20

type route struct {
	operation
	maxBodyBytes int64
	middlewares  []Middleware
}

func newRoute(method string, path string, options ...RouteOption) route {
	r := route{
		operation:    newOperation(method, path),
		maxBodyBytes: defaultMaxBodyBytes,
	}

	for _, option := range options {
		if option == nil {
			panic("amigo: route option cannot be nil")
		}
		option(&r)
	}

	return r
}

func (r route) pattern() string {
	return r.method + " " + r.path
}
