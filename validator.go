package amigo

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

const requiredValidator = "required"

var errRequired = errors.New("is required")

type validatorRegistry map[string]fieldValidator

type fieldValidator struct {
	name     string
	typeOf   reflect.Type
	validate func(reflect.Value, bool) error
}

func newValidatorRegistry() validatorRegistry {
	return validatorRegistry{
		requiredValidator: {
			name: requiredValidator,
			validate: func(value reflect.Value, present bool) error {
				if hasRequiredValue(value, present) {
					return nil
				}
				return errRequired
			},
		},
	}
}

// Validator registers a named typed validator for validate struct tags. It
// must be called before routes that use it are registered. The validator
// function may be called concurrently and must therefore be concurrency-safe.
func (app *API) Validator[T any](name string, validate func(T) error) {
	checkValidatorRegistration(name, validate != nil, app.validators)

	app.validators[name] = fieldValidator{
		name:   name,
		typeOf: reflect.TypeFor[T](),
		validate: func(value reflect.Value, _ bool) error {
			return validate(value.Interface().(T))
		},
	}
}

func checkValidatorRegistration(name string, nonNil bool, validators validatorRegistry) {
	if name == "" || name == "-" || strings.Contains(name, ",") || strings.ContainsFunc(name, unicode.IsSpace) {
		panic(fmt.Sprintf("amigo: invalid validator name %q", name))
	}
	if !nonNil {
		panic(fmt.Sprintf("amigo: validator %q cannot be nil", name))
	}
	if _, exists := validators[name]; exists {
		panic(fmt.Sprintf("amigo: validator %q is already registered", name))
	}
}

func hasRequiredValue(value reflect.Value, present bool) bool {
	if !present || !value.IsValid() {
		return false
	}

	switch value.Kind() {
	case reflect.String:
		return strings.TrimSpace(value.String()) != ""
	case reflect.Array, reflect.Slice, reflect.Map:
		return value.Len() > 0
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Pointer:
		return !value.IsNil()
	default:
		return true
	}
}
