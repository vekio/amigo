package amigo

import (
	"net/http"
	"testing"
)

func TestProblemErrorUsesDetailOrTitle(t *testing.T) {
	withDetail := newProblem(http.StatusBadRequest, "invalid input")
	if got := withDetail.Error(); got != "invalid input" {
		t.Errorf("Error() = %q, want %q", got, "invalid input")
	}

	withoutDetail := newProblem(http.StatusNotFound, "")
	if got := withoutDetail.Error(); got != http.StatusText(http.StatusNotFound) {
		t.Errorf("Error() = %q, want %q", got, http.StatusText(http.StatusNotFound))
	}
}
