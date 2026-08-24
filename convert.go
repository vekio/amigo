package amigo

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
)

func setFieldValue(fieldValue reflect.Value, raw string) error {
	if !fieldValue.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	if fieldValue.Kind() == reflect.Pointer {
		value := reflect.New(fieldValue.Type().Elem())
		if err := setFieldValue(value.Elem(), raw); err != nil {
			return err
		}
		fieldValue.Set(value)
		return nil
	}

	if fieldValue.CanAddr() && fieldValue.Addr().CanInterface() {
		unmarshaler, ok := fieldValue.Addr().Interface().(encoding.TextUnmarshaler)
		if ok {
			err := unmarshaler.UnmarshalText([]byte(raw))
			if err != nil {
				return fmt.Errorf("invalid %s %q: %w", fieldValue.Type(), raw, err)
			}
			return nil
		}
	}

	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(raw)
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("invalid bool %q: %w", raw, err)
		}
		fieldValue.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, fieldValue.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid %s %q: %w", fieldValue.Type(), raw, err)
		}
		fieldValue.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		value, err := strconv.ParseUint(raw, 10, fieldValue.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid %s %q: %w", fieldValue.Type(), raw, err)
		}
		fieldValue.SetUint(value)
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, fieldValue.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid %s %q: %w", fieldValue.Type(), raw, err)
		}
		fieldValue.SetFloat(value)
	default:
		return fmt.Errorf("unsupported type %s", fieldValue.Type())
	}

	return nil
}

func setFieldValues(fieldValue reflect.Value, rawValues []string) error {
	if fieldValue.Kind() != reflect.Slice {
		return setFieldValue(fieldValue, rawValues[0])
	}
	if !fieldValue.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	values := reflect.MakeSlice(fieldValue.Type(), len(rawValues), len(rawValues))
	for i, raw := range rawValues {
		if err := setFieldValue(values.Index(i), raw); err != nil {
			return fmt.Errorf("invalid value at position %d: %w", i, err)
		}
	}

	fieldValue.Set(values)
	return nil
}
