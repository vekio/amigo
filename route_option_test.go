package amigo

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestOpenAPIRouteOptionsSetOperationMetadata(t *testing.T) {
	route := newRoute(
		http.MethodPost,
		"/links",
		WithOperationID("createLink"),
		WithSummary("Create a shortened link"),
		WithDescription("Creates a stable shortened URL for the supplied target."),
		WithTags("Links", "Public"),
		WithTags("Links"),
	)

	if route.operationID != "createLink" {
		t.Errorf("operation ID = %q, want createLink", route.operationID)
	}
	if route.summary != "Create a shortened link" {
		t.Errorf("summary = %q, want Create a shortened link", route.summary)
	}
	if route.description != "Creates a stable shortened URL for the supplied target." {
		t.Errorf("description = %q", route.description)
	}
	wantTags := []string{"Links", "Public"}
	if !reflect.DeepEqual(route.tags, wantTags) {
		t.Errorf("tags = %#v, want %#v", route.tags, wantTags)
	}
}

func TestOpenAPIRouteOptionsRejectEmptyValues(t *testing.T) {
	tests := []struct {
		name   string
		option func() RouteOption
	}{
		{name: "operation ID", option: func() RouteOption { return WithOperationID("  ") }},
		{name: "summary", option: func() RouteOption { return WithSummary("") }},
		{name: "description", option: func() RouteOption { return WithDescription("\t") }},
		{name: "no tags", option: func() RouteOption { return WithTags() }},
		{name: "empty tag", option: func() RouteOption { return WithTags("Links", " ") }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertPanics(t, func() { test.option() })
		})
	}
}

func TestWithErrorMappingAddsRouteMapping(t *testing.T) {
	target := errors.New("conflict")
	route := newRoute(http.MethodPost, "/things", WithErrorMapping(target, http.StatusConflict, "thing already exists"))

	if len(route.errors) != 1 ||
		!errors.Is(route.errors[0].target, target) ||
		route.errors[0].status != http.StatusConflict ||
		route.errors[0].publicDetail != "thing already exists" {
		t.Errorf("errors = %#v", route.errors)
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
	if route.operationID != "get-things" {
		t.Errorf("operation ID = %q, want get-things", route.operationID)
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

func TestWithStatusAcceptsRedirectStatus(t *testing.T) {
	route := newRoute(http.MethodGet, "/old", WithStatus(http.StatusTemporaryRedirect))
	if route.status != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", route.status, http.StatusTemporaryRedirect)
	}
}

func TestWithStatusRejectsErrorOrInformationalStatus(t *testing.T) {
	for _, status := range []int{http.StatusContinue, http.StatusBadRequest} {
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
