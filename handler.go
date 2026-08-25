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
	inputMetadata InputMetadata,
	outputMetadata OutputMetadata,
	config handlerConfig,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if inputMetadata.Body != nil && config.maxBodyBytes > 0 && req.Body != nil {
			req.Body = http.MaxBytesReader(w, req.Body, config.maxBodyBytes)
		}

		input, err := bindInput[In](req, &inputMetadata)
		if err != nil {
			config.errorHandler(w, req, ErrorPhaseBinding, err)
			return
		}

		if err := validateInput(input, &inputMetadata, config.validators); err != nil {
			config.errorHandler(w, req, ErrorPhaseValidation, err)
			return
		}

		output, err := handler(req.Context(), input)
		if err != nil {
			phase := ErrorPhaseHandler
			if _, ok := errors.AsType[*ValidationError](err); ok {
				phase = ErrorPhaseValidation
			}
			config.errorHandler(w, req, phase, err)
			return
		}
		if outputMetadata.Status == http.StatusNoContent || outputMetadata.Status == http.StatusResetContent {
			w.WriteHeader(outputMetadata.Status)
			return
		}

		data, err := json.Marshal(output)
		if err != nil {
			config.errorHandler(w, req, ErrorPhaseResponseEncoding, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(outputMetadata.Status)
		_, _ = w.Write(data)
	})
}
