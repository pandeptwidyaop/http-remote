package assets

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/templates/* web/static/css/* web/static/js/*
var EmbeddedFiles embed.FS

// GetTemplatesFS returns the templates filesystem
func GetTemplatesFS() fs.FS {
	sub, _ := fs.Sub(EmbeddedFiles, "web/templates")
	return sub
}

// GetStaticFS returns the static files filesystem
func GetStaticFS() fs.FS {
	sub, _ := fs.Sub(EmbeddedFiles, "web/static")
	return sub
}

// GetStaticHTTPFS returns http.FileSystem for static files
func GetStaticHTTPFS() http.FileSystem {
	sub, _ := fs.Sub(EmbeddedFiles, "web/static")
	return http.FS(sub)
}
