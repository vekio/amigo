package amigo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHandlerReportsErrorPhases(t *testing.T) {
	tests := []struct {
		name    string
		phase   ErrorPhase
		setup   func(*API)
		method  string
		target  string
		body    string
		headers map[string]string
	}{
		{
			name:  "binding",
			phase: ErrorPhaseBinding,
			setup: func(api *API) {
				api.GET("/items/{id}", func(_ context.Context, _ struct {
					ID int `path:"id"`
				}) (struct{}, error) {
					return struct{}{}, nil
				})
			},
			method: http.MethodGet,
			target: "/items/not-an-int",
		},
		{
			name:  "automatic validation",
			phase: ErrorPhaseValidation,
			setup: func(api *API) {
				api.Validator("positive", func(value int) error {
					if value < 1 {
						return errors.New("must be positive")
					}
					return nil
				})
				api.GET("/items", func(_ context.Context, _ struct {
					Page int `query:"page" validate:"positive"`
				}) (struct{}, error) {
					return struct{}{}, nil
				})
			},
			method: http.MethodGet,
			target: "/items?page=0",
		},
		{
			name:  "handler validation",
			phase: ErrorPhaseValidation,
			setup: func(api *API) {
				api.GET("/items", func(_ context.Context, _ emptyInput) (struct{}, error) {
					return struct{}{}, &ValidationError{}
				})
			},
			method: http.MethodGet,
			target: "/items",
		},
		{
			name:  "handler",
			phase: ErrorPhaseHandler,
			setup: func(api *API) {
				api.GET("/items", func(_ context.Context, _ emptyInput) (struct{}, error) {
					return struct{}{}, errors.New("handler failed")
				})
			},
			method: http.MethodGet,
			target: "/items",
		},
		{
			name:  "response encoding",
			phase: ErrorPhaseResponseEncoding,
			setup: func(api *API) {
				api.GET("/items", func(_ context.Context, _ emptyInput) (struct {
					Unsupported func() `json:"unsupported"`
				}, error) {
					return struct {
						Unsupported func() `json:"unsupported"`
					}{Unsupported: func() {}}, nil
				})
			},
			method: http.MethodGet,
			target: "/items",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := New()
			var gotPhase ErrorPhase
			var gotError error
			api.SetErrorHandler(func(_ http.ResponseWriter, _ *http.Request, phase ErrorPhase, err error) {
				gotPhase = phase
				gotError = err
			})
			test.setup(api)
			var body io.Reader
			if test.body != "" {
				body = strings.NewReader(test.body)
			}
			request(t, api, test.method, test.target, body, test.headers)
			if gotPhase != test.phase || gotError == nil {
				t.Fatalf("phase=%s error=%v", gotPhase, gotError)
			}
		})
	}
}

func TestErrorPhaseString(t *testing.T) {
	tests := map[ErrorPhase]string{
		ErrorPhaseUnknown:          "unknown",
		ErrorPhaseBinding:          "binding",
		ErrorPhaseValidation:       "validation",
		ErrorPhaseHandler:          "handler",
		ErrorPhaseResponseEncoding: "response_encoding",
		ErrorPhase(255):            "unknown",
	}
	for phase, want := range tests {
		if got := phase.String(); got != want {
			t.Fatalf("phase %d string=%q, want %q", phase, got, want)
		}
	}
}
