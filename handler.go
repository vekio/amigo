package amigo

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
)

// Handler processes a typed input and returns a typed output.
type Handler[In, Out any] func(context.Context, In) (Out, error)

func handlerHTTP[In, Out any](
	handler Handler[In, Out],
	metadata *InputMetadata,
	validators validatorRegistry,
	errorHandler ErrorHandler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		input, err := bindInput[In](req, metadata)
		if err != nil {
			errorHandler(w, req, ErrorPhaseBinding, err)
			return
		}

		if err := validateInput(input, metadata, validators); err != nil {
			errorHandler(w, req, ErrorPhaseValidation, err)
			return
		}

		output, err := handler(req.Context(), input)
		if err != nil {
			phase := ErrorPhaseHandler
			if _, ok := errors.AsType[*ValidationError](err); ok {
				phase = ErrorPhaseValidation
			}
			errorHandler(w, req, phase, err)
			return
		}

		data, err := json.Marshal(output)
		if err != nil {
			errorHandler(w, req, ErrorPhaseResponseEncoding, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})
}
