package amigo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouteMapsHandlerError(t *testing.T) {
	target := errors.New("thing not found")
	api := New()
	api.GET("/things/{id}", func(context.Context, struct {
		ID string `path:"id" json:"-"`
	}) (struct{}, error) {
		return struct{}{}, errors.Join(errors.New("repository"), target)
	}, WithErrorMapping(target, http.StatusNotFound, "thing not found"))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/things/42", nil)
	api.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestRouteLimitsRequestBody(t *testing.T) {
	api := New()
	api.POST("/things", func(context.Context, struct {
		Name string `json:"name"`
	}) (struct{}, error) {
		return struct{}{}, nil
	}, WithMaxBodyBytes(4))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/things", strings.NewReader(`{"name":"shorty"}`))
	request.Header.Set("Content-Type", "application/json")
	api.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestRawEndpointOwnsResponse(t *testing.T) {
	api := New()
	api.RAW(http.MethodGet, "/download", func(w http.ResponseWriter, _ *http.Request) error {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("raw"))
		return nil
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/download", nil)
	api.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "raw" {
		t.Errorf("status = %d, body = %q", response.Code, response.Body.String())
	}
}

func TestRawEndpointMapsReturnedError(t *testing.T) {
	target := errors.New("download unavailable")
	api := New()
	api.RAW(http.MethodGet, "/download", func(http.ResponseWriter, *http.Request) error {
		return target
	}, WithErrorMapping(target, http.StatusServiceUnavailable, "download unavailable"))

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/download", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestTypedEndpointReportsResponseEncodingFailure(t *testing.T) {
	api := New()
	api.GET("/things", func(context.Context, struct{}) (struct {
		Unsupported func() `json:"unsupported"`
	}, error) {
		return struct {
			Unsupported func() `json:"unsupported"`
		}{Unsupported: func() {}}, nil
	})

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/things", nil))

	if response.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func TestRouteMiddlewareWrapsTypedAndRawEndpoints(t *testing.T) {
	markResponse := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, request)
		})
	}

	tests := []struct {
		name     string
		register func(*API)
	}{
		{
			name: "typed",
			register: func(api *API) {
				api.GET("/things", func(context.Context, struct{}) (struct{}, error) {
					return struct{}{}, nil
				}, WithMiddleware(markResponse))
			},
		},
		{
			name: "raw",
			register: func(api *API) {
				api.RAW(http.MethodGet, "/things", func(w http.ResponseWriter, _ *http.Request) error {
					w.WriteHeader(http.StatusNoContent)
					return nil
				}, WithMiddleware(markResponse))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := New()
			test.register(api)
			response := httptest.NewRecorder()
			api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/things", nil))

			if response.Header().Get("X-Middleware") != "applied" {
				t.Errorf("X-Middleware = %q", response.Header().Get("X-Middleware"))
			}
		})
	}
}
