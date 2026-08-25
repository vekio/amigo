package amigo

import (
	"net/http"
	"slices"
)

type routerContext struct {
	prefix     string
	tags       []string
	middleware []Middleware
}

func (api *API) build() {
	api.buildOnce.Do(func() {
		api.built = true
		defer func() {
			api.buildFailure = recover()
		}()

		mux := http.NewServeMux()
		operations := make([]*Operation, 0)
		api.buildRouter(
			api.root,
			routerContext{},
			make(map[*Router]bool),
			mux,
			&operations,
		)
		api.mux = mux
		api.operations = operations
	})

	if api.buildFailure != nil {
		panic(api.buildFailure)
	}
}

func (api *API) buildRouter(
	router *Router,
	parent routerContext,
	visiting map[*Router]bool,
	mux *http.ServeMux,
	operations *[]*Operation,
) {
	if visiting[router] {
		panic("amigo: router inclusion cycle detected")
	}
	visiting[router] = true
	defer delete(visiting, router)

	router.frozen = true
	current := routerContext{
		prefix:     joinPath(parent.prefix, router.prefix),
		tags:       slices.Concat(parent.tags, router.tags),
		middleware: slices.Concat(parent.middleware, router.middleware),
	}

	for _, route := range router.routes {
		input := route.Input.clone()
		output := route.Output.clone()
		operationPath := joinPath(current.prefix, route.Path)
		if err := validateRules(input, api.validators); err != nil {
			panic("amigo: " + route.Method + " " + operationPath + ": " + err.Error())
		}
		handler := route.buildHandler(input, api.validators, api.errorHandler)
		operation := &Operation{
			Method:  route.Method,
			Path:    operationPath,
			Tags:    slices.Concat(current.tags, route.Tags),
			Input:   input,
			Output:  output,
			handler: applyMiddleware(handler, current.middleware),
		}

		*operations = append(*operations, operation)
		mux.Handle(operation.Method+" "+operation.Path, operation.handler)
	}

	for _, child := range router.routers {
		api.buildRouter(child, current, visiting, mux, operations)
	}
}
