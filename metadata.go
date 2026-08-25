package amigo

import (
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
	Type        reflect.Type
	Parameters  []ParameterMetadata
	Body        *BodyMetadata
	Validations []ValidationMetadata
}

// ParameterMetadata describes one path, query, header, or cookie parameter.
type ParameterMetadata struct {
	Name     string
	Source   ParameterSource
	Type     reflect.Type
	Required bool
	Default  *string

	index int
}

// BodyMetadata describes the input field containing the JSON request body.
type BodyMetadata struct {
	Type     reflect.Type
	Required bool

	index int
}

// ValidationMetadata describes the named rules applied to one input field.
type ValidationMetadata struct {
	Location string
	Type     reflect.Type
	Rules    []string

	index []int
}

// OutputMetadata describes the value returned by a handler.
type OutputMetadata struct {
	Type   reflect.Type
	Status int
}

func (metadata InputMetadata) clone() InputMetadata {
	cloned := metadata
	cloned.Parameters = append([]ParameterMetadata(nil), metadata.Parameters...)
	for index := range cloned.Parameters {
		if cloned.Parameters[index].Default != nil {
			value := *cloned.Parameters[index].Default
			cloned.Parameters[index].Default = &value
		}
	}
	if metadata.Body != nil {
		body := *metadata.Body
		cloned.Body = &body
	}
	cloned.Validations = make([]ValidationMetadata, len(metadata.Validations))
	for index, validation := range metadata.Validations {
		cloned.Validations[index] = validation
		cloned.Validations[index].index = append([]int(nil), validation.index...)
		cloned.Validations[index].Rules = append([]string(nil), validation.Rules...)
	}
	return cloned
}
