package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	File      string
	APIKey    string
	User      string
	Password  string
	Port      string
	BackupDir string
	// OAuth (WorkOS AuthKit) for the MCP endpoint. All empty ⇒ /mcp stays
	// APIKey-only and advertises no OAuth.
	OAuthIssuer        string // AuthKit domain, e.g. https://x.authkit.app
	MCPResource        string // public /mcp URL, the OAuth audience
	OAuthAllowedEmails string // comma-separated email allowlist (single-user gate)
}

type Client struct {
	Endpoint string
	Token    string
}

func DefaultServerPort() string { return "8761" }

// LoadServer reads ODAK_* env vars as fallback values.
func LoadServer() Server {
	return Server{
		File:      env("ODAK_FILE", ""),
		APIKey:    env("ODAK_API_KEY", ""),
		User:      env("ODAK_USER", ""),
		Password:  env("ODAK_PASSWORD", ""),
		Port:      env("ODAK_PORT", DefaultServerPort()),
		BackupDir: env("ODAK_BACKUP_DIR", ""),

		OAuthIssuer:        env("ODAK_OAUTH_ISSUER", ""),
		MCPResource:        env("ODAK_MCP_RESOURCE", ""),
		OAuthAllowedEmails: env("ODAK_OAUTH_ALLOWED_EMAIL", ""),
	}
}

// LoadClient reads ~/.config/odak/client config + ODAK_CLIENT_* env vars.
func LoadClient() Client {
	c := Client{
		Endpoint: env("ODAK_ENDPOINT", "http://localhost:"+DefaultServerPort()),
		Token:    env("ODAK_TOKEN", ""),
	}
	// simple key=value file: endpoint=... / token=...
	if data, err := os.ReadFile(clientConfigPath()); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
			if !ok {
				continue
			}
			switch strings.TrimSpace(k) {
			case "endpoint":
				if c.Endpoint == "http://localhost:"+DefaultServerPort() {
					c.Endpoint = strings.TrimSpace(v)
				}
			case "token":
				if c.Token == "" {
					c.Token = strings.TrimSpace(v)
				}
			}
		}
	}
	return c
}

func clientConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "odak", "client")
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
