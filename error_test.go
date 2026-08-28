package amigo

import (
	"encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorReturnsSafeInternalServerError(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/things", nil)
	response := httptest.NewRecorder()

	writeError(discardLogger(), response, request, route{}, errors.New("private failure"))

	if response.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if response.Header().Get("Content-Type") != "application/problem+json" {
		t.Errorf("Content-Type = %q", response.Header().Get("Content-Type"))
	}
	var output problem
	if err := json.Unmarshal(response.Body.Bytes(), &output); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if output.Detail != "internal server error" || output.Instance != "/things" {
		t.Errorf("problem = %#v", output)
	}
}

func TestWriteErrorUsesRouteMapping(t *testing.T) {
	target := errors.New("repository sentinel")
	publicDetail := "thing not found"
	request := httptest.NewRequest(http.MethodGet, "/things/42", nil)
	response := httptest.NewRecorder()
	route := newRoute(http.MethodGet, "/things/{id}", WithErrorMapping(target, http.StatusNotFound, publicDetail))

	writeError(discardLogger(), response, request, route, errors.Join(errors.New("private database failure"), target))

	if response.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	var output problem
	if err := json.Unmarshal(response.Body.Bytes(), &output); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if output.Detail != publicDetail || output.Instance != "/things/42" {
		t.Errorf("problem = %#v", output)
	}
}

func TestErrorMappingAllowsOmittingPublicDetail(t *testing.T) {
	target := errors.New("private failure")
	route := newRoute(http.MethodGet, "/things", WithErrorMapping(target, http.StatusNotFound, ""))

	problem := route.resolveProblem(target)
	if problem.Status != http.StatusNotFound || problem.Detail != "" {
		t.Errorf("problem = %#v", problem)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
