package oauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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
	if err != nil {
		t.Fatalf("BuildAuthorizeURL returned error: %v", err)
	}
	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatalf("parse authorize URL: %v", err)
	}
	q := parsed.Query()
	if q.Get("client_id") != "client-id" || q.Get("state") != "state-123" || q.Get("audience") != "api-audience" {
		t.Fatalf("unexpected authorize params: %#v", q)
	}
}

func TestBuildAuthorizeURLError(t *testing.T) {
	t.Parallel()

	client := oauth.New(&config.ServerConfig{
		OAuth: config.OAuthConfig{AuthorizeURL: "://bad-url"},
	}, nil)
	if _, err := client.BuildAuthorizeURL("state-123"); err == nil {
		t.Fatal("expected invalid authorize URL to fail")
	}
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
	if err != nil {
		t.Fatalf("ExchangeCode returned error: %v", err)
	}
	if tokenResp.AccessToken != "token-123" {
		t.Fatalf("unexpected access token: %q", tokenResp.AccessToken)
	}

	userInfo, err := client.FetchUserInfo(context.Background(), tokenResp.AccessToken)
	if err != nil {
		t.Fatalf("FetchUserInfo returned error: %v", err)
	}
	if userInfo.Subject != "sub-1" || userInfo.Email != "user@example.com" || userInfo.Name != "Test User" {
		t.Fatalf("unexpected user info: %#v", userInfo)
	}
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
			if err := tc.run(nil); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	cfg := *baseCfg
	cfg.OAuth.UserInfoURL = server.URL + "/userinfo-fallback-email"
	client := oauth.New(&cfg, server.Client())
	info, err := client.FetchUserInfo(context.Background(), "token")
	if err != nil {
		t.Fatalf("FetchUserInfo fallback email returned error: %v", err)
	}
	if info.Email != "sub-1" || len(info.Groups) != 1 || info.Groups[0] != "admins" {
		t.Fatalf("unexpected fallback user info: %#v", info)
	}
}
