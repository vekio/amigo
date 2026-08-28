package amigo

import (
	"errors"
	"reflect"
	"testing"
)

func TestValidatorRegistersTypedValidator(t *testing.T) {
	api := New()
	wantErr := errors.New("must be an adult")

	api.Validator("adult", func(age int) error {
		if age < 18 {
			return wantErr
		}
		return nil
	})

	validator, exists := api.validators["adult"]
	if !exists {
		t.Fatal("adult validator was not registered")
	}
	if validator.typeOf != reflect.TypeFor[int]() {
		t.Errorf("type = %s, want %s", validator.typeOf, reflect.TypeFor[int]())
	}
	if validator.name != "adult" {
		t.Errorf("name = %q, want %q", validator.name, "adult")
	}
	if err := validator.validate(reflect.ValueOf(17), true); !errors.Is(err, wantErr) {
		t.Errorf("validate() error = %v, want %v", err, wantErr)
	}
	if err := validator.validate(reflect.ValueOf(18), true); err != nil {
		t.Errorf("validate() error = %v, want nil", err)
	}
}

func TestValidatorOnlyAdaptsTypedFunction(t *testing.T) {
	api := New()
	called := false
	api.Validator("adult", func(int) error {
		called = true
		return nil
	})

	if err := api.validators["adult"].validate(reflect.ValueOf(0), false); err != nil {
		t.Errorf("validate() error = %v, want nil", err)
	}
	if !called {
		t.Error("custom validator was not called")
	}
}

func TestRequiredValidatorIsRegisteredByDefault(t *testing.T) {
	api := New()
	validator, exists := api.validators[requiredValidator]
	if !exists {
		t.Fatal("required validator was not registered")
	}
	if validator.name != requiredValidator || validator.typeOf != nil {
		t.Errorf("validator = %#v", validator)
	}

	if err := validator.validate(reflect.ValueOf(0), false); err == nil {
		t.Error("missing value passed required validation")
	}
	if err := validator.validate(reflect.ValueOf(0), true); err != nil {
		t.Errorf("present zero value failed required validation: %v", err)
	}
}

func TestHasRequiredValue(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		present bool
		want    bool
	}{
		{name: "missing", value: 1, want: false},
		{name: "invalid reflection value", value: nil, present: true, want: false},
		{name: "zero integer", value: 0, present: true, want: true},
		{name: "negative integer", value: -1, present: true, want: true},
		{name: "zero float", value: 0.0, present: true, want: true},
		{name: "negative float", value: -1.5, present: true, want: true},
		{name: "false", value: false, present: true, want: true},
		{name: "blank string", value: "  ", present: true, want: false},
		{name: "string", value: "value", present: true, want: true},
		{name: "empty slice", value: []string{}, present: true, want: false},
		{name: "slice", value: []string{"value"}, present: true, want: true},
		{name: "nil pointer", value: (*int)(nil), present: true, want: false},
		{name: "pointer", value: new(int), present: true, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasRequiredValue(reflect.ValueOf(test.value), test.present); got != test.want {
				t.Errorf("hasRequiredValue() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestValidatorRejectsInvalidRegistration(t *testing.T) {
	tests := []struct {
		name     string
		register func(*API)
	}{
		{name: "empty name", register: func(api *API) { api.Validator("", func(int) error { return nil }) }},
		{name: "reserved name", register: func(api *API) { api.Validator("-", func(int) error { return nil }) }},
		{name: "whitespace", register: func(api *API) { api.Validator("adult validator", func(int) error { return nil }) }},
		{name: "comma", register: func(api *API) { api.Validator("adult,min", func(int) error { return nil }) }},
		{name: "required", register: func(api *API) { api.Validator("required", func(int) error { return nil }) }},
		{name: "nil function", register: func(api *API) {
			var validator func(int) error
			api.Validator("adult", validator)
		}},
		{name: "duplicate", register: func(api *API) {
			api.Validator("adult", func(int) error { return nil })
			api.Validator("adult", func(int) error { return nil })
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertPanics(t, func() { test.register(New()) })
		})
	}
}
