package main

import (
	"embed"
	"io/fs"
	"os"

	"github.com/zcag/odak/cmd"
)

//go:embed web/dist
var webEmbed embed.FS

var Version = "dev"

func main() {
	webFS, _ := fs.Sub(webEmbed, "web/dist")
	cmd.Run(os.Args[1:], webFS, Version)
}
