package amigo

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type rendererFunc func(context.Context, io.Writer, string, any) error

func (render rendererFunc) Render(
	ctx context.Context,
	destination io.Writer,
	templateName string,
	data any,
) error {
	return render(ctx, destination, templateName, data)
}

func TestHTMLOutputUsesConfiguredRenderer(t *testing.T) {
	type pageData struct {
		Title string
	}

	renderer := rendererFunc(func(
		_ context.Context,
		destination io.Writer,
		templateName string,
		data any,
	) error {
		page := data.(pageData)
		_, err := io.WriteString(destination, templateName+":"+page.Title)
		return err
	})
	api := New(WithRenderer(renderer))
	api.GET("/", func(context.Context, struct{}) (HTML[pageData], error) {
		return HTMLView("layout", pageData{Title: "Home"}), nil
	}, WithStatus(http.StatusCreated))

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusCreated || response.Body.String() != "layout:Home" {
		t.Errorf("response = (%d, %q), want (201, layout:Home)", response.Code, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != htmlContentType {
		t.Errorf("Content-Type = %q, want %q", contentType, htmlContentType)
	}
	if output := api.operations[0].output; output.mediaType != htmlMediaType || !output.body.isEmpty() {
		t.Errorf("output metadata = %#v", output)
	}
}

func TestHTMLOutputDoesNotCommitPartialRender(t *testing.T) {
	renderFailure := errors.New("template failed")
	renderer := rendererFunc(func(_ context.Context, destination io.Writer, _ string, _ any) error {
		_, _ = io.WriteString(destination, "partial HTML")
		return renderFailure
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	api := New(WithRenderer(renderer), WithLogger(logger))
	api.GET("/", func(context.Context, struct{}) (HTML[struct{}], error) {
		return HTMLView("layout", struct{}{}), nil
	})

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), "partial HTML") {
		t.Errorf("partial render was written: %q", response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", contentType)
	}
}

func TestHTMLOutputRequiresRenderer(t *testing.T) {
	assertPanics(t, func() {
		New().GET("/", func(context.Context, struct{}) (HTML[struct{}], error) {
			return HTMLView("layout", struct{}{}), nil
		})
	})
}

func TestHTMLOutputRejectsBodylessStatus(t *testing.T) {
	renderer := rendererFunc(func(context.Context, io.Writer, string, any) error { return nil })
	assertPanics(t, func() {
		New(WithRenderer(renderer)).GET(
			"/",
			func(context.Context, struct{}) (HTML[struct{}], error) {
				return HTMLView("layout", struct{}{}), nil
			},
			WithStatus(http.StatusNoContent),
		)
	})
}

func TestWithRendererRejectsNil(t *testing.T) {
	assertPanics(t, func() { WithRenderer(nil) })
}
