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
	Type       reflect.Type
	Parameters []ParameterMetadata
	Body       *BodyMetadata
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

// OutputMetadata describes the value returned by a handler.
type OutputMetadata struct {
	Type reflect.Type
}

func (metadata *InputMetadata) clone() *InputMetadata {
	if metadata == nil {
		return nil
	}
	cloned := *metadata
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
	return &cloned
}

func (metadata *OutputMetadata) clone() *OutputMetadata {
	if metadata == nil {
		return nil
	}
	cloned := *metadata
	return &cloned
}
