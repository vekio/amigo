package amigo

import (
	"encoding"
	"fmt"
	"net/textproto"
	"reflect"
	"strings"
)

var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

// Registration checks panic because malformed endpoint metadata is a
// programming error rather than an invalid client request.
func checkSingleInputSource(field reflect.StructField) {
	bindings := 0
	for _, tag := range []string{"path", "query", "header"} {
		if _, exists := field.Tag.Lookup(tag); exists {
			bindings++
		}
	}
	if bindings > 1 {
		panic(fmt.Sprintf("amigo: input field %s cannot bind more than one parameter source", field.Name))
	}
}

func checkPathField(
	field reflect.StructField,
	name string,
	path string,
	pathNames map[string]struct{},
	boundNames map[string]struct{},
) {
	if !field.IsExported() {
		panic(fmt.Sprintf("amigo: path field %s must be exported", field.Name))
	}
	if name == "" {
		panic(fmt.Sprintf("amigo: path field %s has an empty name", field.Name))
	}
	if _, exists := pathNames[name]; !exists {
		panic(fmt.Sprintf("amigo: path parameter %q is not present in route %q", name, path))
	}
	if _, exists := boundNames[name]; exists {
		panic(fmt.Sprintf("amigo: path parameter %q is bound more than once", name))
	}
	if !supportsScalarParameterType(field.Type) {
		panic(fmt.Sprintf("amigo: path parameter %q has unsupported type %s", name, field.Type))
	}
	if jsonTagName(field.Tag.Get("json")) != "-" {
		panic(fmt.Sprintf("amigo: path field %s must use json:\"-\"", field.Name))
	}
}

func checkInputParameterField(
	field reflect.StructField,
	source string,
	name string,
	boundNames map[string]struct{},
) {
	if !field.IsExported() {
		panic(fmt.Sprintf("amigo: %s field %s must be exported", source, field.Name))
	}
	if name == "" {
		panic(fmt.Sprintf("amigo: %s field %s has an empty name", source, field.Name))
	}
	if _, exists := boundNames[normalizedParameterName(source, name)]; exists {
		panic(fmt.Sprintf("amigo: %s parameter %q is bound more than once", source, name))
	}
	if !supportsInputParameterType(source, field.Type) {
		panic(fmt.Sprintf("amigo: %s parameter %q has unsupported type %s", source, name, field.Type))
	}
	if jsonTagName(field.Tag.Get("json")) != "-" {
		panic(fmt.Sprintf("amigo: %s field %s must use json:\"-\"", source, field.Name))
	}
}

func normalizedParameterName(source string, name string) string {
	if source == "header" {
		return textproto.CanonicalMIMEHeaderKey(name)
	}
	return name
}

func checkOutputHeaderField(field reflect.StructField, name string) {
	if !field.IsExported() {
		panic(fmt.Sprintf("amigo: header field %s must be exported", field.Name))
	}
	if name == "" {
		panic(fmt.Sprintf("amigo: header field %s has an empty name", field.Name))
	}
	if field.Type.Kind() != reflect.String {
		panic(fmt.Sprintf("amigo: response header %q must be a string", name))
	}
	if jsonTagName(field.Tag.Get("json")) != "-" {
		panic(fmt.Sprintf("amigo: header field %s must use json:\"-\"", field.Name))
	}
}

func supportsInputParameterType(source string, valueType reflect.Type) bool {
	if source == "query" && valueType.Kind() == reflect.Slice {
		return supportsScalarParameterType(valueType.Elem())
	}
	return supportsScalarParameterType(valueType)
}

func supportsScalarParameterType(valueType reflect.Type) bool {
	if reflect.PointerTo(valueType).Implements(textUnmarshalerType) {
		return true
	}
	switch valueType.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func checkGroupPrefix(prefix string) {
	if prefix == "" {
		panic("amigo: router prefix cannot be empty")
	}
	if !strings.HasPrefix(prefix, "/") {
		panic(fmt.Sprintf("amigo: router prefix %q must start with a slash", prefix))
	}
	if strings.HasSuffix(prefix, "/") {
		panic(fmt.Sprintf("amigo: router prefix %q cannot end with a slash", prefix))
	}
}
