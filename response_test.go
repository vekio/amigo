package amigo

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestRouteSuccessStatus(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		api := New()
		api.POST("/items", func(_ context.Context, _ emptyInput) (map[string]bool, error) {
			return map[string]bool{"created": true}, nil
		}, Status(http.StatusCreated))

		response := request(t, api, http.MethodPost, "/items", nil, nil)
		if response.Code != http.StatusCreated || response.Body.String() != `{"created":true}` {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
		operations := api.Operations()
		if len(operations) != 1 || operations[0].Output.Status != http.StatusCreated {
			t.Fatalf("operations=%+v", operations)
		}
	})

	t.Run("no content skips encoding", func(t *testing.T) {
		api := New()
		api.DELETE("/items", func(_ context.Context, _ emptyInput) (func(), error) {
			return func() {}, nil
		}, Status(http.StatusNoContent))

		response := request(t, api, http.MethodDelete, "/items", nil, nil)
		if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
		if contentType := response.Header().Get("Content-Type"); contentType != "" {
			t.Fatalf("content type=%q", contentType)
		}
	})
}

func TestCompileRejectsInvalidSuccessStatus(t *testing.T) {
	for _, status := range []int{http.StatusContinue, http.StatusPermanentRedirect, http.StatusBadRequest} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			api := New()
			api.GET("/items", func(_ context.Context, _ emptyInput) (struct{}, error) {
				return struct{}{}, nil
			}, Status(status))

			if _, err := api.Compile(); err == nil || !strings.Contains(err.Error(), "success status must be between 200 and 299") {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestMaxBodyBytes(t *testing.T) {
	type input struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	t.Run("too large", func(t *testing.T) {
		api := New()
		api.SetMaxBodyBytes(16)
		called := false
		api.POST("/items", func(_ context.Context, _ input) (struct{}, error) {
			called = true
			return struct{}{}, nil
		})

		response := request(
			t,
			api,
			http.MethodPost,
			"/items",
			strings.NewReader(`{"name":"a value longer than the limit"}`),
			map[string]string{"Content-Type": "application/json"},
		)
		problem := decodeJSON[Problem](t, response)
		if response.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status=%d problem=%+v", response.Code, problem)
		}
		if problem.Detail != "request body exceeds the maximum allowed size" || called {
			t.Fatalf("problem=%+v called=%t", problem, called)
		}
	})

	t.Run("within limit", func(t *testing.T) {
		api := New()
		api.SetMaxBodyBytes(64)
		api.POST("/items", func(_ context.Context, input input) (string, error) {
			return input.Body.Name, nil
		})

		response := request(
			t,
			api,
			http.MethodPost,
			"/items",
			strings.NewReader(`{"name":"amigo"}`),
			map[string]string{"Content-Type": "application/json"},
		)
		if response.Code != http.StatusOK || response.Body.String() != `"amigo"` {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
	})

	requirePanicContains(t, "cannot be negative", func() {
		New().SetMaxBodyBytes(-1)
	})
}

func TestCompileReturnsServeMuxErrors(t *testing.T) {
	api := New()
	handler := func(_ context.Context, _ emptyInput) (struct{}, error) {
		return struct{}{}, nil
	}
	api.GET("/items", handler)
	api.GET("/items", handler)

	compiled, err := api.Compile()
	if compiled != nil || err == nil || !strings.Contains(err.Error(), "conflicts with pattern") {
		t.Fatalf("handler=%v error=%v", compiled, err)
	}
	requirePanicContains(t, "conflicts with pattern", func() {
		api.Handler()
	})
}
