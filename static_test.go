package amigo

import (
	"io/fs"
	"net/http"
	"testing"
	"testing/fstest"
)

func TestStaticServesFileSystem(t *testing.T) {
	api := New()
	api.Static("/static", fstest.MapFS{
		"app.css": &fstest.MapFile{Data: []byte("body { color: navy; }")},
	})

	response := request(t, api, http.MethodGet, "/static/app.css", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if got := response.Body.String(); got != "body { color: navy; }" {
		t.Fatalf("body=%q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/css; charset=utf-8" {
		t.Fatalf("content type=%q", got)
	}

	head := request(t, api, http.MethodHead, "/static/app.css", nil, nil)
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD status=%d body=%q", head.Code, head.Body.String())
	}

	post := request(t, api, http.MethodPost, "/static/app.css", nil, nil)
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d", post.Code)
	}
}

func TestStaticInheritsRouterPrefixAndMiddleware(t *testing.T) {
	api := New()
	router := NewRouter(Prefix("/web"))
	router.Use(func(w http.ResponseWriter, req *http.Request, next http.Handler) {
		w.Header().Set("X-Amigo-Middleware", "applied")
		next.ServeHTTP(w, req)
	})
	router.Static("/assets", fstest.MapFS{
		"message.txt": &fstest.MapFile{Data: []byte("hello")},
	})
	api.Include(router)

	response := request(t, api, http.MethodGet, "/web/assets/message.txt", nil, nil)
	if response.Code != http.StatusOK || response.Body.String() != "hello" {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if got := response.Header().Get("X-Amigo-Middleware"); got != "applied" {
		t.Fatalf("middleware header=%q", got)
	}
	if operations := api.Operations(); len(operations) != 0 {
		t.Fatalf("static mount created operations: %#v", operations)
	}
}

func TestStaticConfiguration(t *testing.T) {
	t.Run("root redirect", func(t *testing.T) {
		api := New()
		api.Static("/static", fstest.MapFS{})

		response := request(t, api, http.MethodGet, "/static", nil, nil)
		if response.Code != http.StatusTemporaryRedirect {
			t.Fatalf("status=%d", response.Code)
		}
		if got := response.Header().Get("Location"); got != "/static/" {
			t.Fatalf("location=%q", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		requirePanicContains(t, "static path cannot be empty", func() {
			New().Static("", fstest.MapFS{})
		})
		requirePanicContains(t, "file system is nil", func() {
			New().Static("/static", nil)
		})
	})

	t.Run("frozen", func(t *testing.T) {
		api := New()
		api.Handler()
		requirePanicContains(t, "API has been built", func() {
			api.Static("/static", fs.FS(fstest.MapFS{}))
		})
	})
}
