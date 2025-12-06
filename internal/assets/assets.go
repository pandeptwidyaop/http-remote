package assets

import (
	"embed"
)

//go:embed web/dist
var EmbeddedFiles embed.FS
