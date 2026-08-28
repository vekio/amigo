package amigo

import (
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
)

type errorMapping struct {
	target       error
	status       int
	publicDetail string
}

// resolveProblem converts an endpoint error into a safe client representation.
// Explicit problems win, followed by route mappings; everything else is a 500.
func (r route) resolveProblem(err error) *problem {
	if direct, ok := errors.AsType[*problem](err); ok {
		clone := *direct
		return &clone
	}
	if invalid, ok := errors.AsType[*validationError](err); ok {
		problem := newProblem(http.StatusUnprocessableEntity, invalid.Error())
		problem.Errors = invalid.errors
		return problem
	}

	for _, mapping := range r.errorMappings {
		if errors.Is(err, mapping.target) {
			return newProblem(mapping.status, mapping.publicDetail)
		}
	}

	return newProblem(http.StatusInternalServerError, "internal server error")
}

func writeError(logger *slog.Logger, w http.ResponseWriter, request *http.Request, route route, err error) {
	problem := route.resolveProblem(err)
	problem.Instance = request.URL.Path

	if problem.Status >= http.StatusInternalServerError {
		logger.ErrorContext(request.Context(), "request failed",
			"method", request.Method,
			"path", request.URL.Path,
			"error", err,
		)
	}

	writeProblem(w, problem)
}

func writeProblem(w http.ResponseWriter, problem *problem) {
	data, err := json.Marshal(problem)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	_, _ = w.Write(data)
}
