package oauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/oauth"
)

func TestBuildAuthorizeURL(t *testing.T) {
	t.Parallel()

	client := oauth.New(&config.ServerConfig{
		OAuth: config.OAuthConfig{
			AuthorizeURL: "https://issuer.example.com/authorize",
			ClientID:     "client-id",
			RedirectURL:  "https://llamero.example.com/callback",
			Scopes:       []string{"openid", "email"},
		},
		JWT: config.JWTConfig{Audience: "api-audience"},
	}, nil)

	target, err := client.BuildAuthorizeURL("state-123")
	require.NoError(t, err)
	parsed, err := url.Parse(target)
	require.NoError(t, err)
	q := parsed.Query()
	assert.Equal(t, "client-id", q.Get("client_id"))
	assert.Equal(t, "state-123", q.Get("state"))
	assert.Equal(t, "api-audience", q.Get("audience"))
}

func TestBuildAuthorizeURLError(t *testing.T) {
	t.Parallel()

	client := oauth.New(&config.ServerConfig{
		OAuth: config.OAuthConfig{AuthorizeURL: "://bad-url"},
	}, nil)
	_, err := client.BuildAuthorizeURL("state-123")
	assert.Error(t, err)
}

func TestExchangeCodeAndFetchUserInfo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if got := r.Header.Get("Content-Type"); !strings.Contains(got, "application/x-www-form-urlencoded") {
				t.Fatalf("unexpected content type: %q", got)
			}
			user, pass, _ := r.BasicAuth()
			if user != "client-id" || pass != "client-secret" {
				t.Fatalf("unexpected basic auth: %q %q", user, pass)
			}
			_, _ = w.Write([]byte(`{"access_token":"token-123","token_type":"Bearer"}`))
		case "/userinfo":
			if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
				t.Fatalf("unexpected auth header: %q", got)
			}
			_, _ = w.Write([]byte(
				`{"sub":"sub-1","email":"user@example.com","name":"Test User","groups":["admins","admins"]}`,
			))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := oauth.New(&config.ServerConfig{
		OAuth: config.OAuthConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURL:  "https://llamero.example.com/callback",
			TokenURL:     server.URL + "/token",
			UserInfoURL:  server.URL + "/userinfo",
		},
	}, server.Client())

	tokenResp, err := client.ExchangeCode(context.Background(), "code-123")
	require.NoError(t, err)
	assert.Equal(t, "token-123", tokenResp.AccessToken)

	userInfo, err := client.FetchUserInfo(context.Background(), tokenResp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "sub-1", userInfo.Subject)
	assert.Equal(t, "user@example.com", userInfo.Email)
	assert.Equal(t, "Test User", userInfo.Name)
}

func TestExchangeCodeAndFetchUserInfoErrors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token-status":
			http.Error(w, "nope", http.StatusBadGateway)
		case "/token-json":
			_, _ = w.Write([]byte("{"))
		case "/userinfo-status":
			http.Error(w, "nope", http.StatusBadGateway)
		case "/userinfo-json":
			_, _ = w.Write([]byte("{"))
		case "/userinfo-missing-sub":
			_, _ = w.Write([]byte(`{"email":"user@example.com"}`))
		case "/userinfo-fallback-email":
			_, _ = w.Write([]byte(`{"sub":"sub-1","groups":"admins,admins"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	baseCfg := &config.ServerConfig{OAuth: config.OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://llamero.example.com/callback",
	}}

	for _, tc := range []struct {
		name string
		run  func(*oauth.HTTPClient) error
	}{
		{
			name: "token status",
			run: func(_ *oauth.HTTPClient) error {
				client := oauth.New(&config.ServerConfig{OAuth: config.OAuthConfig{
					ClientID:     "client-id",
					ClientSecret: "client-secret",
					RedirectURL:  "https://llamero.example.com/callback",
					TokenURL:     server.URL + "/token-status",
				}}, server.Client())
				_, err := client.ExchangeCode(context.Background(), "code")
				return err
			},
		},
		{
			name: "token invalid json",
			run: func(_ *oauth.HTTPClient) error {
				client := oauth.New(&config.ServerConfig{OAuth: config.OAuthConfig{
					ClientID:     "client-id",
					ClientSecret: "client-secret",
					RedirectURL:  "https://llamero.example.com/callback",
					TokenURL:     server.URL + "/token-json",
				}}, server.Client())
				_, err := client.ExchangeCode(context.Background(), "code")
				return err
			},
		},
		{
			name: "userinfo status",
			run: func(_ *oauth.HTTPClient) error {
				cfg := *baseCfg
				cfg.OAuth.UserInfoURL = server.URL + "/userinfo-status"
				client := oauth.New(&cfg, server.Client())
				_, err := client.FetchUserInfo(context.Background(), "token")
				return err
			},
		},
		{
			name: "userinfo invalid json",
			run: func(_ *oauth.HTTPClient) error {
				cfg := *baseCfg
				cfg.OAuth.UserInfoURL = server.URL + "/userinfo-json"
				client := oauth.New(&cfg, server.Client())
				_, err := client.FetchUserInfo(context.Background(), "token")
				return err
			},
		},
		{
			name: "userinfo missing subject",
			run: func(_ *oauth.HTTPClient) error {
				cfg := *baseCfg
				cfg.OAuth.UserInfoURL = server.URL + "/userinfo-missing-sub"
				client := oauth.New(&cfg, server.Client())
				_, err := client.FetchUserInfo(context.Background(), "token")
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, tc.run(nil))
		})
	}

	cfg := *baseCfg
	cfg.OAuth.UserInfoURL = server.URL + "/userinfo-fallback-email"
	client := oauth.New(&cfg, server.Client())
	info, err := client.FetchUserInfo(context.Background(), "token")
	require.NoError(t, err)
	assert.Equal(t, "sub-1", info.Email)
	require.Len(t, info.Groups, 1)
	assert.Equal(t, "admins", info.Groups[0])
}
