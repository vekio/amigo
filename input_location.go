package amigo

import (
	"fmt"
	"reflect"
	"strings"
)

type inputLocation struct {
	source string
	name   string
}

func (location inputLocation) label() string {
	return location.source + "." + location.name
}

func locateInputField(field reflect.StructField) inputLocation {
	for _, source := range []string{"path", "query", "header"} {
		if name, exists := field.Tag.Lookup(source); exists {
			return inputLocation{source: source, name: name}
		}
	}

	name, exists := bodyFieldName(field)
	if !exists {
		return inputLocation{}
	}
	return inputLocation{source: "body", name: name}
}

func bodyFieldName(field reflect.StructField) (string, bool) {
	tag, tagged := field.Tag.Lookup("json")
	if !field.IsExported() {
		if tagged && tag != "-" {
			panic(fmt.Sprintf("amigo: unexported field %s cannot have a JSON tag", field.Name))
		}
		return "", false
	}
	if tag == "-" {
		return "", false
	}

	name, options, _ := strings.Cut(tag, ",")
	for option := range strings.SplitSeq(options, ",") {
		if option == "embed" {
			panic(fmt.Sprintf("amigo: embedded JSON field %s is not supported", field.Name))
		}
		if option == "case:ignore" {
			panic(fmt.Sprintf("amigo: case-insensitive JSON field %s is not supported", field.Name))
		}
	}
	if field.Anonymous && name == "" {
		panic(fmt.Sprintf("amigo: embedded JSON field %s is not supported", field.Name))
	}
	if name == "" {
		name = field.Name
	}
	return name, true
}
