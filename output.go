package amigo

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"reflect"
)

const jsonMediaType = "application/json"

func statusAllowsBody(status int) bool {
	return status >= http.StatusOK &&
		status != http.StatusNoContent &&
		status != http.StatusResetContent &&
		status != http.StatusNotModified
}

func checkOutputStatus(status int, metadata outputMetadata) {
	if !statusAllowsBody(status) && metadata.mediaType != "" {
		panic(fmt.Sprintf("amigo: response status %d does not allow a response body", status))
	}
}

func writeEndpointOutput[Out any](
	w http.ResponseWriter,
	status int,
	output Out,
	metadata outputMetadata,
) error {
	if html, ok := any(output).(HTML); ok {
		writeHTML(w, status, html)
		return nil
	}
	return writeOutput(w, status, output, metadata)
}

func writeOutput[Out any](
	w http.ResponseWriter,
	status int,
	output Out,
	metadata outputMetadata,
) error {
	if !statusAllowsBody(status) || metadata.body.isEmpty() {
		writeOutputHeaders(w, reflect.ValueOf(output), metadata.headers)
		w.WriteHeader(status)
		return nil
	}

	data, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}

	writeOutputHeaders(w, reflect.ValueOf(output), metadata.headers)
	w.Header().Set("Content-Type", jsonMediaType)
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
