package handler_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/handler"
	oauthclient "github.com/rhajizada/llamero/internal/oauth"
	"github.com/rhajizada/llamero/internal/roles"
)

func TestParseOAuthCallbackForm(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("a", int(64<<10)+1)
	req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := handler.ParseOAuthCallbackForm(httptest.NewRecorder(), req)
	var maxErr *http.MaxBytesError
	if !errors.As(err, &maxErr) {
		t.Fatalf("expected MaxBytesError, got %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader("code=abc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if parseErr := handler.ParseOAuthCallbackForm(httptest.NewRecorder(), req); parseErr != nil {
		t.Fatalf("unexpected parse error: %v", parseErr)
	}
	if got := req.Form.Get("code"); got != "abc" {
		t.Fatalf("unexpected parsed code: %s", got)
	}
}

func TestReadProxyPayloadAndWriteProxyReadError(t *testing.T) {
	t.Parallel()

	h := &handler.Handler{}
	req := httptest.NewRequest(http.MethodPost, "/api/chat/completions", bytes.NewReader([]byte(`{"model":"llama"}`)))
	body, err := h.ReadProxyPayload(req)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(body) != `{"model":"llama"}` {
		t.Fatalf("unexpected body: %s", string(body))
	}

	tooLarge := bytes.Repeat([]byte("a"), int(5<<20)+1)
	req = httptest.NewRequest(http.MethodPost, "/api/chat/completions", bytes.NewReader(tooLarge))
	_, err = h.ReadProxyPayload(req)
	if err == nil {
		t.Fatal("expected oversized body error")
	}

	rec := httptest.NewRecorder()
	h.WriteProxyReadError(rec, err)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestOAuthHelpers(t *testing.T) {
	t.Parallel()

	roleStore := loadTestRoles(t)
	h := handler.NewTestHandler(&config.ServerConfig{
		ExternalURL: "https://llamero.example.com/",
		OAuth: config.OAuthConfig{
			AuthorizeURL: "https://issuer.example.com/authorize",
			ClientID:     "client-id",
			RedirectURL:  "https://llamero.example.com/callback",
			Scopes:       []string{"openid", "email"},
		},
		JWT: config.JWTConfig{
			Audience: "api-audience",
			TTL:      2 * time.Hour,
		},
	}, roleStore)

	redirect := h.LoginRedirectURL("token-123")
	if !strings.HasPrefix(redirect, "https://llamero.example.com/login#") {
		t.Fatalf("unexpected redirect URL: %s", redirect)
	}
	if !strings.Contains(redirect, "token=token-123") || !strings.Contains(redirect, "expires_in=7200") {
		t.Fatalf("unexpected redirect query fragment: %s", redirect)
	}

	role, scopes, err := h.DetermineRole(&oauthclient.UserInfo{Subject: "sub-1", Groups: []string{"admins", "admins"}})
	if err != nil {
		t.Fatalf("determineRole returned error: %v", err)
	}
	if role != "admin" || !reflect.DeepEqual(scopes, []string{"models:write", "models:read"}) {
		t.Fatalf("unexpected resolved role/scopes: %s %#v", role, scopes)
	}
	if _, _, roleErr := h.DetermineRole(&oauthclient.UserInfo{
		Subject: "sub-2",
		Groups:  []string{"unknown"},
	}); roleErr == nil {
		t.Fatal("expected unauthorized groups to fail")
	}
}

func loadTestRoles(t *testing.T) *roles.Store {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "roles.yaml")
	raw := strings.TrimSpace(`
default_role: viewer
roles:
  - name: viewer
    scopes: [models:read]
  - name: admin
    scopes: [models:write, models:read]
`)
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write roles file: %v", err)
	}

	store, err := roles.Load(path, map[string][]string{"admin": {"admins"}})
	if err != nil {
		t.Fatalf("load roles: %v", err)
	}
	return store
}
