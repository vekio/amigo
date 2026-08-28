package amigo

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithLoggerConfiguresErrorLogging(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	api := New(WithLogger(logger))
	api.GET("/things", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, errors.New("repository failed")
	})

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/things", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if output := logs.String(); !strings.Contains(output, `"msg":"request failed"`) ||
		!strings.Contains(output, `"path":"/things"`) ||
		!strings.Contains(output, `"error":"repository failed"`) {
		t.Errorf("log output = %s", output)
	}
}

func TestAPIRejectsInvalidOptions(t *testing.T) {
	t.Run("nil option", func(t *testing.T) {
		assertPanics(t, func() { New(nil) })
	})
	t.Run("nil logger", func(t *testing.T) {
		assertPanics(t, func() { WithLogger(nil) })
	})
}
