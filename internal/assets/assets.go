// Package assets provides embedded web templates and static files.
package assets

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/templates/* web/static/css/* web/static/js/* web/dist

// EmbeddedFiles contains all embedded web assets (templates, static files, and React SPA dist).
var EmbeddedFiles embed.FS

// GetTemplatesFS returns the templates filesystem.
func GetTemplatesFS() fs.FS {
	sub, err := fs.Sub(EmbeddedFiles, "web/templates")
	if err != nil {
		panic("failed to access embedded templates: " + err.Error())
	}
	return sub
}

// GetStaticFS returns the static files filesystem.
func GetStaticFS() fs.FS {
	sub, err := fs.Sub(EmbeddedFiles, "web/static")
	if err != nil {
		panic("failed to access embedded static files: " + err.Error())
	}
	return sub
}

// GetStaticHTTPFS returns http.FileSystem for static files.
func GetStaticHTTPFS() http.FileSystem {
	sub, err := fs.Sub(EmbeddedFiles, "web/static")
	if err != nil {
		panic("failed to access embedded static files: " + err.Error())
	}
	return http.FS(sub)
}
