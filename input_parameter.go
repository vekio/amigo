package amigo

import (
	"fmt"
	"net/http"
	"reflect"
)

func bindPathParameters(
	value reflect.Value,
	request *http.Request,
	parameters []inputParameter,
	present fieldSet,
) error {
	for _, parameter := range parameters {
		if err := bindParameterValue(value, parameter, request.PathValue(parameter.name), "path"); err != nil {
			return err
		}
		present.add(parameter.fieldID)
	}
	return nil
}

func bindHeaderParameters(
	value reflect.Value,
	request *http.Request,
	parameters []inputParameter,
	present fieldSet,
) error {
	for _, parameter := range parameters {
		values := request.Header.Values(parameter.name)
		if len(values) == 0 {
			continue
		}
		if err := bindParameterValue(value, parameter, values[0], "header"); err != nil {
			return err
		}
		present.add(parameter.fieldID)
	}
	return nil
}

func bindParameterValue(value reflect.Value, parameter inputParameter, rawValue string, source string) error {
	field := value.FieldByIndex(parameter.fieldIndex)
	if err := setParameterValue(field, rawValue); err != nil {
		return invalidParameterProblem(source, parameter.name)
	}
	return nil
}

func invalidParameterProblem(source string, name string) *problem {
	return newProblem(http.StatusBadRequest, fmt.Sprintf("invalid %s parameter %q", source, name))
}
