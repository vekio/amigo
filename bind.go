package amigo

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

func bindInput[In any](req *http.Request, metadata *InputMetadata) (In, error) {
	var input In
	inputValue := reflect.ValueOf(&input).Elem()
	validation := &ValidationError{}

	for _, parameter := range metadata.Parameters {
		fieldValue := inputValue.Field(parameter.index)
		var err error
		switch parameter.Source {
		case ParameterPath:
			err = bindPathParameter(req, fieldValue, parameter)
		case ParameterQuery:
			err = bindQueryParameter(req, fieldValue, parameter)
		case ParameterHeader:
			err = bindHeaderParameter(req, fieldValue, parameter)
		case ParameterCookie:
			err = bindCookieParameter(req, fieldValue, parameter)
		default:
			err = fmt.Errorf("unsupported parameter source %s", parameter.Source)
		}
		if err != nil {
			validation.Errors = append(validation.Errors, FieldError{
				Location: parameter.Source.String() + "." + parameter.Name,
				Message:  err.Error(),
			})
		}
	}

	if metadata.Body != nil {
		fieldValue := inputValue.Field(metadata.Body.index)
		if err := bindBody(req, fieldValue, *metadata.Body); err != nil {
			if errors.Is(err, errUnsupportedMediaType) {
				return input, err
			}
			validation.Errors = append(validation.Errors, bodyFieldError(err))
		}
	}

	if len(validation.Errors) > 0 {
		return input, validation
	}

	return input, nil
}
