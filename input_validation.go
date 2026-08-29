package amigo

import (
	"fmt"
	"reflect"
	"strings"
)

type fieldValidation struct {
	fieldID    int
	fieldIndex []int
	location   string
	required   *fieldValidator
	validators []fieldValidator
}

type fieldError struct {
	Location string `json:"location"`
	Message  string `json:"message"`
}

type validationError struct {
	errors []fieldError
}

func (err *validationError) Error() string {
	return "request validation failed"
}

func buildFieldValidation(
	field reflect.StructField,
	fieldID int,
	location inputLocation,
	validators validatorRegistry,
) (fieldValidation, bool) {
	tag, tagged := field.Tag.Lookup("validate")
	if !tagged {
		return fieldValidation{}, false
	}
	if tag == "" {
		panic(fmt.Sprintf("amigo: validation field %s has no rules", field.Name))
	}
	if !field.IsExported() {
		panic(fmt.Sprintf("amigo: validation field %s must be exported", field.Name))
	}

	if location.source == "" {
		panic(fmt.Sprintf("amigo: validation field %s is not bound from the request", field.Name))
	}

	validation := fieldValidation{
		fieldID:    fieldID,
		fieldIndex: field.Index,
		location:   location.label(),
	}
	seen := make(map[string]struct{})
	for ruleName := range strings.SplitSeq(tag, ",") {
		if ruleName == "" || strings.TrimSpace(ruleName) != ruleName {
			panic(fmt.Sprintf("amigo: validation field %s has an invalid rule %q", field.Name, ruleName))
		}
		if _, duplicate := seen[ruleName]; duplicate {
			panic(fmt.Sprintf("amigo: validation rule %q is repeated on field %s", ruleName, field.Name))
		}
		seen[ruleName] = struct{}{}

		validator, exists := validators[ruleName]
		if !exists {
			panic(fmt.Sprintf("amigo: validator %q is not registered", ruleName))
		}
		if validator.typeOf != nil && field.Type != validator.typeOf {
			panic(fmt.Sprintf(
				"amigo: validator %q expects %s but field %s has type %s",
				ruleName,
				validator.typeOf,
				field.Name,
				field.Type,
			))
		}
		if validator.name == requiredValidator {
			validation.required = &validator
			continue
		}
		validation.validators = append(validation.validators, validator)
	}
	return validation, true
}

func validateInput(input any, metadata inputMetadata, present fieldSet) error {
	inputValue := reflect.ValueOf(input)
	errors := make([]fieldError, 0)

	for _, validation := range metadata.validations {
		fieldValue := inputValue.FieldByIndex(validation.fieldIndex)
		fieldPresent := present.contains(validation.fieldID)

		if validation.required != nil {
			if err := validation.required.validate(fieldValue, fieldPresent); err != nil {
				errors = append(errors, newFieldError(validation.location, validation.required.name, err))
				continue
			}
		}
		if !fieldPresent {
			continue
		}
		for _, validator := range validation.validators {
			if err := validator.validate(fieldValue, fieldPresent); err != nil {
				errors = append(errors, newFieldError(validation.location, validator.name, err))
			}
		}
	}

	if len(errors) == 0 {
		return nil
	}
	return &validationError{errors: errors}
}

func newFieldError(location string, validatorName string, err error) fieldError {
	message := err.Error()
	if message == "" {
		message = fmt.Sprintf("failed %s validation", validatorName)
	}
	return fieldError{Location: location, Message: message}
}

var _ error = (*validationError)(nil)
