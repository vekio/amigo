package amigo

import "testing"

func assertPanics(t *testing.T, action func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("operation did not panic")
		}
	}()
	action()
}
