package amigo

import "net/http"

const (
	htmlMediaType   = "text/html"
	htmlContentType = htmlMediaType + "; charset=utf-8"
)

// HTML is an endpoint output containing previously rendered HTML content.
type HTML struct {
	Content string
}

// HTMLResponse creates an HTML output from previously rendered content.
func HTMLResponse(content string) HTML {
	return HTML{Content: content}
}

func writeHTML(w http.ResponseWriter, status int, output HTML) {
	w.Header().Set("Content-Type", htmlContentType)
	w.WriteHeader(status)
	_, _ = w.Write([]byte(output.Content))
}
