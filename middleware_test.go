package amigo

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestApplyMiddlewaresUsesDeclarationOrder(t *testing.T) {
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
	handler := applyMiddlewares(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			order = append(order, "handler")
		}),
		[]Middleware{middleware("first"), middleware("second")},
	)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"first:before", "second:before", "handler", "second:after", "first:after"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %#v, want %#v", order, want)
	}
}

func TestApplyMiddlewaresRejectsNilHandlerResult(t *testing.T) {
	middleware := func(http.Handler) http.Handler { return nil }
	assertPanics(t, func() {
		applyMiddlewares(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), []Middleware{middleware})
	})
}
