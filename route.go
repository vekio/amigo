package amigo

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// Route describes an HTTP endpoint relative to its Router.
type Route struct {
	Method string
	Path   string
	Tags   []string
	Input  *InputMetadata
	Output *OutputMetadata

	handlerFactory func(ErrorHandler) http.Handler
}

func get[In, Out any](router *Router, path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return register(router, http.MethodGet, path, handler, options...)
}

func post[In, Out any](router *Router, path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return register(router, http.MethodPost, path, handler, options...)
}

func put[In, Out any](router *Router, path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return register(router, http.MethodPut, path, handler, options...)
}

func patch[In, Out any](router *Router, path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return register(router, http.MethodPatch, path, handler, options...)
}

func deleteRoute[In, Out any](router *Router, path string, handler Handler[In, Out], options ...RouteOption) *Route {
	return register(router, http.MethodDelete, path, handler, options...)
}

func register[In, Out any](router *Router, method, path string, handler Handler[In, Out], options ...RouteOption) *Route {
	input, err := inspectInput(reflect.TypeFor[In]())
	if err != nil {
		panic(fmt.Sprintf("amigo: %s %s: %v", method, path, err))
	}

	route := &Route{
		Method: method,
		Path:   path,
		Input:  input,
		Output: inspectOutput(reflect.TypeFor[Out]()),
		handlerFactory: func(errorHandler ErrorHandler) http.Handler {
			return handlerHTTP(handler, input, errorHandler)
		},
	}
	for _, option := range options {
		option.applyRoute(route)
	}
	router.addRoute(route)
	return route
}

func joinPath(prefix, path string) string {
	if prefix == "" {
		return path
	}
	if path == "" {
		return prefix
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
}
