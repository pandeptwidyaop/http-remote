// Package assets provides embedded web templates and static files.
package assets

import (
	"embed"
)

// EmbeddedFiles contains the embedded web UI assets (HTML, CSS, JS).
//
//go:embed web/dist
var EmbeddedFiles embed.FS
