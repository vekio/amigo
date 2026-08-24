package amigo

import (
	"fmt"
	"reflect"
)

func inspectInput(inputType reflect.Type) (*InputMetadata, error) {
	if inputType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct, got %s", inputType.Kind())
	}

	metadata := &InputMetadata{Type: inputType}
	for i := range inputType.NumField() {
		field := inputType.Field(i)
		source, name, parameter, err := inspectParameter(field)
		if err != nil {
			return nil, err
		}
		if parameter && !field.IsExported() {
			return nil, fmt.Errorf("parameter field %q must be exported", field.Name)
		}

		if field.Name == "Body" {
			if parameter {
				return nil, fmt.Errorf("field %q cannot be both body and %s parameter", field.Name, source)
			}
			if _, ok := field.Tag.Lookup("default"); ok {
				return nil, fmt.Errorf("body field %q cannot have a default", field.Name)
			}
			metadata.Body = &BodyMetadata{
				Type:     field.Type,
				Required: field.Type.Kind() != reflect.Pointer,
				index:    i,
			}
			continue
		}

		if !parameter {
			if _, ok := field.Tag.Lookup("default"); ok {
				return nil, fmt.Errorf("field %q has a default but is not a query parameter", field.Name)
			}
			continue
		}
		if source != ParameterQuery {
			if _, ok := field.Tag.Lookup("default"); ok {
				return nil, fmt.Errorf("%s parameter field %q cannot have a default", source, field.Name)
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
	}

	return metadata, nil
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

func inspectOutput(outputType reflect.Type) *OutputMetadata {
	return &OutputMetadata{Type: outputType}
}
