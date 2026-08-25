package amigo

import (
	"fmt"
	"reflect"
	"strings"
)

type validatorRegistry map[string]fieldValidator

type fieldValidator struct {
	typeOf   reflect.Type
	validate func(reflect.Value) error
}

// Validator registers a named typed validator for validate struct tags. It
// must be called before the API is built. The function must be safe for
// concurrent use.
func (api *API) Validator[T any](name string, validate func(T) error) {
	if api.built {
		panic("amigo: cannot register a validator after the API has been built")
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "-" || strings.Contains(name, ",") {
		panic(fmt.Sprintf("amigo: invalid validator name %q", name))
	}
	if validate == nil {
		panic(fmt.Sprintf("amigo: validator %q is nil", name))
	}
	if _, exists := api.validators[name]; exists {
		panic(fmt.Sprintf("amigo: validator %q is already registered", name))
	}
	if api.validators == nil {
		api.validators = make(validatorRegistry)
	}

	api.validators[name] = fieldValidator{
		typeOf: reflect.TypeFor[T](),
		validate: func(value reflect.Value) error {
			return validate(value.Interface().(T))
		},
	}
}

func validateRules(metadata *InputMetadata, validators validatorRegistry) error {
	for _, validation := range metadata.Validations {
		for _, rule := range validation.Rules {
			validator, exists := validators[rule]
			if !exists {
				return fmt.Errorf("validator %q used by %s is not registered", rule, validation.Location)
			}
			if !validation.Type.AssignableTo(validator.typeOf) {
				return fmt.Errorf(
					"validator %q expects %s but %s has type %s",
					rule,
					validator.typeOf,
					validation.Location,
					validation.Type,
				)
			}
		}
	}
	return nil
}

func validateInput(input any, metadata *InputMetadata, validators validatorRegistry) error {
	inputValue := reflect.ValueOf(input)
	validationError := &ValidationError{}

	for _, validation := range metadata.Validations {
		fieldValue, present := fieldByIndex(inputValue, validation.index)
		if !present {
			continue
		}
		for _, rule := range validation.Rules {
			if err := validators[rule].validate(fieldValue); err != nil {
				validationError.Errors = append(validationError.Errors, FieldError{
					Location: validation.Location,
					Message:  err.Error(),
				})
			}
		}
	}

	if len(validationError.Errors) > 0 {
		return validationError
	}
	return nil
}

func fieldByIndex(value reflect.Value, index []int) (reflect.Value, bool) {
	for _, fieldIndex := range index {
		for value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return reflect.Value{}, false
			}
			value = value.Elem()
		}
		value = value.Field(fieldIndex)
	}
	return value, true
}
