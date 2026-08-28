package amigo

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"reflect"
)

func writeOutput[Out any](
	w http.ResponseWriter,
	status int,
	output Out,
	metadata outputMetadata,
) error {
	if status == http.StatusNoContent || status == http.StatusResetContent {
		writeOutputHeaders(w, reflect.ValueOf(output), metadata.headers)
		w.WriteHeader(status)
		return nil
	}

	data, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}

	writeOutputHeaders(w, reflect.ValueOf(output), metadata.headers)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
	return nil
}

func writeOutputHeaders(w http.ResponseWriter, output reflect.Value, headers []outputHeader) {
	for _, header := range headers {
		if value := output.FieldByIndex(header.fieldIndex).String(); value != "" {
			w.Header().Set(header.name, value)
		}
	}
}
