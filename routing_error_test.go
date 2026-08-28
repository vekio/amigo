package amigo

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIReturnsProblemForUnknownRoute(t *testing.T) {
	api := New()
	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing", nil))

	assertRoutingProblem(t, response, http.StatusNotFound, "resource not found", "/missing")
}

func TestAPIReturnsProblemForUnsupportedMethod(t *testing.T) {
	api := New()
	api.GET("/things", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})
	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/things", nil))

	assertRoutingProblem(t, response, http.StatusMethodNotAllowed, "method not allowed", "/things")
	if allow := response.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Errorf("Allow = %q, want %q", allow, "GET, HEAD")
	}
}

func TestAPIPreservesServeMuxRedirects(t *testing.T) {
	api := New()
	api.GET("/things/", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})
	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/things", nil))

	if response.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", response.Code, http.StatusTemporaryRedirect)
	}
	if location := response.Header().Get("Location"); location != "/things/" {
		t.Errorf("Location = %q, want %q", location, "/things/")
	}
}

func assertRoutingProblem(
	t *testing.T,
	response *httptest.ResponseRecorder,
	status int,
	detail string,
	instance string,
) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", contentType)
	}

	var got problem
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if got.Status != status || got.Detail != detail || got.Instance != instance {
		t.Errorf("problem = %#v", got)
	}
}
