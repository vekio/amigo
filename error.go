package amigo

import (
	"errors"
	"log/slog"
	"net/http"
)

// ErrorPhase identifies the stage where a request failed.
type ErrorPhase uint8

const (
	ErrorPhaseUnknown ErrorPhase = iota
	ErrorPhaseBinding
	ErrorPhaseValidation
	ErrorPhaseHandler
	ErrorPhaseResponseEncoding
)

func (phase ErrorPhase) String() string {
	switch phase {
	case ErrorPhaseBinding:
		return "binding"
	case ErrorPhaseValidation:
		return "validation"
	case ErrorPhaseHandler:
		return "handler"
	case ErrorPhaseResponseEncoding:
		return "response_encoding"
	default:
		return "unknown"
	}
}

// ErrorHandler handles an error produced while serving an HTTP request.
type ErrorHandler func(http.ResponseWriter, *http.Request, ErrorPhase, error)

// DefaultErrorHandler logs private errors and writes an RFC 9457 response.
func DefaultErrorHandler(
	w http.ResponseWriter,
	req *http.Request,
	phase ErrorPhase,
	err error,
) {
	if err == nil {
		err = errors.New("nil request error")
		phase = ErrorPhaseUnknown
	}

	var problem *Problem

	if validation, ok := errors.AsType[*ValidationError](err); ok {
		problem = problemFromValidation(validation)
	} else if publicProblem, ok := errors.AsType[*Problem](err); ok {
		problem = publicProblem
		if cause := errors.Unwrap(publicProblem); cause != nil {
			logRequestError(req, phase, cause)
		} else if publicProblem.Status >= http.StatusInternalServerError {
			logRequestError(req, phase, err)
		}
	} else if statusError, ok := errors.AsType[StatusError](err); ok {
		status := statusError.StatusCode()
		if status < 400 || status > 599 {
			logRequestError(req, phase, err)
			problem = InternalServerError("internal server error")
		} else {
			problem = NewProblem(status, statusError.Error())
		}
		if status >= http.StatusInternalServerError && status <= 599 {
			logRequestError(req, phase, err)
		}
	} else if phase == ErrorPhaseBinding {
		status := http.StatusBadRequest
		if errors.Is(err, errUnsupportedMediaType) {
			status = http.StatusUnsupportedMediaType
		}
		problem = NewProblem(status, err.Error())
	} else {
		logRequestError(req, phase, err)
		problem = InternalServerError("internal server error")
	}

	problem = problemWithInstance(problem, req.URL.Path)
	writeProblem(w, problem)
}

func problemWithInstance(problem *Problem, instance string) *Problem {
	if problem == nil {
		problem = InternalServerError("internal server error")
	}

	copy := *problem
	if copy.Instance == "" {
		copy.Instance = instance
	}
	return &copy
}

func logRequestError(req *http.Request, phase ErrorPhase, err error) {
	slog.ErrorContext(
		req.Context(),
		"request failed",
		"phase", phase.String(),
		"method", req.Method,
		"path", req.URL.Path,
		"error", err,
	)
}
