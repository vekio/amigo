package amigo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type testStatusError struct {
	status int
	detail string
}

func (err testStatusError) Error() string   { return err.detail }
func (err testStatusError) StatusCode() int { return err.status }

func TestProblemHelpers(t *testing.T) {
	tests := []struct {
		name   string
		status int
		new    func(string) *Problem
	}{
		{"bad request", http.StatusBadRequest, BadRequest},
		{"unauthorized", http.StatusUnauthorized, Unauthorized},
		{"forbidden", http.StatusForbidden, Forbidden},
		{"not found", http.StatusNotFound, NotFound},
		{"conflict", http.StatusConflict, Conflict},
		{"unprocessable entity", http.StatusUnprocessableEntity, UnprocessableEntity},
		{"internal server error", http.StatusInternalServerError, InternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			problem := test.new("detail")
			if problem.Status != test.status || problem.StatusCode() != test.status {
				t.Fatalf("status=%d statusCode=%d", problem.Status, problem.StatusCode())
			}
			if problem.Title != http.StatusText(test.status) || problem.Error() != "detail" {
				t.Fatalf("problem=%+v", problem)
			}
		})
	}
}

func TestWrapProblemPreservesCauseWithoutMutatingOriginal(t *testing.T) {
	cause := errors.New("database unavailable")
	original := Conflict("public conflict")
	wrapper := WrapProblem(cause, original)

	if wrapper == original {
		t.Fatal("WrapProblem returned the original pointer")
	}
	if !errors.Is(wrapper, cause) {
		t.Fatal("wrapped cause is not visible through errors.Is")
	}
	if original.Unwrap() != nil {
		t.Fatal("original problem was mutated")
	}
	if wrapper.Status != http.StatusConflict || wrapper.Detail != "public conflict" {
		t.Fatalf("wrapper=%+v", wrapper)
	}
}

func TestDefaultErrorHandlerResponses(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		phase      ErrorPhase
		status     int
		detail     string
		fieldError []FieldError
	}{
		{
			name:   "public problem",
			err:    Conflict("email already exists"),
			phase:  ErrorPhaseHandler,
			status: http.StatusConflict,
			detail: "email already exists",
		},
		{
			name:   "status error",
			err:    testStatusError{status: http.StatusNotFound, detail: "user not found"},
			phase:  ErrorPhaseHandler,
			status: http.StatusNotFound,
			detail: "user not found",
		},
		{
			name:   "validation error",
			err:    &ValidationError{Errors: []FieldError{{Location: "body.email", Message: "value is required"}}},
			phase:  ErrorPhaseValidation,
			status: http.StatusBadRequest,
			detail: "request parameters are invalid",
			fieldError: []FieldError{
				{Location: "body.email", Message: "value is required"},
			},
		},
		{
			name:   "private error",
			err:    errors.New("database password is secret"),
			phase:  ErrorPhaseHandler,
			status: http.StatusInternalServerError,
			detail: "internal server error",
		},
		{
			name:   "invalid status error",
			err:    testStatusError{status: 200, detail: "must not be public"},
			phase:  ErrorPhaseHandler,
			status: http.StatusInternalServerError,
			detail: "internal server error",
		},
		{
			name:   "nil error",
			phase:  ErrorPhaseHandler,
			status: http.StatusInternalServerError,
			detail: "internal server error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/items?token=secret", nil)
			response := httptest.NewRecorder()
			DefaultErrorHandler(response, req, test.phase, test.err)
			problem := decodeJSON[Problem](t, response)

			if response.Code != test.status || problem.Status != test.status || problem.Detail != test.detail {
				t.Fatalf("response=%d problem=%+v", response.Code, problem)
			}
			if response.Header().Get("Content-Type") != "application/problem+json" {
				t.Fatalf("Content-Type=%q", response.Header().Get("Content-Type"))
			}
			if problem.Instance != "/items" {
				t.Fatalf("instance=%q", problem.Instance)
			}
			if !reflect.DeepEqual(problem.Errors, test.fieldError) {
				t.Fatalf("errors=%#v, want %#v", problem.Errors, test.fieldError)
			}
			if strings.Contains(response.Body.String(), "database password is secret") {
				t.Fatal("private error leaked")
			}
		})
	}
}

func TestWriteProblemNormalizesInvalidProblem(t *testing.T) {
	response := httptest.NewRecorder()
	writeProblem(response, &Problem{Status: http.StatusOK, Detail: "must not leak", Instance: "/items"})
	problem := decodeJSON[Problem](t, response)

	if response.Code != http.StatusInternalServerError || problem.Detail != "internal server error" || problem.Instance != "/items" {
		t.Fatalf("problem=%+v status=%d", problem, response.Code)
	}
}

func TestValidationErrorImplementsStatusError(t *testing.T) {
	validation := &ValidationError{}
	if validation.Error() != "request parameters are invalid" || validation.StatusCode() != http.StatusBadRequest {
		t.Fatalf("validation error=%q status=%d", validation.Error(), validation.StatusCode())
	}
}

func TestSetErrorHandlerNilRestoresDefault(t *testing.T) {
	api := New()
	api.SetErrorHandler(func(http.ResponseWriter, *http.Request, ErrorPhase, error) {})
	api.SetErrorHandler(nil)
	api.GET("/items", func(_ context.Context, _ emptyInput) (struct{}, error) {
		return struct{}{}, Conflict("conflict")
	})
	response := request(t, api, http.MethodGet, "/items", nil, nil)
	if response.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
