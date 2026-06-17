package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testKID = "test-key-1"

// startJWKS serves a JWKS containing pub at /oauth2/jwks, mimicking a WorkOS
// AuthKit instance. The server URL doubles as the issuer.
func startJWKS(t *testing.T, pub *rsa.PublicKey) *httptest.Server {
	t.Helper()
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	jwks := fmt.Sprintf(`{"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":%q,"n":%q,"e":%q}]}`, testKID, n, e)
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jwks))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// mint signs a WorkOS-style access token. aud=="" → resource; email=="" → omitted.
func mint(t *testing.T, priv *rsa.PrivateKey, issuer, resource, aud, email string, exp time.Time) string {
	t.Helper()
	if aud == "" {
		aud = resource
	}
	claims := jwt.MapClaims{
		"iss": issuer,
		"aud": aud,
		"sub": "user_123",
		"iat": time.Now().Unix(),
		"exp": exp.Unix(),
	}
	if email != "" {
		claims["email"] = email
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	s, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func newOAuth(t *testing.T, allowedEmails, allowedSubs string) (*MCPOAuth, *rsa.PrivateKey, string, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := startJWKS(t, &priv.PublicKey)
	issuer := jwks.URL
	resource := "https://odak.test/mcp"
	o := LoadMCPOAuth(context.Background(), issuer, resource, allowedEmails, allowedSubs)
	if o == nil {
		t.Fatal("LoadMCPOAuth returned nil")
	}
	return o, priv, issuer, resource
}

func TestLoadMCPOAuth_DisabledWhenUnset(t *testing.T) {
	if o := LoadMCPOAuth(context.Background(), "", "", "", ""); o != nil {
		t.Fatal("want nil when issuer/resource empty")
	}
	if o := LoadMCPOAuth(context.Background(), "https://x.authkit.app", "", "", ""); o != nil {
		t.Fatal("want nil when resource empty")
	}
}

func TestVerifyJWT(t *testing.T) {
	o, priv, issuer, resource := newOAuth(t, "", "")

	if !o.verify(mint(t, priv, issuer, resource, "", "", time.Now().Add(time.Hour))) {
		t.Error("valid token rejected")
	}
	if o.verify(mint(t, priv, issuer, resource, "https://evil.test/mcp", "", time.Now().Add(time.Hour))) {
		t.Error("wrong-audience token accepted (RFC 8707 replay defense)")
	}
	if o.verify(mint(t, priv, "https://evil.example", resource, "", "", time.Now().Add(time.Hour))) {
		t.Error("wrong-issuer token accepted")
	}
	if o.verify(mint(t, priv, issuer, resource, "", "", time.Now().Add(-time.Hour))) {
		t.Error("expired token accepted")
	}
	if o.verify("not.a.jwt") {
		t.Error("garbage token accepted")
	}
}

func TestVerifyJWT_EmailAllowlist(t *testing.T) {
	o, priv, issuer, resource := newOAuth(t, "me@example.com, other@example.com", "")

	if !o.verify(mint(t, priv, issuer, resource, "", "me@example.com", time.Now().Add(time.Hour))) {
		t.Error("allowlisted email rejected")
	}
	if !o.verify(mint(t, priv, issuer, resource, "", "ME@Example.com", time.Now().Add(time.Hour))) {
		t.Error("allowlist should be case-insensitive")
	}
	if o.verify(mint(t, priv, issuer, resource, "", "stranger@example.com", time.Now().Add(time.Hour))) {
		t.Error("non-allowlisted email accepted")
	}
	if o.verify(mint(t, priv, issuer, resource, "", "", time.Now().Add(time.Hour))) {
		t.Error("token with no email accepted despite allowlist")
	}
}

func TestVerifyJWT_SubAllowlist(t *testing.T) {
	// mint() always sets sub="user_123"; gate on it, with no email in the token.
	o, priv, issuer, resource := newOAuth(t, "", "user_123")

	if !o.verify(mint(t, priv, issuer, resource, "", "", time.Now().Add(time.Hour))) {
		t.Error("allowlisted sub rejected (email-less access token)")
	}
	o2, priv2, issuer2, resource2 := newOAuth(t, "", "user_999")
	if o2.verify(mint(t, priv2, issuer2, resource2, "", "", time.Now().Add(time.Hour))) {
		t.Error("non-allowlisted sub accepted")
	}
}

func TestPRMJSON(t *testing.T) {
	o, _, issuer, resource := newOAuth(t, "", "")
	var prm struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
		ScopesSupported      []string `json:"scopes_supported"`
	}
	if err := json.Unmarshal(o.prmJSON(), &prm); err != nil {
		t.Fatal(err)
	}
	if prm.Resource != resource {
		t.Errorf("resource = %q, want %q", prm.Resource, resource)
	}
	if len(prm.AuthorizationServers) != 1 || prm.AuthorizationServers[0] != issuer {
		t.Errorf("authorization_servers = %v, want [%s]", prm.AuthorizationServers, issuer)
	}
	want := false
	for _, s := range prm.ScopesSupported {
		if s == "offline_access" {
			want = true
		}
	}
	if !want {
		t.Error("offline_access must be advertised so refresh tokens are issued")
	}
}

func TestChallenge_PointsAtPathScopedPRM(t *testing.T) {
	o, _, _, _ := newOAuth(t, "", "")
	got := o.challenge()
	const wantMeta = `resource_metadata="https://odak.test/.well-known/oauth-protected-resource/mcp"`
	if !strings.Contains(got, wantMeta) {
		t.Errorf("challenge = %q, want it to contain %q", got, wantMeta)
	}
}

// TestMCPAuth covers the combined guard: static APIKey OR a valid JWT, with a
// discovery challenge on failure.
func TestMCPAuth(t *testing.T) {
	o, priv, issuer, resource := newOAuth(t, "", "")
	h := &handler{cfg: Config{APIKey: "secret-key", OAuth: o}}
	next := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
	guard := h.mcpAuth(next)

	call := func(authz string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/mcp", nil)
		if authz != "" {
			req.Header.Set("Authorization", authz)
		}
		rec := httptest.NewRecorder()
		guard(rec, req)
		return rec
	}

	if rec := call("Bearer secret-key"); rec.Code != http.StatusNoContent {
		t.Errorf("static API key: got %d, want 204", rec.Code)
	}
	good := mint(t, priv, issuer, resource, "", "", time.Now().Add(time.Hour))
	if rec := call("Bearer " + good); rec.Code != http.StatusNoContent {
		t.Errorf("valid JWT: got %d, want 204", rec.Code)
	}
	rec := call("")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no creds: got %d, want 401", rec.Code)
	}
	if wa := rec.Header().Get("WWW-Authenticate"); !strings.Contains(wa, "resource_metadata=") {
		t.Errorf("401 missing resource_metadata challenge: %q", wa)
	}
	if rec := call("Bearer wrong"); rec.Code != http.StatusUnauthorized {
		t.Errorf("bad token: got %d, want 401", rec.Code)
	}
}
