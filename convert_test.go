package amigo

import (
	"reflect"
	"testing"
	"time"
	"uuid"
)

func converted[T any](raw string) (T, error) {
	var value T
	err := setFieldValue(reflect.ValueOf(&value).Elem(), raw)
	return value, err
}

func TestSetFieldValue(t *testing.T) {
	if value, err := converted[string]("amigo"); err != nil || value != "amigo" {
		t.Fatalf("string = %q, err=%v", value, err)
	}
	if value, err := converted[bool]("true"); err != nil || !value {
		t.Fatalf("bool = %v, err=%v", value, err)
	}
	if value, err := converted[int8]("127"); err != nil || value != 127 {
		t.Fatalf("int8 = %d, err=%v", value, err)
	}
	if value, err := converted[uint16]("65535"); err != nil || value != 65535 {
		t.Fatalf("uint16 = %d, err=%v", value, err)
	}
	if value, err := converted[float64]("3.5"); err != nil || value != 3.5 {
		t.Fatalf("float64 = %v, err=%v", value, err)
	}
	wantTime := time.Date(2026, 8, 25, 10, 30, 0, 0, time.UTC)
	if value, err := converted[time.Time](wantTime.Format(time.RFC3339)); err != nil || !value.Equal(wantTime) {
		t.Fatalf("time = %v, err=%v", value, err)
	}
	wantUUID := uuid.MustParse("01941f29-7c00-7d00-a5c9-345f08c39fbd")
	if value, err := converted[uuid.UUID](wantUUID.String()); err != nil || value != wantUUID {
		t.Fatalf("UUID = %v, err=%v", value, err)
	}
}

func TestSetFieldValueRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "boolean", run: func() error { _, err := converted[bool]("yes"); return err }},
		{name: "integer overflow", run: func() error { _, err := converted[int8]("128"); return err }},
		{name: "unsigned integer", run: func() error { _, err := converted[uint]("-1"); return err }},
		{name: "number", run: func() error { _, err := converted[float64]("number"); return err }},
		{name: "time", run: func() error { _, err := converted[time.Time]("today"); return err }},
		{name: "UUID", run: func() error { _, err := converted[uuid.UUID]("uuid"); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); err == nil {
				t.Fatal("expected conversion error")
			}
		})
	}
}

func TestSetFieldValuesSupportsSlicesAndPointers(t *testing.T) {
	var values *[]int
	field := reflect.ValueOf(&values).Elem()
	if err := setFieldValues(field, []string{"1", "2", "3"}); err != nil {
		t.Fatal(err)
	}
	if values == nil || !reflect.DeepEqual(*values, []int{1, 2, 3}) {
		t.Fatalf("values = %#v", values)
	}
}
