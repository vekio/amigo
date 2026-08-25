package amigo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

type emptyInput struct{}

func TestHTTPVerbs(t *testing.T) {
	api := New()
	handler := func(_ context.Context, _ emptyInput) (map[string]bool, error) {
		return map[string]bool{"ok": true}, nil
	}
	api.GET("/get", handler)
	api.POST("/post", handler)
	api.PUT("/put", handler)
	api.PATCH("/patch", handler)
	api.DELETE("/delete", handler)
	router := NewRouter(Prefix("/router"))
	router.GET("/get", handler)
	router.POST("/post", handler)
	router.PUT("/put", handler)
	router.PATCH("/patch", handler)
	router.DELETE("/delete", handler)
	api.Include(router)

	for _, test := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/get"},
		{http.MethodPost, "/post"},
		{http.MethodPut, "/put"},
		{http.MethodPatch, "/patch"},
		{http.MethodDelete, "/delete"},
		{http.MethodGet, "/router/get"},
		{http.MethodPost, "/router/post"},
		{http.MethodPut, "/router/put"},
		{http.MethodPatch, "/router/patch"},
		{http.MethodDelete, "/router/delete"},
	} {
		t.Run(test.method, func(t *testing.T) {
			response := request(t, api, test.method, test.path, nil, nil)
			if response.Code != http.StatusOK || response.Body.String() != `{"ok":true}` {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}

	response := request(t, api, http.MethodPost, "/get", nil, nil)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method mismatch status = %d", response.Code)
	}
}

func TestNestedRoutersInheritPrefixTagsAndMiddleware(t *testing.T) {
	var calls []string
	middleware := func(name string) Middleware {
		return func(w http.ResponseWriter, req *http.Request, next http.Handler) {
			calls = append(calls, name+":before")
			next.ServeHTTP(w, req)
			calls = append(calls, name+":after")
		}
	}

	api := New()
	api.Use(middleware("root"))
	parent := NewRouter(Prefix("/api/"), Tags("api"))
	parent.Use(middleware("parent"))
	users := NewRouter(Prefix("/users"), Tags("users"))
	users.Use(middleware("users"))
	users.GET("/{id}", func(_ context.Context, _ struct {
		ID int `path:"id"`
	}) (struct {
		ID int `json:"id"`
	}, error) {
		calls = append(calls, "handler")
		return struct {
			ID int `json:"id"`
		}{ID: 1}, nil
	}, Tags("read"))
	parent.Include(users)
	api.Include(parent)

	response := request(t, api, http.MethodGet, "/api/users/1", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	wantCalls := []string{
		"root:before", "parent:before", "users:before", "handler",
		"users:after", "parent:after", "root:after",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls=%#v, want %#v", calls, wantCalls)
	}

	operation := api.Operations()[0]
	if operation.Method != http.MethodGet || operation.Path != "/api/users/{id}" {
		t.Fatalf("operation=%+v", operation)
	}
	if !reflect.DeepEqual(operation.Tags, []string{"api", "users", "read"}) {
		t.Fatalf("tags=%#v", operation.Tags)
	}
}

func TestAPIFreezesAfterBuild(t *testing.T) {
	api := New()
	child := NewRouter()
	child.GET("/items", func(_ context.Context, _ emptyInput) (struct{}, error) { return struct{}{}, nil })
	api.Include(child)
	api.Handler()

	tests := []struct {
		name string
		run  func()
	}{
		{name: "API route", run: func() {
			api.GET("/late", func(_ context.Context, _ emptyInput) (struct{}, error) { return struct{}{}, nil })
		}},
		{name: "API include", run: func() { api.Include(NewRouter()) }},
		{name: "API middleware", run: func() { api.Use(func(http.ResponseWriter, *http.Request, http.Handler) {}) }},
		{name: "error handler", run: func() { api.SetErrorHandler(nil) }},
		{name: "validator", run: func() { api.Validator("late", func(int) error { return nil }) }},
		{name: "maximum body size", run: func() { api.SetMaxBodyBytes(1024) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requirePanicContains(t, "API has been built", test.run)
		})
	}
}

func TestRouterIncludeUsesSnapshot(t *testing.T) {
	var lateMiddlewareCalls int
	child := NewRouter(Prefix("/children"), Tags("children"))
	child.GET("/first", func(_ context.Context, _ emptyInput) (struct{}, error) {
		return struct{}{}, nil
	})

	parent := NewRouter(Prefix("/api"), Tags("api"))
	parent.Include(child)

	child.GET("/late", func(_ context.Context, _ emptyInput) (struct{}, error) {
		return struct{}{}, nil
	})
	child.Use(func(w http.ResponseWriter, req *http.Request, next http.Handler) {
		lateMiddlewareCalls++
		next.ServeHTTP(w, req)
	})

	api := New()
	api.Include(parent)

	parent.GET("/also-late", func(_ context.Context, _ emptyInput) (struct{}, error) {
		return struct{}{}, nil
	})
	parent.Include(NewRouter(Prefix("/late-router")))
	parent.Use(func(http.ResponseWriter, *http.Request, http.Handler) {
		lateMiddlewareCalls++
	})

	response := request(t, api, http.MethodGet, "/api/children/first", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("snapshot route status=%d body=%s", response.Code, response.Body.String())
	}
	if lateMiddlewareCalls != 0 {
		t.Fatalf("late middleware calls=%d", lateMiddlewareCalls)
	}
	for _, path := range []string{"/api/children/late", "/api/also-late"} {
		response := request(t, api, http.MethodGet, path, nil, nil)
		if response.Code != http.StatusNotFound {
			t.Fatalf("late route %s status=%d", path, response.Code)
		}
	}

	operations := api.Operations()
	if len(operations) != 1 {
		t.Fatalf("operations=%d", len(operations))
	}
	if !reflect.DeepEqual(operations[0].Tags, []string{"api", "children"}) {
		t.Fatalf("tags=%#v", operations[0].Tags)
	}
}

func TestAPIConfigurationAfterIncludeIsAppliedAtBuild(t *testing.T) {
	type input struct {
		Page int `query:"page" validate:"positive"`
	}

	router := NewRouter(Prefix("/api"))
	router.GET("/items", func(_ context.Context, input input) (int, error) {
		return input.Page, nil
	})

	api := New()
	api.Include(router)
	api.Validator("positive", func(value int) error {
		if value < 1 {
			return errors.New("must be positive")
		}
		return nil
	})
	api.Use(func(w http.ResponseWriter, req *http.Request, next http.Handler) {
		w.Header().Set("X-Configured-After-Include", "true")
		next.ServeHTTP(w, req)
	})

	response := request(t, api, http.MethodGet, "/api/items?page=2", nil, nil)
	if response.Code != http.StatusOK || response.Body.String() != "2" {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if got := response.Header().Get("X-Configured-After-Include"); got != "true" {
		t.Fatalf("middleware header=%q", got)
	}
}

func TestHandlerDoesNotExposeServeMux(t *testing.T) {
	api := New()
	if _, exposed := api.Handler().(*http.ServeMux); exposed {
		t.Fatal("Handler exposed the mutable ServeMux")
	}
}

func TestRouterConfigurationRejectsNilValues(t *testing.T) {
	requirePanicContains(t, "cannot include a nil router", func() {
		NewRouter().Include(nil)
	})
	requirePanicContains(t, "middleware at index 1 is nil", func() {
		NewRouter().Use(func(http.ResponseWriter, *http.Request, http.Handler) {}, nil)
	})

	var handler Handler[emptyInput, struct{}]
	requirePanicContains(t, "handler is nil", func() {
		New().GET("/items", handler)
	})
}

func TestRouterCycleAndDuplicatePatternsFailBuild(t *testing.T) {
	t.Run("cycle", func(t *testing.T) {
		first := NewRouter(Prefix("/first"))
		second := NewRouter(Prefix("/second"))
		first.Include(second)
		requirePanicContains(t, "router inclusion cycle detected", func() {
			second.Include(first)
		})
		requirePanicContains(t, "router inclusion cycle detected", func() {
			first.Include(first)
		})
	})

	t.Run("duplicate pattern", func(t *testing.T) {
		api := New()
		handler := func(_ context.Context, _ emptyInput) (struct{}, error) { return struct{}{}, nil }
		api.GET("/items", handler)
		api.GET("/items", handler)
		requirePanicContains(t, "conflicts with pattern", func() {
			api.Handler()
		})
		requirePanicContains(t, "conflicts with pattern", func() {
			api.Handler()
		})
	})
}

func TestOperationsReturnsDeepSnapshots(t *testing.T) {
	type input struct {
		Page int `query:"page" default:"1" validate:"positive"`
		Body struct {
			Name string `json:"name" validate:"required"`
		}
	}

	api := New()
	api.Validator("positive", func(int) error { return nil })
	api.Validator("required", func(string) error { return nil })
	api.POST("/items", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil }, Tags("items"))

	first := api.Operations()
	first[0].Tags[0] = "changed"
	first[0].Input.Parameters[0].Name = "changed"
	*first[0].Input.Parameters[0].Default = "99"
	first[0].Input.Validations[0].Location = "changed"
	first[0].Input.Validations[0].Rules[0] = "changed"
	first[0].Input.Body.Required = false

	second := api.Operations()
	if second[0].Tags[0] != "items" || second[0].Input.Parameters[0].Name != "page" {
		t.Fatalf("snapshot leaked: %+v", second[0])
	}
	if got := *second[0].Input.Parameters[0].Default; got != "1" {
		t.Fatalf("default=%q", got)
	}
	if second[0].Input.Validations[0].Location != "query.page" || second[0].Input.Validations[0].Rules[0] != "positive" {
		t.Fatalf("validations=%#v", second[0].Input.Validations)
	}
	if !second[0].Input.Body.Required {
		t.Fatal("body metadata mutation leaked")
	}
}

func TestAPIHandlesConcurrentRequestsAfterConfiguration(t *testing.T) {
	const requests = 100
	var calls atomic.Int64
	api := New()
	api.GET("/items/{id}", func(_ context.Context, input struct {
		ID int `path:"id"`
	}) (struct {
		ID int `json:"id"`
	}, error) {
		calls.Add(1)
		return struct {
			ID int `json:"id"`
		}{ID: input.ID}, nil
	})

	var wait sync.WaitGroup
	errors := make(chan int, requests)
	for range requests {
		wait.Add(1)
		go func() {
			defer wait.Done()
			response := httptest.NewRecorder()
			api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/items/42", nil))
			if response.Code != http.StatusOK {
				errors <- response.Code
			}
		}()
	}
	wait.Wait()
	close(errors)
	for status := range errors {
		t.Errorf("unexpected status %d", status)
	}
	if got := calls.Load(); got != requests {
		t.Fatalf("handler calls=%d, want %d", got, requests)
	}
}
