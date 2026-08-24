package amigo

import (
	"fmt"
	"reflect"
)

// ParameterSource identifies where an HTTP parameter comes from.
type ParameterSource uint8

const (
	ParameterUnknown ParameterSource = iota
	ParameterPath
	ParameterQuery
	ParameterHeader
	ParameterCookie
)

func (source ParameterSource) String() string {
	switch source {
	case ParameterPath:
		return "path"
	case ParameterQuery:
		return "query"
	case ParameterHeader:
		return "header"
	case ParameterCookie:
		return "cookie"
	default:
		return "unknown"
	}
}

// InputMetadata describes how an input type is populated from an HTTP request.
type InputMetadata struct {
	Type       reflect.Type
	Parameters []ParameterMetadata
	Body       *BodyMetadata
}

// ParameterMetadata describes one path, query, header, or cookie parameter.
type ParameterMetadata struct {
	Index    int
	Name     string
	Source   ParameterSource
	Type     reflect.Type
	Required bool
	Default  *string
}

// BodyMetadata describes the input field containing the JSON request body.
type BodyMetadata struct {
	Index    int
	Type     reflect.Type
	Required bool
}

// OutputMetadata describes the value returned by a handler.
type OutputMetadata struct {
	Type reflect.Type
}

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

		if field.Name == "Body" {
			if parameter {
				return nil, fmt.Errorf("field %q cannot be both body and %s parameter", field.Name, source)
			}
			metadata.Body = &BodyMetadata{
				Index:    i,
				Type:     field.Type,
				Required: field.Type.Kind() != reflect.Pointer,
			}
			continue
		}

		if !parameter {
			continue
		}

		parameterMetadata := ParameterMetadata{
			Index:    i,
			Name:     name,
			Source:   source,
			Type:     field.Type,
			Required: parameterRequired(source, field.Type),
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
