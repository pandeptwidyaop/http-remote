// Package assets provides embedded web templates and static files.
package assets

import (
	"embed"
)

//go:embed web/dist
var EmbeddedFiles embed.FS
