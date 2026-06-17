package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	apiclient "github.com/zcag/odak/internal/client"
	"github.com/zcag/odak/internal/mcp"
	"github.com/zcag/odak/internal/store"
)

type Config struct {
	APIKey   string
	User     string
	Password string
	ServeUI  bool
	WebFS    fs.FS
	// MCPClient, if set, mounts the MCP endpoint at /mcp. It is a loopback API
	// client so MCP tool calls reuse the REST handlers' filtering/validation.
	MCPClient *apiclient.Client
	// OAuth, if set, lets /mcp also accept WorkOS AuthKit JWTs (the Claude.ai /
	// ChatGPT "Connect" flow) alongside the static APIKey, and serves the RFC 9728
	// Protected Resource Metadata. Nil ⇒ /mcp stays APIKey-only.
	OAuth *MCPOAuth
}

type handler struct {
	store *store.Store
	cfg   Config
	hub   *Hub
}

func New(st *store.Store, cfg Config) http.Handler {
	hub := newHub()
	go hub.Run()
	h := &handler{store: st, cfg: cfg, hub: hub}

	// broadcast to WS clients when file changes externally
	st.WatchFile(func() { hub.Broadcast([]byte(`{"type":"reload"}`)) })
	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("GET /todos", h.auth(h.list))
	mux.HandleFunc("POST /todos", h.auth(h.create))
	mux.HandleFunc("GET /todos/{id}", h.auth(h.get))
	mux.HandleFunc("PATCH /todos/{id}", h.auth(h.update))
	mux.HandleFunc("DELETE /todos/{id}", h.auth(h.delete))
	mux.HandleFunc("PATCH /todos/{id}/done", h.auth(h.toggleDone))
	mux.HandleFunc("POST /todos/{id}/move", h.auth(h.move))
	mux.HandleFunc("POST /todos/reorder", h.auth(h.reorder))
	mux.HandleFunc("GET /sections", h.auth(h.sections))
	mux.HandleFunc("GET /raw", h.auth(h.getRaw))
	mux.HandleFunc("PUT /raw", h.auth(h.putRaw))
	mux.HandleFunc("GET /ws", h.ws)

	if cfg.MCPClient != nil {
		mux.HandleFunc("/mcp", h.mcpAuth(mcp.Handler(cfg.MCPClient)))
		// RFC 9728 discovery for the "Connect" flow. Public + static; Claude probes
		// both the root and the path-scoped variant. Only mounted when OAuth is on.
		if cfg.OAuth != nil {
			mux.HandleFunc("GET "+wellKnownPRMPath, h.servePRM)
			mux.HandleFunc("GET "+wellKnownPRMPath+"/mcp", h.servePRM)
		}
	}

	if cfg.ServeUI && cfg.WebFS != nil {
		fs := http.FileServer(http.FS(cfg.WebFS))
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" || r.URL.Path == "/index.html" {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			}
			fs.ServeHTTP(w, r)
		}))
	}

	return mux
}

// mcpAuth guards /mcp. It accepts the static APIKey (TUI/CLI/open-webui) and,
// when OAuth is configured, a valid WorkOS AuthKit JWT (Claude.ai / ChatGPT
// Connect). On failure with OAuth on, it emits a WWW-Authenticate challenge so
// MCP clients can discover the authorization server via the PRM.
func (h *handler) mcpAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if h.cfg.APIKey != "" && authz == "Bearer "+h.cfg.APIKey {
			next(w, r)
			return
		}
		if h.cfg.OAuth != nil {
			if tok, ok := strings.CutPrefix(authz, "Bearer "); ok && h.cfg.OAuth.verify(tok) {
				next(w, r)
				return
			}
			w.Header().Set("WWW-Authenticate", h.cfg.OAuth.challenge())
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
}

func (h *handler) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "Bearer "+h.cfg.APIKey {
			next(w, r)
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeErr(w, 400, "bad request")
		return
	}
	if h.cfg.User == "" || h.cfg.Password == "" {
		writeErr(w, 403, "password auth not configured")
		return
	}
	if creds.User != h.cfg.User || creds.Password != h.cfg.Password {
		writeErr(w, 401, "invalid credentials")
		return
	}
	writeJSON(w, 200, map[string]string{"token": h.cfg.APIKey})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
