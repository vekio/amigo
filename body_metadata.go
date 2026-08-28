package amigo

import (
	"encoding/json/jsontext"
	"fmt"
	"reflect"
)

type bodyMetadata struct {
	fields      []bodyField
	indexByName map[string]int
}

type bodyField struct {
	name       string
	fieldID    int
	fieldIndex []int
	fieldType  reflect.Type
	jsonTag    string
}

func newBodyMetadata() bodyMetadata {
	return bodyMetadata{indexByName: make(map[string]int)}
}

func (body *bodyMetadata) add(field reflect.StructField, fieldID int, name string) {
	if _, exists := body.indexByName[name]; exists {
		panic(fmt.Sprintf("amigo: JSON body field %q is bound more than once", name))
	}

	body.indexByName[name] = len(body.fields)
	body.fields = append(body.fields, bodyField{
		name:       name,
		fieldID:    fieldID,
		fieldIndex: field.Index,
		fieldType:  field.Type,
		jsonTag:    field.Tag.Get("json"),
	})
}

func (body bodyMetadata) isEmpty() bool {
	return len(body.fields) == 0
}

func (body bodyMetadata) markPresent(properties map[string]jsontext.Value, present fieldSet) {
	for name, rawValue := range properties {
		if rawValue.Kind() == jsontext.KindNull {
			continue
		}
		if index, exists := body.indexByName[name]; exists {
			present.add(body.fields[index].fieldID)
		}
	}
}
