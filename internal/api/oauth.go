package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// wellKnownPRMPath is the RFC 9728 Protected Resource Metadata base path. Claude
// and ChatGPT probe both this and the path-scoped "<base>/mcp" variant.
const wellKnownPRMPath = "/.well-known/oauth-protected-resource"

// MCPOAuth is the OAuth 2.1 Resource-Server config for /mcp. It lets the endpoint
// accept WorkOS AuthKit-issued JWTs (the Claude.ai / ChatGPT "Connect" flow) in
// addition to the static ODAK_API_KEY, and serves the Protected Resource Metadata
// (RFC 9728) that bootstraps that flow.
//
// It is nil (OAuth disabled) unless both issuer and resource are configured —
// when unconfigured /mcp stays API-key-only and advertises no OAuth, so this
// whole layer is inert until the WorkOS env vars land.
type MCPOAuth struct {
	issuer   string // AuthKit domain, e.g. https://x.authkit.app
	resource string // the aud we enforce (this MCP endpoint's public URL)
	keyfunc  keyfunc.Keyfunc
	allowed  map[string]bool // allowed email claims; empty ⇒ any valid token from the issuer
}

// LoadMCPOAuth builds the OAuth config, or returns nil when issuer/resource are
// empty (disabled). ctx bounds the background JWKS refresh goroutine. allowedEmails
// is a comma-separated allowlist of the `email` claim — odak is single-user, so we
// gate on identity rather than trust every account the AuthKit app might hold.
func LoadMCPOAuth(ctx context.Context, issuer, resource, allowedEmails string) *MCPOAuth {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	resource = strings.TrimSpace(resource)
	if issuer == "" || resource == "" {
		return nil
	}
	kf, err := keyfunc.NewDefaultCtx(ctx, []string{issuer + "/oauth2/jwks"})
	if err != nil {
		log.Printf("odak: mcp oauth init failed — OAuth disabled, MCP stays API-key-only: %v", err)
		return nil
	}
	allowed := map[string]bool{}
	for _, e := range strings.Split(allowedEmails, ",") {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			allowed[e] = true
		}
	}
	log.Printf("odak: mcp oauth enabled (issuer=%s resource=%s allowlist=%d)", issuer, resource, len(allowed))
	return &MCPOAuth{issuer: issuer, resource: resource, keyfunc: kf, allowed: allowed}
}

// verify validates a WorkOS access token against the JWKS, enforcing issuer,
// audience (== resource; RFC 8707 replay defense), signing alg, a small clock-skew
// leeway, and the optional email allowlist. Returns true when the token is good.
func (o *MCPOAuth) verify(token string) bool {
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(token, claims, o.keyfunc.Keyfunc,
		jwt.WithIssuer(o.issuer),
		jwt.WithAudience(o.resource),
		jwt.WithValidMethods([]string{"RS256", "ES256"}),
		jwt.WithLeeway(60*time.Second),
	)
	if err != nil || !tok.Valid {
		return false
	}
	if len(o.allowed) > 0 {
		email, _ := claims["email"].(string)
		if !o.allowed[strings.ToLower(email)] {
			return false
		}
	}
	return true
}

// prmJSON is the Protected Resource Metadata document (RFC 9728). offline_access
// is advertised so the client requests it and WorkOS issues a refresh token —
// without it the access token silently expires and the host 401s with no way to
// refresh (forcing a full reconnect).
func (o *MCPOAuth) prmJSON() []byte {
	b, _ := json.Marshal(map[string]any{
		"resource":                 o.resource,
		"authorization_servers":    []string{o.issuer},
		"scopes_supported":         []string{"openid", "email", "offline_access"},
		"bearer_methods_supported": []string{"header"},
	})
	return b
}

// challenge is the WWW-Authenticate value for a 401 on the MCP endpoint: it points
// the client at the path-scoped PRM so it can discover the authorization server.
func (o *MCPOAuth) challenge() string {
	meta := o.resource
	if u, err := url.Parse(o.resource); err == nil {
		meta = u.Scheme + "://" + u.Host + wellKnownPRMPath + u.Path
	}
	return `Bearer resource_metadata="` + meta + `"`
}

// servePRM serves the Protected Resource Metadata. Public + static.
func (h *handler) servePRM(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(h.cfg.OAuth.prmJSON())
}
