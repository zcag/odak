package main

import (
	"os"

	"github.com/zcag/odak/cmd"
)

var Version = "dev"

func main() {
	cmd.Run(os.Args[1:], getWebFS(), Version)
}
