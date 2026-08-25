package amigo

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

func inspectInput(inputType reflect.Type) (InputMetadata, error) {
	if inputType.Kind() != reflect.Struct {
		return InputMetadata{}, fmt.Errorf("input must be a struct, got %s", inputType.Kind())
	}

	metadata := InputMetadata{Type: inputType}
	for i := range inputType.NumField() {
		field := inputType.Field(i)
		source, name, parameter, err := inspectParameter(field)
		if err != nil {
			return InputMetadata{}, err
		}
		if parameter && !field.IsExported() {
			return InputMetadata{}, fmt.Errorf("parameter field %q must be exported", field.Name)
		}

		if field.Name == "Body" {
			if parameter {
				return InputMetadata{}, fmt.Errorf("field %q cannot be both body and %s parameter", field.Name, source)
			}
			if _, ok := field.Tag.Lookup("default"); ok {
				return InputMetadata{}, fmt.Errorf("body field %q cannot have a default", field.Name)
			}
			metadata.Body = &BodyMetadata{
				Type:     field.Type,
				Required: field.Type.Kind() != reflect.Pointer,
				index:    i,
			}
			if err := inspectValidation(field, "body", []int{i}, &metadata); err != nil {
				return InputMetadata{}, err
			}
			if err := inspectBodyValidations(field.Type, "body", []int{i}, &metadata); err != nil {
				return InputMetadata{}, err
			}
			continue
		}

		if !parameter {
			if _, ok := field.Tag.Lookup("default"); ok {
				return InputMetadata{}, fmt.Errorf("field %q has a default but is not a query parameter", field.Name)
			}
			if _, ok := field.Tag.Lookup("validate"); ok {
				return InputMetadata{}, fmt.Errorf("field %q has validation rules but is not a request parameter", field.Name)
			}
			continue
		}
		if source != ParameterQuery {
			if _, ok := field.Tag.Lookup("default"); ok {
				return InputMetadata{}, fmt.Errorf("%s parameter field %q cannot have a default", source, field.Name)
			}
		}

		parameterMetadata := ParameterMetadata{
			Name:     name,
			Source:   source,
			Type:     field.Type,
			Required: parameterRequired(source, field.Type),
			index:    i,
		}
		if source == ParameterQuery {
			if value, ok := field.Tag.Lookup("default"); ok {
				parameterMetadata.Default = &value
			}
		}

		metadata.Parameters = append(metadata.Parameters, parameterMetadata)
		location := source.String() + "." + name
		if err := inspectValidation(field, location, []int{i}, &metadata); err != nil {
			return InputMetadata{}, err
		}
	}

	return metadata, nil
}

func inspectBodyValidations(
	fieldType reflect.Type,
	location string,
	index []int,
	metadata *InputMetadata,
) error {
	return inspectNestedBodyValidations(fieldType, location, index, metadata, make(map[reflect.Type]bool))
}

func inspectNestedBodyValidations(
	fieldType reflect.Type,
	location string,
	index []int,
	metadata *InputMetadata,
	visiting map[reflect.Type]bool,
) error {
	for fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}
	if fieldType.Kind() != reflect.Struct {
		return nil
	}
	if visiting[fieldType] {
		return nil
	}
	visiting[fieldType] = true
	defer delete(visiting, fieldType)

	for fieldIndex := range fieldType.NumField() {
		field := fieldType.Field(fieldIndex)
		if !field.IsExported() {
			continue
		}
		name, included := jsonFieldName(field)
		if !included {
			continue
		}
		fieldLocation := location + "." + name
		fieldPath := append(append([]int(nil), index...), fieldIndex)
		if err := inspectValidation(field, fieldLocation, fieldPath, metadata); err != nil {
			return err
		}
		if err := inspectNestedBodyValidations(field.Type, fieldLocation, fieldPath, metadata, visiting); err != nil {
			return err
		}
	}
	return nil
}

func inspectValidation(
	field reflect.StructField,
	location string,
	index []int,
	metadata *InputMetadata,
) error {
	raw, exists := field.Tag.Lookup("validate")
	if !exists || raw == "-" {
		return nil
	}
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("field %q has an empty validate tag", field.Name)
	}

	rules := strings.Split(raw, ",")
	for ruleIndex := range rules {
		rules[ruleIndex] = strings.TrimSpace(rules[ruleIndex])
		if rules[ruleIndex] == "" || rules[ruleIndex] == "-" {
			return fmt.Errorf("field %q has an invalid validation rule", field.Name)
		}
	}
	metadata.Validations = append(metadata.Validations, ValidationMetadata{
		Location: location,
		Type:     field.Type,
		Rules:    rules,
		index:    append([]int(nil), index...),
	})
	return nil
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "-" {
		return "", false
	}
	if name == "" {
		name = field.Name
	}
	return name, true
}

func inspectParameter(field reflect.StructField) (ParameterSource, string, bool, error) {
	var source ParameterSource
	var name string

	for _, candidate := range []struct {
		tag    string
		source ParameterSource
	}{
		{tag: "path", source: ParameterPath},
		{tag: "query", source: ParameterQuery},
		{tag: "header", source: ParameterHeader},
		{tag: "cookie", source: ParameterCookie},
	} {
		value, ok := field.Tag.Lookup(candidate.tag)
		if !ok {
			continue
		}
		if source != ParameterUnknown {
			return ParameterUnknown, "", false, fmt.Errorf("field %q cannot have multiple parameter sources", field.Name)
		}
		if value == "" {
			return ParameterUnknown, "", false, fmt.Errorf("%s field %q has an empty tag", candidate.source, field.Name)
		}
		source = candidate.source
		name = value
	}

	return source, name, source != ParameterUnknown, nil
}

func parameterRequired(source ParameterSource, fieldType reflect.Type) bool {
	if source == ParameterPath {
		return true
	}
	if source == ParameterQuery {
		return false
	}
	return fieldType.Kind() != reflect.Pointer
}

func inspectOutput(outputType reflect.Type) OutputMetadata {
	return OutputMetadata{
		Type:   outputType,
		Status: http.StatusOK,
	}
}
