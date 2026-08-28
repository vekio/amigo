package amigo

import (
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

// operation is the Go representation of one OpenAPI operation. Typed routes
// populate their input and output types and metadata during registration. Raw
// routes leave those fields empty because their schemas cannot be inferred.
type operation struct {
	method      string
	path        string
	status      int
	operationID string
	summary     string
	description string
	tags        []string
	inputType   reflect.Type
	outputType  reflect.Type
	input       inputMetadata
	output      outputMetadata
	errors      []errorMapping
}

func newOperation(method string, path string) operation {
	return operation{
		method:      method,
		path:        path,
		status:      http.StatusOK,
		operationID: defaultOperationID(method, path),
	}
}

func defaultOperationID(method string, path string) string {
	methodSlug := slug(method)
	if methodSlug == "" {
		methodSlug = "operation"
	}
	pathSlug := slug(path)
	if pathSlug == "" {
		pathSlug = "root"
	}
	return methodSlug + "-" + pathSlug
}

func slug(value string) string {
	var result strings.Builder
	separator := false

	for _, character := range value {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if separator && result.Len() > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(unicode.ToLower(character))
			separator = false
			continue
		}
		separator = true
	}

	return result.String()
}
