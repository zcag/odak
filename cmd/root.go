package cmd

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/zcag/odak/internal/tui"
)

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

func Run(args []string, webFS fs.FS, version string) {
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-v" || args[0] == "version") {
		fmt.Println("odak", version)
		return
	}
	if len(args) == 0 {
		if !isTTY() {
			runList(nil)
			return
		}
		c := newClient()
		if err := tui.Run(c); err != nil {
			fmt.Fprintln(os.Stderr, "odak:", err)
			os.Exit(1)
		}
		return
	}

	// server mode: odak --server [--ui] [flags...]
	if args[0] == "--server" {
		runServer(args[1:], webFS)
		return
	}

	// bare filter token: `odak t:personal` is shorthand for `odak ls t:personal`
	if isFilterToken(args[0]) || args[0] == "--all" || args[0] == "-a" {
		runList(args)
		return
	}

	// client subcommands
	switch args[0] {
	case "tui", "ui":
		c := newClient()
		if err := tui.Run(c); err != nil {
			fmt.Fprintln(os.Stderr, "odak:", err)
			os.Exit(1)
		}
	case "list", "ls":
		runList(args[1:])
	case "add":
		runAdd(args[1:])
	case "done":
		runDone(args[1:])
	case "rm", "del", "delete":
		runRm(args[1:])
	case "move", "mv":
		runMove(args[1:])
	case "show":
		runShow(args[1:])
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "odak: unknown command %q\n\n", args[0])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`odak — file-backed todo service

Server:
  odak --server [flags]        start REST API server
  odak --server --ui [flags]   also serve web UI

  --file        path to todos.md (required, or ODAK_FILE)
  --api-key     bearer token (required, or ODAK_API_KEY)
  --user        web UI username (or ODAK_USER)
  --password    web UI password (or ODAK_PASSWORD)
  --port        listen port (default 8761, or ODAK_PORT)
  --backup-dir  backup directory (or ODAK_BACKUP_DIR)

  MCP OAuth (optional, env-only; lets Claude.ai / ChatGPT "Connect" via WorkOS
  AuthKit alongside the static API key — all unset ⇒ OAuth off):
  ODAK_OAUTH_ISSUER         AuthKit domain, e.g. https://x.authkit.app
  ODAK_MCP_RESOURCE         public /mcp URL (OAuth audience), e.g. https://odak.cagdas.io/mcp
  ODAK_OAUTH_ALLOWED_EMAIL  comma-separated email allowlist (single-user gate)

Client (reads ~/.config/odak/client or ODAK_ENDPOINT / ODAK_TOKEN):
  odak list [section] [t:TAG ...] [t:-TAG ...] [--all]   list todos (done hidden unless --all)
  odak t:TAG                           shorthand for: odak list t:TAG
  odak add <text> [--section S] [--tag T] [--urgent] [--deadline D] [--parent ID]
  odak done <id>                       toggle done
  odak rm <id>                         delete
  odak move <id> <section>             move to section
  odak show <id>                       show details

Sections: Focus Today Next Backlog Someday Recurring Inbox

MCP server (Model Context Protocol over Streamable HTTP):
  served by 'odak --server' at /mcp (bearer-authed, same API key)
  register: claude mcp add --transport http odak http://<host>:<port>/mcp --header "Authorization: Bearer <key>"
`)

}
