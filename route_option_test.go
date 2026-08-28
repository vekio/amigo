package amigo

import (
	"errors"
	"net/http"
	"testing"
)

func TestWithErrorMappingAddsRouteMapping(t *testing.T) {
	target := errors.New("conflict")
	route := newRoute(http.MethodPost, "/things", WithErrorMapping(target, http.StatusConflict, "thing already exists"))

	if len(route.errorMappings) != 1 ||
		!errors.Is(route.errorMappings[0].target, target) ||
		route.errorMappings[0].status != http.StatusConflict ||
		route.errorMappings[0].publicDetail != "thing already exists" {
		t.Errorf("errorMappings = %#v", route.errorMappings)
	}
}

func TestRouteDefaults(t *testing.T) {
	route := newRoute(http.MethodGet, "/things")

	if route.status != http.StatusOK {
		t.Errorf("status = %d, want %d", route.status, http.StatusOK)
	}
	if route.maxBodyBytes != defaultMaxBodyBytes {
		t.Errorf("maxBodyBytes = %d, want %d", route.maxBodyBytes, defaultMaxBodyBytes)
	}
}

func TestRouteOptionsOverrideDefaults(t *testing.T) {
	route := newRoute(
		http.MethodPost,
		"/things",
		WithStatus(http.StatusCreated),
		WithMaxBodyBytes(512),
	)

	if route.status != http.StatusCreated {
		t.Errorf("status = %d, want %d", route.status, http.StatusCreated)
	}
	if route.maxBodyBytes != 512 {
		t.Errorf("maxBodyBytes = %d, want %d", route.maxBodyBytes, 512)
	}
}

func TestWithMiddlewareAddsRouteMiddleware(t *testing.T) {
	middleware := func(next http.Handler) http.Handler { return next }
	route := newRoute(http.MethodGet, "/things", WithMiddleware(middleware))

	if len(route.middlewares) != 1 {
		t.Errorf("middlewares = %d, want %d", len(route.middlewares), 1)
	}
}

func TestWithMiddlewareRejectsNilMiddleware(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("WithMiddleware() did not panic")
		}
	}()

	_ = WithMiddleware(nil)
}

func TestWithErrorMappingRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		target error
		status int
	}{
		{name: "nil error", status: http.StatusBadRequest},
		{name: "success status", target: errors.New("failure"), status: http.StatusOK},
		{name: "invalid status", target: errors.New("failure"), status: 600},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("WithErrorMapping() did not panic")
				}
			}()
			_ = WithErrorMapping(test.target, test.status, "public failure")
		})
	}
}

func TestWithStatusRejectsNonSuccessStatus(t *testing.T) {
	for _, status := range []int{http.StatusContinue, http.StatusMultipleChoices} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			assertPanics(t, func() { WithStatus(status) })
		})
	}
}

func TestWithMaxBodyBytesRejectsNegativeLimit(t *testing.T) {
	assertPanics(t, func() { WithMaxBodyBytes(-1) })
}

func TestNewRouteRejectsNilOption(t *testing.T) {
	assertPanics(t, func() { newRoute(http.MethodGet, "/things", nil) })
}
