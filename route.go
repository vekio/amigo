package amigo

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// route describes an HTTP endpoint relative to its Router.
type route struct {
	Method string
	Path   string
	Tags   []string
	Input  *InputMetadata
	Output *OutputMetadata

	handlerFactory func(*InputMetadata, ErrorHandler) http.Handler
}

func register[In, Out any](router *Router, method, path string, handler Handler[In, Out], options ...RouteOption) {
	router.assertMutable()
	if handler == nil {
		panic(fmt.Sprintf("amigo: %s %s: handler is nil", method, path))
	}

	input, err := inspectInput(reflect.TypeFor[In]())
	if err != nil {
		panic(fmt.Sprintf("amigo: %s %s: %v", method, path, err))
	}

	registered := &route{
		Method: method,
		Path:   path,
		Input:  input,
		Output: inspectOutput(reflect.TypeFor[Out]()),
		handlerFactory: func(metadata *InputMetadata, errorHandler ErrorHandler) http.Handler {
			return handlerHTTP(handler, metadata, errorHandler)
		},
	}
	for _, option := range options {
		option.applyRoute(registered)
	}
	router.addRoute(registered)
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
