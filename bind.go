package amigo

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"reflect"
	"strings"
)

var errUnsupportedMediaType = errors.New("content type must be application/json")

func bindInput[In any](req *http.Request, metadata *InputMetadata) (In, error) {
	var input In
	inputValue := reflect.ValueOf(&input).Elem()
	validation := &ValidationError{}

	for _, parameter := range metadata.Parameters {
		fieldValue := inputValue.Field(parameter.Index)
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
		fieldValue := inputValue.Field(metadata.Body.Index)
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

func bindPathParameter(
	req *http.Request,
	fieldValue reflect.Value,
	metadata ParameterMetadata,
) error {
	raw := req.PathValue(metadata.Name)
	if raw == "" {
		return fmt.Errorf("value is required")
	}

	if err := setFieldValue(fieldValue, raw); err != nil {
		return errors.New(expectedValue(metadata.Type))
	}

	return nil
}

func bindHeaderParameter(
	req *http.Request,
	fieldValue reflect.Value,
	metadata ParameterMetadata,
) error {
	values := req.Header.Values(metadata.Name)
	if len(values) == 0 {
		if !metadata.Required {
			return nil
		}
		return fmt.Errorf("value is required")
	}

	if err := setFieldValue(fieldValue, values[0]); err != nil {
		return errors.New(expectedValue(metadata.Type))
	}

	return nil
}

func bindCookieParameter(
	req *http.Request,
	fieldValue reflect.Value,
	metadata ParameterMetadata,
) error {
	cookie, err := req.Cookie(metadata.Name)
	if err != nil {
		if !metadata.Required {
			return nil
		}
		return fmt.Errorf("value is required")
	}

	if err := setFieldValue(fieldValue, cookie.Value); err != nil {
		return errors.New(expectedValue(metadata.Type))
	}

	return nil
}

func bindBody(
	req *http.Request,
	fieldValue reflect.Value,
	metadata BodyMetadata,
) error {
	if req.Body == nil || req.Body == http.NoBody || req.ContentLength == 0 {
		if !metadata.Required {
			return nil
		}
		return fmt.Errorf("value is required")
	}

	mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errUnsupportedMediaType
	}

	if !fieldValue.CanAddr() {
		return fmt.Errorf("body field cannot be addressed")
	}

	if err := json.UnmarshalRead(
		req.Body,
		fieldValue.Addr().Interface(),
		json.RejectUnknownMembers(true),
	); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}

	return nil
}

func bindQueryParameter(
	req *http.Request,
	fieldValue reflect.Value,
	metadata ParameterMetadata,
) error {
	values, present := req.URL.Query()[metadata.Name]
	if !present {
		if metadata.Default == nil {
			return nil
		}
		values = []string{*metadata.Default}
	}

	if len(values) == 0 {
		values = []string{""}
	}

	if err := setFieldValues(fieldValue, values); err != nil {
		return errors.New(expectedValue(metadata.Type))
	}

	return nil
}

func expectedValue(fieldType reflect.Type) string {
	for fieldType.Kind() == reflect.Pointer || fieldType.Kind() == reflect.Slice {
		fieldType = fieldType.Elem()
	}

	switch fieldType.Kind() {
	case reflect.String:
		return "expected string"
	case reflect.Bool:
		return "expected boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "expected integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return "expected unsigned integer"
	case reflect.Float32, reflect.Float64:
		return "expected number"
	default:
		return "expected " + fieldType.String()
	}
}

func bodyFieldError(err error) FieldError {
	if semantic, ok := errors.AsType[*json.SemanticError](err); ok {
		message := "invalid value"
		if semantic.GoType != nil {
			message = expectedValue(semantic.GoType)
		}
		if semantic.Err != nil && strings.Contains(semantic.Err.Error(), "unknown object member") {
			message = semantic.Err.Error()
		}
		return FieldError{
			Location: bodyLocation(semantic.JSONPointer),
			Message:  message,
		}
	}

	if syntactic, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		return FieldError{
			Location: bodyLocation(syntactic.JSONPointer),
			Message:  "invalid JSON",
		}
	}

	return FieldError{
		Location: "body",
		Message:  err.Error(),
	}
}

func bodyLocation(pointer jsontext.Pointer) string {
	parts := []string{"body"}
	for token := range pointer.Tokens() {
		parts = append(parts, token)
	}
	return strings.Join(parts, ".")
}
