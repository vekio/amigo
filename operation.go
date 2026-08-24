package amigo

import "net/http"

// Operation describes a complete route registered in an API.
type Operation struct {
	Method string
	Path   string
	Tags   []string
	Input  *InputMetadata
	Output *OutputMetadata

	handler http.Handler
}
