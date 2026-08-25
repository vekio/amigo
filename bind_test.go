package amigo

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"
)

type bindingBody struct {
	Name string `json:"name"`
}

type bindingInput struct {
	ID      int        `path:"id"`
	Page    int        `query:"page" default:"1"`
	Tags    []string   `query:"tag"`
	Since   *time.Time `query:"since"`
	Enabled bool       `query:"enabled"`
	Token   string     `header:"X-Token"`
	Trace   *string    `header:"X-Trace"`
	Session string     `cookie:"session"`
	Body    bindingBody
}

type bindingOutput struct {
	Input bindingInput `json:"input"`
}

func TestBindRequestSources(t *testing.T) {
	api := New()
	api.POST("/items/{id}", func(_ context.Context, input bindingInput) (bindingOutput, error) {
		return bindingOutput{Input: input}, nil
	})

	response := request(
		t,
		api,
		http.MethodPost,
		"/items/42?page=3&tag=go&tag=http&since=2026-08-25T10:30:00Z&enabled=true",
		strings.NewReader(`{"name":"Amigo"}`),
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
			"X-Token":      "secret",
			"X-Trace":      "trace-1",
			"Cookie":       "session=session-1",
		},
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	output := decodeJSON[bindingOutput](t, response)
	if output.Input.ID != 42 || output.Input.Page != 3 || output.Input.Body.Name != "Amigo" {
		t.Fatalf("unexpected input: %+v", output.Input)
	}
	if !reflect.DeepEqual(output.Input.Tags, []string{"go", "http"}) {
		t.Fatalf("tags = %#v", output.Input.Tags)
	}
	if output.Input.Since == nil || !output.Input.Since.Equal(time.Date(2026, 8, 25, 10, 30, 0, 0, time.UTC)) {
		t.Fatalf("since = %v", output.Input.Since)
	}
	if !output.Input.Enabled || output.Input.Token != "secret" || output.Input.Session != "session-1" {
		t.Fatalf("unexpected parameters: %+v", output.Input)
	}
	if output.Input.Trace == nil || *output.Input.Trace != "trace-1" {
		t.Fatalf("trace = %v", output.Input.Trace)
	}
}

func TestBindQueryDefaultsAndOptionalPointers(t *testing.T) {
	type input struct {
		Page  int       `query:"page" default:"7"`
		Tags  *[]string `query:"tag"`
		Token *string   `header:"X-Token"`
		Body  *bindingBody
	}

	api := New()
	api.GET("/items", func(_ context.Context, input input) (input, error) {
		return input, nil
	})
	response := request(t, api, http.MethodGet, "/items", nil, nil)

	output := decodeJSON[input](t, response)
	if output.Page != 7 || output.Tags != nil || output.Token != nil || output.Body != nil {
		t.Fatalf("unexpected input: %+v", output)
	}
}

func TestBindUUIDTextUnmarshaler(t *testing.T) {
	type input struct {
		ID uuid.UUID `path:"id"`
	}

	want := uuid.MustParse("01941f29-7c00-7d00-a5c9-345f08c39fbd")
	api := New()
	api.GET("/items/{id}", func(_ context.Context, input input) (input, error) {
		return input, nil
	})
	response := request(t, api, http.MethodGet, "/items/"+want.String(), nil, nil)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if got := decodeJSON[input](t, response).ID; got != want {
		t.Fatalf("ID = %s, want %s", got, want)
	}
}

func TestBindingAccumulatesParameterErrors(t *testing.T) {
	type input struct {
		ID      int    `path:"id"`
		Page    int    `query:"page"`
		Token   string `header:"X-Token"`
		Session string `cookie:"session"`
	}

	api := New()
	api.GET("/items/{id}", func(_ context.Context, input input) (struct{}, error) {
		return struct{}{}, nil
	})
	response := request(t, api, http.MethodGet, "/items/nope?page=nope", nil, nil)
	problem := decodeJSON[Problem](t, response)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
	want := []FieldError{
		{Location: "path.id", Message: "expected integer"},
		{Location: "query.page", Message: "expected integer"},
		{Location: "header.X-Token", Message: "value is required"},
		{Location: "cookie.session", Message: "value is required"},
	}
	if !reflect.DeepEqual(problem.Errors, want) {
		t.Fatalf("errors = %#v, want %#v", problem.Errors, want)
	}
}

func TestBindBodyErrors(t *testing.T) {
	type nested struct {
		Age int `json:"age"`
	}
	type input struct {
		Body nested
	}

	newAPI := func() *API {
		api := New()
		api.POST("/items", func(_ context.Context, input input) (struct{}, error) {
			return struct{}{}, nil
		})
		return api
	}

	tests := []struct {
		name        string
		body        string
		contentType string
		status      int
		location    string
		message     string
	}{
		{name: "missing body", status: http.StatusBadRequest, location: "body", message: "value is required"},
		{name: "unsupported media type", body: `{}`, contentType: "text/plain", status: http.StatusUnsupportedMediaType},
		{name: "invalid JSON", body: `{`, contentType: "application/json", status: http.StatusBadRequest, location: "body", message: "invalid JSON"},
		{name: "wrong field type", body: `{"age":"old"}`, contentType: "application/json", status: http.StatusBadRequest, location: "body.age", message: "expected integer"},
		{name: "unknown member", body: `{"unknown":true}`, contentType: "application/json", status: http.StatusBadRequest, location: "body.unknown"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body io.Reader
			if test.body != "" {
				body = strings.NewReader(test.body)
			}
			response := request(t, newAPI(), http.MethodPost, "/items", body, map[string]string{"Content-Type": test.contentType})
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.status, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/problem+json" {
				t.Fatalf("Content-Type = %q", contentType)
			}
			problem := decodeJSON[Problem](t, response)
			if problem.Instance != "/items" {
				t.Fatalf("instance = %q", problem.Instance)
			}
			if test.location != "" {
				if len(problem.Errors) != 1 || problem.Errors[0].Location != test.location {
					t.Fatalf("errors = %#v", problem.Errors)
				}
				if test.message != "" && problem.Errors[0].Message != test.message {
					t.Fatalf("message = %q, want %q", problem.Errors[0].Message, test.message)
				}
			}
		})
	}
}
