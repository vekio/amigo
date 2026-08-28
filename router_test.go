package amigo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRouterBuildsNestedRoutePath(t *testing.T) {
	api := New()
	admin := api.Group("/api").Group("/admin")
	admin.GET("/things/{id}", func(_ context.Context, input struct {
		ID string `path:"id" json:"-"`
	}) (struct {
		ID string `json:"id"`
	}, error) {
		return struct {
			ID string `json:"id"`
		}{ID: input.ID}, nil
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/things/42", nil)
	api.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != `{"id":"42"}` {
		t.Errorf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestRouterInheritsMiddlewareInHierarchyOrder(t *testing.T) {
	order := []string{}
	middleware := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				order = append(order, name+":before")
				next.ServeHTTP(w, request)
				order = append(order, name+":after")
			})
		}
	}

	api := New()
	parent := api.Group("/parent", middleware("parent"))
	child := parent.Group("/child", middleware("child"))
	child.GET("", func(context.Context, struct{}) (struct{}, error) {
		order = append(order, "endpoint")
		return struct{}{}, nil
	}, WithMiddleware(middleware("route")))

	api.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/parent/child", nil),
	)

	want := []string{
		"parent:before",
		"child:before",
		"route:before",
		"endpoint",
		"route:after",
		"child:after",
		"parent:after",
	}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %#v, want %#v", order, want)
	}
}

func TestRouterAllowsSamePrefixWithDifferentRoutes(t *testing.T) {
	api := New()
	first := api.Group("/things")
	second := api.Group("/things")
	first.GET("/first", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})
	second.GET("/second", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})

	for _, path := range []string{"/things/first", "/things/second"} {
		response := httptest.NewRecorder()
		api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Errorf("%s status = %d, want %d", path, response.Code, http.StatusOK)
		}
	}
}

func TestTypedMutationMethodsRegisterOnAPIAndRouter(t *testing.T) {
	type output struct {
		Method string `json:"method"`
	}
	endpoint := func(method string) EndpointFunc[struct{}, output] {
		return func(context.Context, struct{}) (output, error) {
			return output{Method: method}, nil
		}
	}

	api := New()
	api.PUT("/root-put", endpoint(http.MethodPut))
	api.PATCH("/root-patch", endpoint(http.MethodPatch))
	api.DELETE("/root-delete", endpoint(http.MethodDelete))

	things := api.Group("/things")
	things.PUT("/put", endpoint(http.MethodPut))
	things.PATCH("/patch", endpoint(http.MethodPatch))
	things.DELETE("/delete", endpoint(http.MethodDelete))

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPut, path: "/root-put"},
		{method: http.MethodPatch, path: "/root-patch"},
		{method: http.MethodDelete, path: "/root-delete"},
		{method: http.MethodPut, path: "/things/put"},
		{method: http.MethodPatch, path: "/things/patch"},
		{method: http.MethodDelete, path: "/things/delete"},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, nil)
			api.ServeHTTP(response, request)

			wantBody := `{"method":"` + test.method + `"}`
			if response.Code != http.StatusOK || response.Body.String() != wantBody {
				t.Errorf("status = %d, body = %s, want body = %s", response.Code, response.Body.String(), wantBody)
			}
		})
	}
}

func TestServeMuxRejectsDuplicateRoutePattern(t *testing.T) {
	api := New()
	first := api.Group("/things")
	second := api.Group("/things")
	endpoint := func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}
	first.GET("", endpoint)

	defer func() {
		if recover() == nil {
			t.Error("duplicate route registration did not panic")
		}
	}()

	second.GET("", endpoint)
}

func TestRouterRejectsInvalidPrefix(t *testing.T) {
	tests := []string{"", "links", "/links/"}
	for _, prefix := range tests {
		t.Run(prefix, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("Group() did not panic")
				}
			}()
			New().Group(prefix)
		})
	}
}

func TestRouterRejectsRouteWithoutLeadingSlash(t *testing.T) {
	assertPanics(t, func() {
		New().GET("things", func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, nil
		})
	})
}

func TestRootRouterEmptyPathRegistersSlash(t *testing.T) {
	api := New()
	api.GET("", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})

	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestRouterRejectsNilEndpoints(t *testing.T) {
	t.Run("typed", func(t *testing.T) {
		var endpoint EndpointFunc[struct{}, struct{}]
		assertPanics(t, func() { New().GET("/things", endpoint) })
	})

	t.Run("raw", func(t *testing.T) {
		assertPanics(t, func() { New().RAW(http.MethodGet, "/things", nil) })
	})
}

func TestGroupRejectsNilMiddleware(t *testing.T) {
	assertPanics(t, func() { New().Group("/things", nil) })
}
