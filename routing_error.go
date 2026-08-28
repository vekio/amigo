package amigo

import "net/http"

type muxFallback struct {
	status int
	header http.Header
}

type muxFallbackWriter struct {
	header http.Header
	status int
}

func newMuxFallbackWriter() *muxFallbackWriter {
	return &muxFallbackWriter{header: make(http.Header)}
}

func (w *muxFallbackWriter) Header() http.Header {
	return w.header
}

func (w *muxFallbackWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *muxFallbackWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return len(data), nil
}

func inspectMuxFallback(handler http.Handler, request *http.Request) muxFallback {
	writer := newMuxFallbackWriter()
	handler.ServeHTTP(writer, request)
	return muxFallback{status: writer.status, header: writer.header}
}

func writeRoutingProblem(w http.ResponseWriter, request *http.Request, fallback muxFallback) {
	if allow := fallback.header.Get("Allow"); allow != "" {
		w.Header().Set("Allow", allow)
	}

	detail := "resource not found"
	if fallback.status == http.StatusMethodNotAllowed {
		detail = "method not allowed"
	}
	problem := newProblem(fallback.status, detail)
	problem.Instance = request.URL.Path
	writeProblem(w, problem)
}
