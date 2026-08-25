package amigo

import (
	"fmt"
	"io/fs"
)

// staticMount describes a file system mounted relative to a Router.
type staticMount struct {
	path string
	root fs.FS
}

// Static serves the contents of root below path. Files opened by root must
// implement io.Seeker, as required by http.FileServerFS.
func (api *API) Static(path string, root fs.FS) {
	api.root.Static(path, root)
}

// Static serves the contents of root below path. Files opened by root must
// implement io.Seeker, as required by http.FileServerFS.
func (router *Router) Static(path string, root fs.FS) {
	router.assertMutable()
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
