package amigo

import (
	"fmt"
	"net/http"
)

func (api *API) compile() (mux *http.ServeMux, operations []Operation, err error) {
	defer func() {
		if value := recover(); value != nil {
			mux = nil
			operations = nil
			err = fmt.Errorf("amigo: %v", value)
		}
	}()

	mux = http.NewServeMux()
	operations = make([]Operation, 0, len(api.root.routes))
	config := handlerConfig{
		validators:   api.validators,
		errorHandler: api.errorHandler,
		maxBodyBytes: api.maxBodyBytes,
	}

	for _, route := range api.root.routes {
		operation, err := route.toOperation(
			api.root.middleware,
			config,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("amigo: %s %s: %w", route.method, route.path, err)
		}
		operations = append(operations, operation)
		mux.Handle(operation.Method+" "+operation.Path, operation.handler)
	}

	for _, mount := range api.root.staticMounts {
		pattern, handler := mount.httpHandler(api.root.middleware)
		mux.Handle(pattern, handler)
	}

	return mux, operations, nil
}
