package amigo

import (
	"fmt"
	"net/http"
	"slices"
)

// Operation describes a complete route registered in an API.
type Operation struct {
	Method string
	Path   string
	Tags   []string
	Input  InputMetadata
	Output OutputMetadata

	handler http.Handler
}

func (operation Operation) clone() Operation {
	cloned := operation
	cloned.Tags = slices.Clone(operation.Tags)
	cloned.Input = operation.Input.clone()
	return cloned
}

func (route route) toOperation(
	middleware []Middleware,
	config handlerConfig,
) (Operation, error) {
	if err := validatePathParameters(route.path, route.input); err != nil {
		return Operation{}, err
	}
	if err := validateRules(&route.input, config.validators); err != nil {
		return Operation{}, err
	}
	if route.output.Status < http.StatusOK || route.output.Status > 299 {
		return Operation{}, fmt.Errorf("success status must be between 200 and 299, got %d", route.output.Status)
	}

	handler := route.buildHandler(route.input, route.output, config)
	return Operation{
		Method:  route.method,
		Path:    route.path,
		Tags:    route.tags,
		Input:   route.input,
		Output:  route.output,
		handler: applyMiddleware(handler, slices.Concat(middleware, route.middleware)),
	}, nil
}
