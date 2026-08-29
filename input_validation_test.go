package amigo

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestEndpointValidationAggregatesFieldErrors(t *testing.T) {
	api := New()
	api.Validator("adult", func(age int) error {
		if age < 18 {
			return errors.New("must be at least 18")
		}
		return nil
	})

	called := false
	api.POST("/users", func(context.Context, struct {
		Age           int    `json:"age" validate:"required,adult"`
		Name          string `json:"name" validate:"required"`
		Authorization string `header:"Authorization" json:"-" validate:"required"`
		Enabled       bool   `query:"enabled" json:"-" validate:"required"`
	}) (struct{}, error) {
		called = true
		return struct{}{}, nil
	})

	request := httptest.NewRequest(http.MethodPost, "/users?enabled=false", strings.NewReader(`{"age":16}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	api.ServeHTTP(response, request)

	if called {
		t.Error("endpoint was called with invalid input")
	}
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", contentType)
	}

	var got problem
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	want := []fieldError{
		{Location: "body.age", Message: "must be at least 18"},
		{Location: "body.name", Message: "is required"},
		{Location: "header.Authorization", Message: "is required"},
	}
	if !reflect.DeepEqual(got.Errors, want) {
		t.Errorf("errors = %#v, want %#v", got.Errors, want)
	}
}

func TestRequiredUsesRequestPresenceForBodyValues(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "absent body", wantStatus: http.StatusUnprocessableEntity},
		{name: "absent property", body: `{}`, wantStatus: http.StatusUnprocessableEntity},
		{name: "null property", body: `{"count":null}`, wantStatus: http.StatusUnprocessableEntity},
		{name: "present zero", body: `{"count":0}`, wantStatus: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := New()
			api.POST("/counts", func(_ context.Context, input struct {
				Count int `json:"count" validate:"required"`
			}) (struct{}, error) {
				return struct{}{}, nil
			})

			var request *http.Request
			if test.body == "" {
				request = httptest.NewRequest(http.MethodPost, "/counts", nil)
			} else {
				request = httptest.NewRequest(http.MethodPost, "/counts", strings.NewReader(test.body))
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			api.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestOptionalValidatorSkipsMissingField(t *testing.T) {
	api := New()
	validatorCalled := false
	api.Validator("positive", func(value int) error {
		validatorCalled = true
		if value <= 0 {
			return errors.New("must be positive")
		}
		return nil
	})
	api.GET("/things", func(context.Context, struct {
		Limit int `query:"limit" json:"-" validate:"positive"`
	}) (struct{}, error) {
		return struct{}{}, nil
	})

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/things", nil))

	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if validatorCalled {
		t.Error("custom validator was called for a missing optional field")
	}
}

func TestRequiredRunsFirstAndStopsFieldValidation(t *testing.T) {
	tests := []struct {
		name                string
		target              string
		wantStatus          int
		wantValidatorCalled bool
	}{
		{name: "missing", target: "/things", wantStatus: http.StatusUnprocessableEntity},
		{name: "present but blank", target: "/things?name=+++", wantStatus: http.StatusUnprocessableEntity},
		{name: "present and valid", target: "/things?name=amigo", wantStatus: http.StatusOK, wantValidatorCalled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := New()
			validatorCalled := false
			api.Validator("known", func(string) error {
				validatorCalled = true
				return nil
			})
			api.GET("/things", func(context.Context, struct {
				Name string `query:"name" json:"-" validate:"known,required"`
			}) (struct{}, error) {
				return struct{}{}, nil
			})

			response := httptest.NewRecorder()
			api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.target, nil))

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if validatorCalled != test.wantValidatorCalled {
				t.Errorf("validator called = %t, want %t", validatorCalled, test.wantValidatorCalled)
			}
			if test.wantStatus == http.StatusUnprocessableEntity {
				var got problem
				if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
					t.Fatalf("decode problem: %v", err)
				}
				want := []fieldError{{Location: "query.name", Message: "is required"}}
				if !reflect.DeepEqual(got.Errors, want) {
					t.Errorf("errors = %#v, want %#v", got.Errors, want)
				}
			}
		})
	}
}

func TestQuerySliceSupportsRequiredAndCustomValidation(t *testing.T) {
	api := New()
	api.Validator("multiple", func(values []string) error {
		if len(values) < 2 {
			return errors.New("must contain at least two values")
		}
		return nil
	})
	var received []string
	api.GET("/things", func(_ context.Context, input struct {
		Tags []string `query:"tag" json:"-" validate:"required,multiple"`
	}) (struct{}, error) {
		received = input.Tags
		return struct{}{}, nil
	})

	tests := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{name: "missing", target: "/things", wantStatus: http.StatusUnprocessableEntity},
		{name: "one value", target: "/things?tag=go", wantStatus: http.StatusUnprocessableEntity},
		{name: "repeated values", target: "/things?tag=go&tag=http", wantStatus: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			received = nil
			response := httptest.NewRecorder()
			api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.target, nil))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantStatus == http.StatusOK {
				want := []string{"go", "http"}
				if !reflect.DeepEqual(received, want) {
					t.Errorf("received = %#v, want %#v", received, want)
				}
			}
		})
	}
}

func TestRouteRejectsInvalidValidationTags(t *testing.T) {
	tests := []struct {
		name     string
		register func(*API)
	}{
		{
			name: "no rules",
			register: func(api *API) {
				api.GET("/things", endpointFor[struct {
					Value string `query:"value" json:"-" validate:""`
				}])
			},
		},
		{
			name: "unbound field",
			register: func(api *API) {
				api.GET("/things", endpointFor[struct {
					Value string `json:"-" validate:"required"`
				}])
			},
		},
		{
			name: "invalid rule",
			register: func(api *API) {
				api.GET("/things", endpointFor[struct {
					Value string `query:"value" json:"-" validate:"required, adult"`
				}])
			},
		},
		{
			name: "duplicate rule",
			register: func(api *API) {
				api.GET("/things", endpointFor[struct {
					Value string `query:"value" json:"-" validate:"required,required"`
				}])
			},
		},
		{
			name: "unknown validator",
			register: func(api *API) {
				api.GET("/things", endpointFor[struct {
					Value int `query:"value" json:"-" validate:"positive"`
				}])
			},
		},
		{
			name: "wrong validator type",
			register: func(api *API) {
				api.Validator("positive", func(int) error { return nil })
				api.GET("/things", endpointFor[struct {
					Value string `query:"value" json:"-" validate:"positive"`
				}])
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertPanics(t, func() { test.register(New()) })
		})
	}
}

func endpointFor[In any](context.Context, In) (struct{}, error) {
	return struct{}{}, nil
}
