package amigo

import (
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"strings"
)

// staticMount describes a file system mounted relative to a Router.
type staticMount struct {
	path       string
	root       fs.FS
	middleware []Middleware
}

// Static serves the contents of root below path. Files opened by root must
// implement io.Seeker, as required by http.FileServerFS.
func (api *API) Static(path string, root fs.FS) {
	api.assertMutable("mount static files")
	api.root.Static(path, root)
}

// Static serves the contents of root below path. Files opened by root must
// implement io.Seeker, as required by http.FileServerFS.
func (router *Router) Static(path string, root fs.FS) {
	if path == "" {
		panic("amigo: static path cannot be empty")
	}
	if root == nil {
		panic(fmt.Sprintf("amigo: static %s: file system is nil", path))
	}

	router.staticMounts = append(router.staticMounts, staticMount{
		path: path,
		root: root,
	})
}

func (mount staticMount) httpHandler(middleware []Middleware) (string, http.Handler) {
	prefix := strings.TrimRight(mount.path, "/")
	pattern := http.MethodGet + " " + prefix + "/"
	handler := http.StripPrefix(prefix, http.FileServerFS(mount.root))
	return pattern, applyMiddleware(handler, slices.Concat(middleware, mount.middleware))
}
