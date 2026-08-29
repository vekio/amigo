package amigo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTMLOutputWritesRenderedContent(t *testing.T) {
	api := New()
	api.GET("/", func(context.Context, struct{}) (HTML, error) {
		return HTMLResponse("<h1>Home</h1>"), nil
	}, WithStatus(http.StatusCreated))

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusCreated || response.Body.String() != "<h1>Home</h1>" {
		t.Errorf("response = (%d, %q), want (201, <h1>Home</h1>)", response.Code, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != htmlContentType {
		t.Errorf("Content-Type = %q, want %q", contentType, htmlContentType)
	}
	if output := api.operations[0].output; output.mediaType != htmlMediaType || !output.body.isEmpty() {
		t.Errorf("output metadata = %#v", output)
	}
}

func TestHTMLOutputRejectsBodylessStatus(t *testing.T) {
	assertPanics(t, func() {
		New().GET(
			"/",
			func(context.Context, struct{}) (HTML, error) {
				return HTMLResponse("<h1>Home</h1>"), nil
			},
			WithStatus(http.StatusNoContent),
		)
	})
}
