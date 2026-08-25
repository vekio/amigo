package amigo

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// route describes an HTTP endpoint relative to its Router.
type route struct {
	method string
	path   string
	tags   []string
	input  InputMetadata
	output OutputMetadata

	buildHandler handlerBuilder
	middleware   []Middleware
}

type handlerConfig struct {
	validators   validatorRegistry
	errorHandler ErrorHandler
	maxBodyBytes int64
}

type handlerBuilder func(InputMetadata, OutputMetadata, handlerConfig) http.Handler

func register[In, Out any](router *Router, method, path string, handler Handler[In, Out], options ...RouteOption) {
	if handler == nil {
		panic(fmt.Sprintf("amigo: %s %s: handler is nil", method, path))
	}

	input, err := inspectInput(reflect.TypeFor[In]())
	if err != nil {
		panic(fmt.Sprintf("amigo: %s %s: %v", method, path, err))
	}

	registered := route{
		method: method,
		path:   path,
		input:  input,
		output: inspectOutput(reflect.TypeFor[Out]()),
		buildHandler: func(
			input InputMetadata,
			output OutputMetadata,
			config handlerConfig,
		) http.Handler {
			return handlerHTTP(handler, input, output, config)
		},
	}
	for _, option := range options {
		option.applyRoute(&registered)
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
