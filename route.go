package amigo

import "net/http"

const defaultMaxBodyBytes int64 = 1 << 20

type route struct {
	method        string
	path          string
	status        int
	maxBodyBytes  int64
	errorMappings []errorMapping
	middlewares   []Middleware
}

func newRoute(method string, path string, options ...RouteOption) route {
	r := route{
		method:       method,
		path:         path,
		status:       http.StatusOK,
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
