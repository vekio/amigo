package amigo

import (
	"fmt"
	"reflect"
	"strings"
)

type inputMetadata struct {
	pathParameters   []inputParameter
	queryParameters  []inputParameter
	headerParameters []inputParameter
	bodyFields       map[string]int
	validations      []fieldValidation
}

type inputParameter struct {
	name       string
	fieldID    int
	fieldIndex []int
}

type outputMetadata struct {
	headers []outputHeader
}

type outputHeader struct {
	name       string
	fieldIndex []int
}

func buildInputMetadata[In any](
	path string,
	validators validatorRegistry,
) inputMetadata {
	inputType := reflect.TypeFor[In]()
	if inputType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("amigo: endpoint input must be a struct, got %s", inputType))
	}

	pathNames := routePathNames(path)
	boundPathNames := make(map[string]struct{}, len(pathNames))
	boundQueryNames := make(map[string]struct{})
	boundHeaderNames := make(map[string]struct{})
	metadata := inputMetadata{bodyFields: make(map[string]int)}

	for field := range inputType.Fields() {
		checkSingleInputSource(field)
		fieldID := field.Index[0]

		if parameter, ok := buildPathParameter(field, fieldID, path, pathNames, boundPathNames); ok {
			metadata.pathParameters = append(metadata.pathParameters, parameter)
		}
		if parameter, ok := buildInputParameter(field, fieldID, "query", boundQueryNames); ok {
			metadata.queryParameters = append(metadata.queryParameters, parameter)
		}
		if parameter, ok := buildInputParameter(field, fieldID, "header", boundHeaderNames); ok {
			metadata.headerParameters = append(metadata.headerParameters, parameter)
		}
		if validation, ok := buildFieldValidation(field, fieldID, validators); ok {
			metadata.validations = append(metadata.validations, validation)
			if source, name := inputFieldLocation(field); source == "body" {
				metadata.bodyFields[name] = fieldID
			}
		}
	}

	for name := range pathNames {
		if _, exists := boundPathNames[name]; !exists {
			panic(fmt.Sprintf("amigo: path parameter %q is not bound by the endpoint input", name))
		}
	}

	return metadata
}

func buildPathParameter(
	field reflect.StructField,
	fieldID int,
	path string,
	pathNames map[string]struct{},
	boundNames map[string]struct{},
) (inputParameter, bool) {
	name, tagged := field.Tag.Lookup("path")
	if !tagged {
		return inputParameter{}, false
	}
	checkPathField(field, name, path, pathNames, boundNames)

	boundNames[name] = struct{}{}
	return inputParameter{name: name, fieldID: fieldID, fieldIndex: field.Index}, true
}

func buildInputParameter(
	field reflect.StructField,
	fieldID int,
	tag string,
	boundNames map[string]struct{},
) (inputParameter, bool) {
	name, tagged := field.Tag.Lookup(tag)
	if !tagged {
		return inputParameter{}, false
	}
	checkInputParameterField(field, tag, name, boundNames)

	boundNames[normalizedParameterName(tag, name)] = struct{}{}
	return inputParameter{name: name, fieldID: fieldID, fieldIndex: field.Index}, true
}

func buildOutputMetadata[Out any]() outputMetadata {
	outputType := reflect.TypeFor[Out]()
	if outputType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("amigo: endpoint output must be a struct, got %s", outputType))
	}

	metadata := outputMetadata{}
	for field := range outputType.Fields() {
		header, ok := buildOutputHeader(field)
		if !ok {
			continue
		}
		metadata.headers = append(metadata.headers, header)
	}
	return metadata
}

func buildOutputHeader(field reflect.StructField) (outputHeader, bool) {
	name, tagged := field.Tag.Lookup("header")
	if !tagged {
		return outputHeader{}, false
	}
	checkOutputHeaderField(field, name)
	return outputHeader{name: name, fieldIndex: field.Index}, true
}

func routePathNames(path string) map[string]struct{} {
	names := make(map[string]struct{})
	for segment := range strings.SplitSeq(path, "/") {
		if len(segment) < 3 || segment[0] != '{' || segment[len(segment)-1] != '}' {
			continue
		}
		name := strings.TrimSuffix(segment[1:len(segment)-1], "...")
		if name != "$" {
			names[name] = struct{}{}
		}
	}
	return names
}

func jsonTagName(tag string) string {
	name, _, _ := strings.Cut(tag, ",")
	return name
}
