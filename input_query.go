package amigo

import (
	"fmt"
	"net/http"
	"reflect"
)

func bindQueryParameters(
	value reflect.Value,
	request *http.Request,
	parameters []inputParameter,
	present fieldSet,
) error {
	query := request.URL.Query()
	for _, parameter := range parameters {
		values, exists := query[parameter.name]
		if !exists {
			continue
		}
		if err := bindQueryParameter(value, parameter, values); err != nil {
			return err
		}
		present.add(parameter.fieldID)
	}
	return nil
}

func bindQueryParameter(value reflect.Value, parameter inputParameter, values []string) error {
	field := value.FieldByIndex(parameter.fieldIndex)
	if field.Kind() == reflect.Slice {
		if err := setParameterSlice(field, values); err != nil {
			return invalidParameterProblem("query", parameter.name)
		}
		return nil
	}
	if len(values) > 1 {
		return newProblem(
			http.StatusBadRequest,
			fmt.Sprintf("query parameter %q must not be repeated", parameter.name),
		)
	}
	if err := setParameterValue(field, values[0]); err != nil {
		return invalidParameterProblem("query", parameter.name)
	}
	return nil
}

func setParameterSlice(field reflect.Value, values []string) error {
	result := reflect.MakeSlice(field.Type(), len(values), len(values))
	for index, value := range values {
		if err := setParameterValue(result.Index(index), value); err != nil {
			return err
		}
	}
	field.Set(result)
	return nil
}
