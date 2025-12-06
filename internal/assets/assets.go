// Package assets provides embedded web templates and static files.
package assets

import (
	"embed"
)

//go:embed web/dist
// EmbeddedFiles contains the embedded web UI assets (HTML, CSS, JS).
var EmbeddedFiles embed.FS
