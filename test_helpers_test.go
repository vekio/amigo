package amigo

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func request(
	t *testing.T,
	handler http.Handler,
	method string,
	target string,
	body io.Reader,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, body)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func decodeJSON[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()

	var value T
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode JSON response: %v; body=%q", err, response.Body.String())
	}
	return value
}

func requirePanicContains(t *testing.T, expected string, f func()) {
	t.Helper()

	defer func() {
		panicValue := recover()
		if panicValue == nil {
			t.Fatalf("expected panic containing %q", expected)
		}
		if message := panicMessage(panicValue); !strings.Contains(message, expected) {
			t.Fatalf("panic %q does not contain %q", message, expected)
		}
	}()
	f()
}

func panicMessage(value any) string {
	if err, ok := value.(error); ok {
		return err.Error()
	}
	if message, ok := value.(string); ok {
		return message
	}
	return ""
}
