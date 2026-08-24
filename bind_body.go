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
