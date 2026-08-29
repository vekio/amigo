package amigo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	htmlMediaType   = "text/html"
	htmlContentType = htmlMediaType + "; charset=utf-8"
)

// Renderer renders a named template with data into destination. Amigo does
// not prescribe the template engine or how templates are stored.
type Renderer interface {
	Render(ctx context.Context, destination io.Writer, templateName string, data any) error
}

// HTML is a typed endpoint output rendered with the API's configured
// [Renderer].
type HTML[T any] struct {
	Template string
	Data     T
}

// HTMLView creates a typed HTML output.
func HTMLView[T any](templateName string, data T) HTML[T] {
	return HTML[T]{Template: templateName, Data: data}
}

type renderedOutput interface {
	render(context.Context, Renderer, io.Writer) error
}

func (output HTML[T]) render(ctx context.Context, renderer Renderer, destination io.Writer) error {
	return renderer.Render(ctx, destination, output.Template, output.Data)
}

func writeRenderedOutput(
	ctx context.Context,
	w http.ResponseWriter,
	status int,
	output renderedOutput,
	renderer Renderer,
) error {
	if renderer == nil {
		return fmt.Errorf("render response: renderer is not configured")
	}

	var rendered bytes.Buffer
	if err := output.render(ctx, renderer, &rendered); err != nil {
		return fmt.Errorf("render response: %w", err)
	}

	w.Header().Set("Content-Type", htmlContentType)
	w.WriteHeader(status)
	_, _ = rendered.WriteTo(w)
	return nil
}
