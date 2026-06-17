# odak — agent notes

## Shipping a change

After any code change, don't stop at commit/push — the binary in use must be updated too:

- `make build` — local binary → `~/.local/bin/odak` (CLI/TUI/MCP). **Always run this after a change**, even for CLI-only edits; otherwise the installed `odak` stays stale.
- `make deploy` — cross-compile + rsync to archer (server, systemd restart) + marko, and install locally. Run when the server or remote hosts are affected.

Atomic install: the Makefile copies to `odak.new` then `mv -f`s into place, because a running `odak` process (the long-lived `odak --server`, which now also serves MCP over HTTP, or an open TUI) holds the binary open and a plain `cp` fails with `ETXTBSY` ("text file busy").

So the full cycle for an odak change is: **commit → push → `make build`/`make deploy` (install the binary)**.
