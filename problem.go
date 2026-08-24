package amigo

import (
	"encoding/json/v2"
	"net/http"
)

// StatusError is an error whose status and message are safe to expose.
type StatusError interface {
	error
	StatusCode() int
}

// Problem describes an HTTP API error following RFC 9457.
type Problem struct {
	Type     string       `json:"type,omitempty"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`

	err error
}

// FieldError describes one invalid request value.
type FieldError struct {
	Location string `json:"location"`
	Message  string `json:"message"`
}

// ValidationError contains all invalid request values found during binding.
type ValidationError struct {
	Errors []FieldError
}

func (validation *ValidationError) Error() string {
	return "request parameters are invalid"
}

func (validation *ValidationError) StatusCode() int {
	return http.StatusBadRequest
}

// NewProblem creates a problem using the standard HTTP status title.
func NewProblem(status int, detail string) *Problem {
	return &Problem{
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
}

// WrapProblem attaches a private cause to a public problem.
func WrapProblem(err error, problem *Problem) *Problem {
	if problem == nil {
		problem = InternalServerError("internal server error")
	}

	wrapped := *problem
	wrapped.err = err
	return &wrapped
}

// BadRequest creates a 400 problem.
func BadRequest(detail string) *Problem {
	return NewProblem(http.StatusBadRequest, detail)
}

// Unauthorized creates a 401 problem.
func Unauthorized(detail string) *Problem {
	return NewProblem(http.StatusUnauthorized, detail)
}

// Forbidden creates a 403 problem.
func Forbidden(detail string) *Problem {
	return NewProblem(http.StatusForbidden, detail)
}

// NotFound creates a 404 problem.
func NotFound(detail string) *Problem {
	return NewProblem(http.StatusNotFound, detail)
}

// Conflict creates a 409 problem.
func Conflict(detail string) *Problem {
	return NewProblem(http.StatusConflict, detail)
}

// UnprocessableEntity creates a 422 problem.
func UnprocessableEntity(detail string) *Problem {
	return NewProblem(http.StatusUnprocessableEntity, detail)
}

// InternalServerError creates a 500 problem.
func InternalServerError(detail string) *Problem {
	return NewProblem(http.StatusInternalServerError, detail)
}

func problemFromValidation(validation *ValidationError) *Problem {
	problem := BadRequest("request parameters are invalid")
	if validation != nil {
		problem.Errors = append([]FieldError(nil), validation.Errors...)
	}
	return problem
}

func (problem *Problem) Error() string {
	if problem == nil {
		return http.StatusText(http.StatusInternalServerError)
	}
	if problem.Detail != "" {
		return problem.Detail
	}
	return problem.Title
}

func (problem *Problem) StatusCode() int {
	if problem == nil {
		return http.StatusInternalServerError
	}
	return problem.Status
}

// Unwrap returns the private cause of the problem.
func (problem *Problem) Unwrap() error {
	if problem == nil {
		return nil
	}
	return problem.err
}

func writeProblem(w http.ResponseWriter, problem *Problem) {
	if problem == nil || problem.Status < 400 || problem.Status > 599 {
		instance := ""
		if problem != nil {
			instance = problem.Instance
		}
		problem = InternalServerError("internal server error")
		problem.Instance = instance
	} else {
		copy := *problem
		problem = &copy
	}
	if problem.Title == "" {
		problem.Title = http.StatusText(problem.Status)
	}

	data, _ := json.Marshal(problem)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	_, _ = w.Write(data)
}
