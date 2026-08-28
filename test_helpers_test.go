package amigo

import (
	"net/http"
	"testing"
)

func bindInputValue[In any](request *http.Request, metadata inputMetadata) (In, error) {
	bound, err := bindInput[In](request, metadata)
	return bound.value, err
}

func assertPanics(t *testing.T, action func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("operation did not panic")
		}
	}()
	action()
}
