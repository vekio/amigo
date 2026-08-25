package amigo

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestValidatorsAccumulateErrorsAndSkipHandler(t *testing.T) {
	type address struct {
		City string `json:"city" validate:"required"`
	}
	type body struct {
		Age     int     `json:"age" validate:"adult"`
		Name    string  `json:"name" validate:"required,long"`
		Address address `json:"address"`
	}
	type input struct {
		Page int `query:"page" validate:"positive"`
		Body body
	}

	called := false
	api := New()
	api.Validator("positive", func(value int) error {
		if value < 1 {
			return errors.New("must be greater than 0")
		}
		return nil
	})
	api.Validator("adult", func(value int) error {
		if value < 18 {
			return errors.New("must be at least 18")
		}
		return nil
	})
	api.Validator("required", func(value string) error {
		if value == "" {
			return errors.New("value is required")
		}
		return nil
	})
	api.Validator("long", func(value string) error {
		if len(value) < 3 {
			return errors.New("must contain at least 3 characters")
		}
		return nil
	})
	api.POST("/users", func(_ context.Context, _ input) (struct{}, error) {
		called = true
		return struct{}{}, nil
	})

	response := request(
		t,
		api,
		http.MethodPost,
		"/users?page=0",
		strings.NewReader(`{"age":17,"name":"","address":{"city":""}}`),
		map[string]string{"Content-Type": "application/json"},
	)
	problem := decodeJSON[Problem](t, response)

	if response.Code != http.StatusBadRequest || called {
		t.Fatalf("status=%d, handler called=%t", response.Code, called)
	}
	want := []FieldError{
		{Location: "query.page", Message: "must be greater than 0"},
		{Location: "body.age", Message: "must be at least 18"},
		{Location: "body.name", Message: "value is required"},
		{Location: "body.name", Message: "must contain at least 3 characters"},
		{Location: "body.address.city", Message: "value is required"},
	}
	if !reflect.DeepEqual(problem.Errors, want) {
		t.Fatalf("errors = %#v, want %#v", problem.Errors, want)
	}
}

func TestValidationSkipsFieldsBelowNilPointer(t *testing.T) {
	type address struct {
		City string `json:"city" validate:"required"`
	}
	type input struct {
		Body struct {
			Address *address `json:"address"`
		}
	}

	api := New()
	api.Validator("required", func(value string) error {
		if value == "" {
			return errors.New("value is required")
		}
		return nil
	})
	api.POST("/users", func(_ context.Context, _ input) (struct{}, error) {
		return struct{}{}, nil
	})
	response := request(
		t,
		api,
		http.MethodPost,
		"/users",
		strings.NewReader(`{}`),
		map[string]string{"Content-Type": "application/json"},
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", response.Code, response.Body.String())
	}
}

func TestValidatorConfigurationFailures(t *testing.T) {
	t.Run("unknown rule", func(t *testing.T) {
		type input struct {
			Page int `query:"page" validate:"missing"`
		}
		api := New()
		api.GET("/items", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
		requirePanicContains(t, `validator "missing" used by query.page is not registered`, func() {
			api.Handler()
		})
		requirePanicContains(t, `validator "missing"`, func() {
			api.Handler()
		})
	})

	t.Run("incompatible type", func(t *testing.T) {
		type input struct {
			Age string `query:"age" validate:"adult"`
		}
		api := New()
		api.Validator("adult", func(int) error { return nil })
		api.GET("/items", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
		requirePanicContains(t, `validator "adult" expects int but query.age has type string`, func() {
			api.Handler()
		})
	})

	t.Run("duplicate name", func(t *testing.T) {
		api := New()
		api.Validator("rule", func(int) error { return nil })
		requirePanicContains(t, `validator "rule" is already registered`, func() {
			api.Validator("rule", func(int) error { return nil })
		})
	})

	t.Run("invalid name", func(t *testing.T) {
		for _, name := range []string{"", "-", "one,two"} {
			requirePanicContains(t, "invalid validator name", func() {
				New().Validator(name, func(int) error { return nil })
			})
		}
	})

	t.Run("nil function", func(t *testing.T) {
		var validator func(int) error
		requirePanicContains(t, `validator "rule" is nil`, func() {
			New().Validator("rule", validator)
		})
	})

	t.Run("registration after build", func(t *testing.T) {
		api := New()
		api.Handler()
		requirePanicContains(t, "cannot register a validator after the API has been built", func() {
			api.Validator("rule", func(int) error { return nil })
		})
	})
}

func TestValidationMetadataUsesRequestLocations(t *testing.T) {
	type input struct {
		ID   int `path:"id" validate:"positive"`
		Body struct {
			Email string `json:"email" validate:"required"`
			Skip  string `json:"-" validate:"ignored"`
		}
	}

	api := New()
	api.Validator("positive", func(int) error { return nil })
	api.Validator("required", func(string) error { return nil })
	api.GET("/items/{id}", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
	operation := api.Operations()[0]

	if len(operation.Input.Validations) != 2 {
		t.Fatalf("validations = %#v", operation.Input.Validations)
	}
	if operation.Input.Validations[0].Location != "path.id" || operation.Input.Validations[1].Location != "body.email" {
		t.Fatalf("locations = %#v", operation.Input.Validations)
	}
}
