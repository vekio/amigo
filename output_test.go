package amigo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteOutputWritesJSON(t *testing.T) {
	type outputBody struct {
		Code string `json:"code"`
	}
	response := httptest.NewRecorder()

	err := writeOutput(response, http.StatusCreated, outputBody{Code: "abc"}, buildOutputMetadata[outputBody]())

	if err != nil {
		t.Fatalf("writeOutput() error = %v", err)
	}
	if response.Code != http.StatusCreated || response.Body.String() != `{"code":"abc"}` {
		t.Errorf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", response.Header().Get("Content-Type"))
	}
}

func TestWriteOutputWritesNoContent(t *testing.T) {
	response := httptest.NewRecorder()

	err := writeOutput(response, http.StatusNoContent, struct{}{}, buildOutputMetadata[struct{}]())

	if err != nil {
		t.Fatalf("writeOutput() error = %v", err)
	}
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Errorf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestWriteOutputReturnsEncodingErrorBeforeWriting(t *testing.T) {
	type outputBody struct {
		Unsupported func() `json:"unsupported"`
	}
	response := httptest.NewRecorder()

	err := writeOutput(
		response,
		http.StatusOK,
		outputBody{Unsupported: func() {}},
		buildOutputMetadata[outputBody](),
	)

	if err == nil {
		t.Fatal("writeOutput() error = nil")
	}
	if response.Body.Len() != 0 || response.Header().Get("Content-Type") != "" {
		t.Errorf("headers = %#v, body = %s", response.Header(), response.Body.String())
	}
}

func TestWriteOutputWritesHeaders(t *testing.T) {
	type outputBody struct {
		Location string `header:"Location" json:"-"`
		Code     string `json:"code"`
	}
	response := httptest.NewRecorder()

	err := writeOutput(
		response,
		http.StatusCreated,
		outputBody{Location: "/things/abc", Code: "abc"},
		buildOutputMetadata[outputBody](),
	)

	if err != nil {
		t.Fatalf("writeOutput() error = %v", err)
	}
	if response.Header().Get("Location") != "/things/abc" {
		t.Errorf("Location = %q", response.Header().Get("Location"))
	}
	if response.Body.String() != `{"code":"abc"}` {
		t.Errorf("body = %s", response.Body.String())
	}
}
