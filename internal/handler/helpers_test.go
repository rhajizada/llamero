package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/handler"
	oauthclient "github.com/rhajizada/llamero/internal/oauth"
	"github.com/rhajizada/llamero/internal/roles"
)

func TestHandlerHelpers(t *testing.T) {
	t.Parallel()

	h := &handler.Handler{}
	roleStore := loadTestRoles(t)
	oauthHandler := handler.NewTestHandler(&config.ServerConfig{
		ExternalURL: "https://llamero.example.com/",
		OAuth: config.OAuthConfig{
			AuthorizeURL: "https://issuer.example.com/authorize",
			ClientID:     "client-id",
			RedirectURL:  "https://llamero.example.com/callback",
			Scopes:       []string{"openid", "email"},
		},
		JWT: config.JWTConfig{Audience: "api-audience", TTL: 2 * time.Hour},
	}, roleStore)

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "rejects oversized oauth callback form",
			run: func(t *testing.T) {
				body := strings.Repeat("a", int(64<<10)+1)
				req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				err := handler.ParseOAuthCallbackForm(httptest.NewRecorder(), req)
				var maxErr *http.MaxBytesError
				assert.ErrorAs(t, err, &maxErr)
			},
		},
		{
			name: "parses oauth callback form",
			run: func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader("code=abc"))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				require.NoError(t, handler.ParseOAuthCallbackForm(httptest.NewRecorder(), req))
				assert.Equal(t, "abc", req.Form.Get("code"))
			},
		},
		{
			name: "reads proxy payload",
			run: func(t *testing.T) {
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/chat/completions",
					bytes.NewReader([]byte(`{"model":"llama"}`)),
				)
				body, err := h.ReadProxyPayload(req)
				require.NoError(t, err)
				assert.JSONEq(t, `{"model":"llama"}`, string(body))
			},
		},
		{
			name: "writes proxy payload size error",
			run: func(t *testing.T) {
				tooLarge := bytes.Repeat([]byte("a"), int(5<<20)+1)
				req := httptest.NewRequest(http.MethodPost, "/api/chat/completions", bytes.NewReader(tooLarge))
				_, err := h.ReadProxyPayload(req)
				require.Error(t, err)

				rec := httptest.NewRecorder()
				h.WriteProxyReadError(rec, err)
				assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
			},
		},
		{
			name: "builds login redirect",
			run: func(t *testing.T) {
				redirect := oauthHandler.LoginRedirectURL("token-123")
				assert.True(t, strings.HasPrefix(redirect, "https://llamero.example.com/login#"))
				assert.Contains(t, redirect, "token=token-123")
				assert.Contains(t, redirect, "expires_in=7200")
			},
		},
		{
			name: "resolves role from oauth groups",
			run: func(t *testing.T) {
				role, scopes, err := oauthHandler.DetermineRole(&oauthclient.UserInfo{
					Subject: "sub-1",
					Groups:  []string{"admins", "admins"},
				})
				require.NoError(t, err)
				assert.Equal(t, "admin", role)
				assert.Equal(t, []string{"models:write", "models:read"}, scopes)
			},
		},
		{
			name: "rejects unauthorized oauth groups",
			run: func(t *testing.T) {
				_, _, err := oauthHandler.DetermineRole(&oauthclient.UserInfo{
					Subject: "sub-2",
					Groups:  []string{"unknown"},
				})
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
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
	require.NoError(t, os.WriteFile(path, []byte(raw), 0o600))

	store, err := roles.Load(path, map[string][]string{"admin": {"admins"}})
	require.NoError(t, err)
	return store
}
