package amigo_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/vekio/amigo"
)

type integrationInput struct {
	ProjectID uuid.UUID `path:"project_id" json:"-"`
	Tags      []string  `query:"tag" json:"-" validate:"required"`
	RequestID string    `header:"X-Request-ID" json:"-" validate:"required"`
	Name      string    `json:"name" validate:"required"`
}

type integrationOutput struct {
	Location  string   `header:"Location" json:"-"`
	ProjectID string   `json:"project_id"`
	Name      string   `json:"name"`
	Tags      []string `json:"tags"`
	RequestID string   `json:"request_id"`
}

func TestTypedEndpointBindsAndWritesCompleteHTTPContract(t *testing.T) {
	projectID := uuid.MustParse("f81d4fae-7dec-11d0-a765-00a0c91e6bf6")
	wantInput := integrationInput{
		ProjectID: projectID,
		Tags:      []string{"go", "http"},
		RequestID: "request-42",
		Name:      "shorty",
	}
	var received integrationInput

	api := amigo.New()
	api.POST("/projects/{project_id}/links", func(_ context.Context, input integrationInput) (integrationOutput, error) {
		received = input
		return integrationOutput{
			Location:  "/projects/" + input.ProjectID.String() + "/links/created",
			ProjectID: input.ProjectID.String(),
			Name:      input.Name,
			Tags:      input.Tags,
			RequestID: input.RequestID,
		}, nil
	}, amigo.WithStatus(http.StatusCreated))

	request := httptest.NewRequest(
		http.MethodPost,
		"/projects/"+projectID.String()+"/links?tag=go&tag=http",
		strings.NewReader(`{"name":"shorty"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "request-42")
	response := httptest.NewRecorder()
	api.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
	wantLocation := "/projects/" + projectID.String() + "/links/created"
	if location := response.Header().Get("Location"); location != wantLocation {
		t.Errorf("Location = %q, want %q", location, wantLocation)
	}
	if !reflect.DeepEqual(received, wantInput) {
		t.Errorf("endpoint input = %#v, want %#v", received, wantInput)
	}

	var output integrationOutput
	if err := json.Unmarshal(response.Body.Bytes(), &output); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	wantOutput := integrationOutput{
		ProjectID: projectID.String(),
		Name:      "shorty",
		Tags:      []string{"go", "http"},
		RequestID: "request-42",
	}
	if !reflect.DeepEqual(output, wantOutput) {
		t.Errorf("output = %#v, want %#v", output, wantOutput)
	}
}
