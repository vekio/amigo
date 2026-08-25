package amigo

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestPathParametersMatchInput(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		input InputMetadata
		err   string
	}{
		{
			name: "matching",
			path: "/users/{id}",
			input: InputMetadata{Parameters: []ParameterMetadata{
				{Name: "id", Source: ParameterPath},
			}},
		},
		{
			name: "input missing from route",
			path: "/users/{id}",
			input: InputMetadata{Parameters: []ParameterMetadata{
				{Name: "user_id", Source: ParameterPath},
			}},
			err: `path parameter "user_id" is not declared in the route`,
		},
		{
			name: "route missing from input",
			path: "/users/{id}",
			err:  `route path parameter "id" has no matching input field`,
		},
		{
			name: "duplicate input",
			path: "/users/{id}",
			input: InputMetadata{Parameters: []ParameterMetadata{
				{Name: "id", Source: ParameterPath},
				{Name: "id", Source: ParameterPath},
			}},
			err: `path parameter "id" is declared more than once`,
		},
		{
			name: "duplicate wildcard",
			path: "/{id}/{id}",
			err:  `route path parameter "id" is declared more than once`,
		},
		{
			name: "catch all not final",
			path: "/files/{path...}/info",
			err:  `multi-segment wildcard "path" must be the final segment`,
		},
		{
			name: "missing leading slash",
			path: "users/{id}",
			err:  "path must start with a slash",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePathParameters(test.path, test.input)
			if test.err == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.err) {
				t.Fatalf("error=%v, want %q", err, test.err)
			}
		})
	}
}

func TestCompileRejectsMismatchedPathParameters(t *testing.T) {
	api := New()
	api.GET("/users/{id}", func(_ context.Context, _ struct {
		UserID int `path:"user_id"`
	}) (struct{}, error) {
		return struct{}{}, nil
	})

	handler, err := api.Compile()
	if handler != nil || err == nil || !strings.Contains(err.Error(), `GET /users/{id}: path parameter "user_id"`) {
		t.Fatalf("handler=%v error=%v", handler, err)
	}
	if _, secondErr := api.Compile(); secondErr == nil || secondErr.Error() != err.Error() {
		t.Fatalf("second compile error=%v, want %v", secondErr, err)
	}
}

func TestCatchAllPathParameter(t *testing.T) {
	api := New()
	api.GET("/files/{path...}", func(_ context.Context, input struct {
		Path string `path:"path"`
	}) (string, error) {
		return input.Path, nil
	})

	response := request(t, api, http.MethodGet, "/files/css/app.css", nil, nil)
	if response.Code != http.StatusOK || response.Body.String() != `"css/app.css"` {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}
