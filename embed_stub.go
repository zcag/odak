//go:build !ui

package main

import "io/fs"

func getWebFS() fs.FS { return nil }
