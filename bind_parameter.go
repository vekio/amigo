package amigo

import (
	"errors"
	"net/http"
	"reflect"
)

func bindPathParameter(
	req *http.Request,
	fieldValue reflect.Value,
	metadata ParameterMetadata,
) error {
	raw := req.PathValue(metadata.Name)
	if raw == "" {
		return errors.New("value is required")
	}

	if err := setFieldValue(fieldValue, raw); err != nil {
		return errors.New(expectedValue(metadata.Type))
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
		return errors.New("value is required")
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
		return errors.New("value is required")
	}

	if err := setFieldValue(fieldValue, cookie.Value); err != nil {
		return errors.New(expectedValue(metadata.Type))
	}

	return nil
}
