package amigo

import (
	"encoding"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strconv"
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

func bindInput[In any](request *http.Request, metadata inputMetadata) (In, error) {
	bound, err := bindInputWithPresence[In](request, metadata)
	return bound.value, err
}

func bindInputWithPresence[In any](
	request *http.Request,
	metadata inputMetadata,
) (boundInput[In], error) {
	var input In
	value := reflect.ValueOf(&input).Elem()
	bound := boundInput[In]{value: input, present: make(fieldSet)}

	properties, err := bindJSONBody(request, &input)
	if err != nil {
		return bound, err
	}
	markBodyFields(bound.present, properties, metadata.bodyFields)
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

func limitRequestBody(w http.ResponseWriter, request *http.Request, limit int64) {
	if limit > 0 && request.Body != nil {
		request.Body = http.MaxBytesReader(w, request.Body, limit)
	}
}

func bindJSONBody(request *http.Request, destination any) (map[string]jsontext.Value, error) {
	if request.Body == nil || request.Body == http.NoBody || request.ContentLength == 0 {
		return nil, nil
	}

	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return nil, newProblem(http.StatusUnsupportedMediaType, "content type must be application/json")
	}

	data, err := io.ReadAll(request.Body)
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return nil, newProblem(http.StatusRequestEntityTooLarge, "request body exceeds the maximum allowed size")
		}
		return nil, fmt.Errorf("read request body: %w", err)
	}
	if err := json.Unmarshal(data, destination, json.RejectUnknownMembers(true)); err != nil {
		return nil, newProblem(http.StatusBadRequest, "invalid JSON request body")
	}

	var properties map[string]jsontext.Value
	if err := json.Unmarshal(data, &properties); err != nil {
		return nil, newProblem(http.StatusBadRequest, "invalid JSON request body")
	}
	return properties, nil
}

func markBodyFields(present fieldSet, properties map[string]jsontext.Value, bodyFields map[string]int) {
	for name, rawValue := range properties {
		if rawValue.Kind() == jsontext.KindNull {
			continue
		}
		if fieldID, exists := bodyFields[name]; exists {
			present.add(fieldID)
		}
	}
}

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

func setParameterValue(field reflect.Value, value string) error {
	if unmarshaler, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
		return unmarshaler.UnmarshalText([]byte(value))
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(parsed)
	default:
		panic("amigo: unsupported input parameter type reached binding")
	}
	return nil
}
