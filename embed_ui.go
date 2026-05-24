//go:build ui

package main

import (
	"embed"
	"io/fs"
)

//go:embed web/dist
var webEmbed embed.FS

func getWebFS() fs.FS {
	f, _ := fs.Sub(webEmbed, "web/dist")
	return f
}
