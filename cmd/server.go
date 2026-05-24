package cmd

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/zcag/odak/config"
	"github.com/zcag/odak/internal/api"
	"github.com/zcag/odak/internal/store"
)

func runServer(args []string, webFS fs.FS) {
	fset := flag.NewFlagSet("server", flag.ExitOnError)
	cfg := config.LoadServer()

	file := fset.String("file", cfg.File, "path to todos.md")
	apiKey := fset.String("api-key", cfg.APIKey, "API bearer token")
	user := fset.String("user", cfg.User, "web UI username")
	password := fset.String("password", cfg.Password, "web UI password")
	port := fset.String("port", cfg.Port, "listen port")
	backupDir := fset.String("backup-dir", cfg.BackupDir, "backup directory")
	ui := fset.Bool("ui", false, "serve web UI")
	fset.Parse(args)

	if *file == "" {
		fmt.Fprintln(os.Stderr, "odak: --file is required")
		os.Exit(1)
	}
	if *apiKey == "" {
		fmt.Fprintln(os.Stderr, "odak: --api-key is required")
		os.Exit(1)
	}

	st := store.New(*file, *backupDir)
	h := api.New(st, api.Config{
		APIKey:   *apiKey,
		User:     *user,
		Password: *password,
		ServeUI:  *ui,
		WebFS:    webFS,
	})

	addr := ":" + *port
	log.Printf("odak listening on %s (ui=%v)", addr, *ui)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatal(err)
	}
}
