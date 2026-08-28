package amigo

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"
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

func TestWriteOutputMapsSpecialJSONTypes(t *testing.T) {
	type outputBody struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
	}
	response := httptest.NewRecorder()
	output := outputBody{
		ID:        uuid.MustParse("f81d4fae-7dec-11d0-a765-00a0c91e6bf6"),
		CreatedAt: time.Date(2026, time.August, 28, 14, 30, 45, 0, time.UTC),
	}

	err := writeOutput(response, http.StatusOK, output, buildOutputMetadata[outputBody]())

	if err != nil {
		t.Fatalf("writeOutput() error = %v", err)
	}
	want := `{"id":"f81d4fae-7dec-11d0-a765-00a0c91e6bf6","created_at":"2026-08-28T14:30:45Z"}`
	if response.Body.String() != want {
		t.Errorf("body = %s, want %s", response.Body.String(), want)
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

func TestWriteOutputSkipsJSONForHeaderOnlyOutput(t *testing.T) {
	type outputHeaders struct {
		Location string `header:"Location" json:"-"`
	}

	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusTemporaryRedirect} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			response := httptest.NewRecorder()
			err := writeOutput(
				response,
				status,
				outputHeaders{Location: "/things/abc"},
				buildOutputMetadata[outputHeaders](),
			)

			if err != nil {
				t.Fatalf("writeOutput() error = %v", err)
			}
			if response.Code != status || response.Body.Len() != 0 {
				t.Errorf("status = %d, body = %q", response.Code, response.Body.String())
			}
			if response.Header().Get("Location") != "/things/abc" {
				t.Errorf("Location = %q", response.Header().Get("Location"))
			}
			if response.Header().Get("Content-Type") != "" {
				t.Errorf("Content-Type = %q, want empty", response.Header().Get("Content-Type"))
			}
		})
	}
}

func TestWriteOutputAllowsJSONForRedirect(t *testing.T) {
	type outputBody struct {
		Choices []string `json:"choices"`
	}
	response := httptest.NewRecorder()

	err := writeOutput(
		response,
		http.StatusMultipleChoices,
		outputBody{Choices: []string{"/first", "/second"}},
		buildOutputMetadata[outputBody](),
	)

	if err != nil {
		t.Fatalf("writeOutput() error = %v", err)
	}
	if response.Code != http.StatusMultipleChoices || response.Body.String() != `{"choices":["/first","/second"]}` {
		t.Errorf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestCheckOutputStatusRejectsJSONBodyForBodylessStatus(t *testing.T) {
	type outputBody struct {
		Code string `json:"code"`
	}
	metadata := buildOutputMetadata[outputBody]()

	for _, status := range []int{http.StatusNoContent, http.StatusResetContent, http.StatusNotModified} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			assertPanics(t, func() { checkOutputStatus(status, metadata) })
		})
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
