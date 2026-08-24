package amigo

import (
	"net/http"
	"slices"
)

// Operation describes a complete route registered in an API.
type Operation struct {
	Method string
	Path   string
	Tags   []string
	Input  *InputMetadata
	Output *OutputMetadata

	handler http.Handler
}

func (operation *Operation) clone() Operation {
	cloned := *operation
	cloned.Tags = slices.Clone(operation.Tags)
	cloned.Input = operation.Input.clone()
	cloned.Output = operation.Output.clone()
	return cloned
}
