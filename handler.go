package amigo

import (
	"context"
	"log/slog"
	"net/http"
)

// EndpointFunc is a typed HTTP endpoint. In and Out must be structs. Input
// fields tagged with path, query, or header are bound from their corresponding
// request values; the remaining input is decoded from JSON. Output fields
// tagged with header become response headers; the remaining output is encoded
// as JSON. Transport fields must also use json:"-" so metadata cannot leak into
// JSON representations. Input fields may use validate:"required,name" to apply
// the built-in presence check and validators registered on the API. Query
// slices collect repeated keys; scalar query fields reject repeated keys.
// Parameter types implementing encoding.TextUnmarshaler are also supported;
// this includes time.Time (RFC 3339) and uuid.UUID (RFC 9562 text form).
type EndpointFunc[In, Out any] func(context.Context, In) (Out, error)

// RawEndpointFunc is the escape hatch for endpoints that need direct access to
// net/http, such as streaming responses or file downloads. An error should be
// returned before writing a response, because a committed response cannot be
// replaced with an error representation.
type RawEndpointFunc func(http.ResponseWriter, *http.Request) error

func endpointHandler[In, Out any](
	logger *slog.Logger,
	route route,
	inputMetadata inputMetadata,
	outputMetadata outputMetadata,
	endpoint EndpointFunc[In, Out],
) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		limitRequestBody(w, request, route.maxBodyBytes)

		bound, err := bindInput[In](request, inputMetadata)
		if err != nil {
			writeError(logger, w, request, route, err)
			return
		}
		if err := validateInput(bound.value, inputMetadata, bound.present); err != nil {
			writeError(logger, w, request, route, err)
			return
		}

		output, err := endpoint(request.Context(), bound.value)
		if err != nil {
			writeError(logger, w, request, route, err)
			return
		}

		if err := writeOutput(w, route.status, output, outputMetadata); err != nil {
			writeError(logger, w, request, route, err)
		}
	}
}

func rawEndpointHandler(logger *slog.Logger, route route, endpoint RawEndpointFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		limitRequestBody(w, request, route.maxBodyBytes)
		if err := endpoint(w, request); err != nil {
			writeError(logger, w, request, route, err)
		}
	}
}
