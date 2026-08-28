package amigo

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
)

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
