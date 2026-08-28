package amigo

import (
	"net/http"
	"reflect"
)

type fieldSet map[int]struct{}

func (fields fieldSet) add(fieldID int) {
	fields[fieldID] = struct{}{}
}

func (fields fieldSet) contains(fieldID int) bool {
	_, exists := fields[fieldID]
	return exists
}

type boundInput[In any] struct {
	value   In
	present fieldSet
}

func bindInput[In any](request *http.Request, metadata inputMetadata) (boundInput[In], error) {
	var input In
	value := reflect.ValueOf(&input).Elem()
	bound := boundInput[In]{value: input, present: make(fieldSet)}

	properties, err := bindJSONBody(request, &input)
	if err != nil {
		return bound, err
	}
	metadata.body.markPresent(properties, bound.present)
	if err := bindPathParameters(value, request, metadata.pathParameters, bound.present); err != nil {
		return bound, err
	}
	if err := bindQueryParameters(value, request, metadata.queryParameters, bound.present); err != nil {
		return bound, err
	}
	if err := bindHeaderParameters(value, request, metadata.headerParameters, bound.present); err != nil {
		return bound, err
	}
	bound.value = input
	return bound, nil
}
