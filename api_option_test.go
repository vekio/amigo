package amigo

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
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

func TestStaticFilesServesFileSystem(t *testing.T) {
	api := New()
	api.StaticFiles("/static", fstest.MapFS{
		"app.css": {Data: []byte("body { color: navy; }")},
	})

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/static/app.css", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); body != "body { color: navy; }" {
		t.Errorf("body = %q", body)
	}
	if len(api.operations) != 0 {
		t.Errorf("operations = %d, want 0", len(api.operations))
	}
}

func TestStaticFilesSupportsHead(t *testing.T) {
	api := New()
	api.StaticFiles("/static/", fstest.MapFS{
		"app.js": {Data: []byte("alert('hello')")},
	})

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodHead, "/static/app.js", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", response.Body.String())
	}
}

func TestStaticFilesRedirectsPrefixToDirectory(t *testing.T) {
	api := New()
	api.StaticFiles("/assets", fstest.MapFS{
		"index.html": {Data: []byte("home")},
	})

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/assets", nil))

	if response.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTemporaryRedirect)
	}
	if location := response.Header().Get("Location"); location != "/assets/" {
		t.Errorf("Location = %q, want /assets/", location)
	}
}

func TestStaticFilesRejectsUnsupportedMethods(t *testing.T) {
	api := New()
	api.StaticFiles("/static", fstest.MapFS{
		"app.css": {Data: []byte("body {}")},
	})

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/static/app.css", nil))

	assertRoutingProblem(t, response, http.StatusMethodNotAllowed, "method not allowed", "/static/app.css")
}

func TestStaticFilesCanMountAtRoot(t *testing.T) {
	api := New()
	api.StaticFiles("/", fstest.MapFS{
		"index.html": {Data: []byte("home")},
	})

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusOK || response.Body.String() != "home" {
		t.Errorf("response = (%d, %q), want (200, home)", response.Code, response.Body.String())
	}
}

func TestStaticFilesRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		root   fs.FS
	}{
		{name: "empty prefix", root: fstest.MapFS{}},
		{name: "relative prefix", prefix: "static", root: fstest.MapFS{}},
		{name: "nil filesystem", prefix: "/static"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertPanics(t, func() { New().StaticFiles(test.prefix, test.root) })
		})
	}
}
