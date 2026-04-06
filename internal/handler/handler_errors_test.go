package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/handler"
	"github.com/rhajizada/llamero/internal/models"
	oauthclient "github.com/rhajizada/llamero/internal/oauth"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/service"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestModelHandlerErrors(t *testing.T) {
	t.Parallel()

	h := handler.NewTestHandlerWithDeps(
		&config.ServerConfig{},
		nil,
		&fakeService{
			listModelsFn: func(context.Context) (models.ModelList, error) {
				return models.ModelList{}, &service.Error{Code: http.StatusTeapot, Message: "teapot"}
			},
			getModelFn: func(context.Context, string) (models.Model, error) {
				return models.Model{}, errors.New("boom")
			},
		},
		nil,
		nil,
		nil,
		nil,
	)

	rec := httptest.NewRecorder()
	h.HandleListModels(rec, httptest.NewRequest(http.MethodGet, "/api/models", nil))
	assertStatusContainsErr(t, rec, http.StatusTeapot, "teapot")

	rec = httptest.NewRecorder()
	h.HandleGetModel(rec, httptest.NewRequest(http.MethodGet, "/api/models", nil))
	assertStatusContainsErr(t, rec, http.StatusNotFound, "model not found")

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/models/llama3", nil)
	req.SetPathValue("modelID", "llama3")
	h.HandleGetModel(rec, req)
	assertStatusContainsErr(t, rec, http.StatusInternalServerError, "failed to load model")
}

func TestOAuthErrorPaths(t *testing.T) {
	t.Parallel()

	h := handler.NewTestHandlerWithDeps(
		&config.ServerConfig{
			ExternalURL: "https://llamero.example.com/",
			JWT:         config.JWTConfig{TTL: time.Hour},
			OAuth:       config.OAuthConfig{ProviderName: "issuer"},
		},
		testutil.MustLoadRoles(t, "default_role: viewer\nroles:\n  - name: viewer\n    scopes: [models:list]", nil),
		&fakeService{getUserFn: func(context.Context, uuid.UUID) (models.User, error) {
			return models.User{}, &service.Error{Code: http.StatusNotFound, Message: "not found"}
		}},
		&fakeOAuthClient{
			buildAuthorizeURLFn: func(string) (string, error) { return "", errors.New("bad config") },
			exchangeCodeFn: func(context.Context, string) (*oauthclient.TokenResponse, error) {
				return nil, errors.New("exchange failed")
			},
			fetchUserInfoFn: func(context.Context, string) (*oauthclient.UserInfo, error) {
				return nil, errors.New("userinfo failed")
			},
		},
		nil,
		testutil.MustNewTokenIssuer(t),
		nil,
	)

	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	assertStatusContainsErr(t, rec, http.StatusInternalServerError, "configuration error")

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/callback", strings.NewReader("code=x&state=bad"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Callback(rec, req)
	assertStatusContainsErr(t, rec, http.StatusBadRequest, "invalid state")

	rec = httptest.NewRecorder()
	h.Profile(rec, httptest.NewRequest(http.MethodGet, "/api/profile", nil))
	assertStatusContainsErr(t, rec, http.StatusUnauthorized, "missing authentication context")

	rec = httptest.NewRecorder()
	req = withClaims(
		httptest.NewRequest(http.MethodGet, "/api/profile", nil),
		&auth.Claims{RegisteredClaims: jwtClaims("not-a-uuid")},
	)
	h.Profile(rec, req)
	assertStatusContainsErr(t, rec, http.StatusInternalServerError, "invalid user identifier")
}

func TestOAuthCallbackAdditionalErrors(t *testing.T) {
	t.Parallel()

	h := handler.NewTestHandlerWithDeps(
		&config.ServerConfig{
			ExternalURL: "https://llamero.example.com/",
			JWT:         config.JWTConfig{TTL: time.Hour},
			OAuth:       config.OAuthConfig{ProviderName: "issuer"},
		},
		testutil.MustLoadRoles(
			t,
			"default_role: admin\nroles:\n  - name: admin\n    scopes: [models:list]",
			map[string][]string{"admin": {"admins"}},
		),
		&fakeService{upsertUserFn: func(_ context.Context, _ repository.UpsertUserParams) (repository.User, error) {
			return repository.User{ID: uuid.Nil}, nil
		}},
		&fakeOAuthClient{
			buildAuthorizeURLFn: func(state string) (string, error) { return "https://issuer.example.com/authorize?state=" + state, nil },
			exchangeCodeFn: func(context.Context, string) (*oauthclient.TokenResponse, error) {
				return &oauthclient.TokenResponse{}, nil
			},
			fetchUserInfoFn: func(context.Context, string) (*oauthclient.UserInfo, error) {
				return &oauthclient.UserInfo{
					Subject: "sub-1",
					Email:   "user@example.com",
					Groups:  []string{"admins"},
				}, nil
			},
		},
		nil,
		testutil.MustNewTokenIssuer(t),
		nil,
	)

	loginRec := httptest.NewRecorder()
	h.Login(loginRec, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	state := mustLocationStateErr(t, loginRec.Header().Get("Location"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/callback", strings.NewReader("code=x&state="+state))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Callback(rec, req)
	assertStatusContainsErr(t, rec, http.StatusBadGateway, "provider did not return access token")
}

func TestLLMProxyInvalidPayloads(t *testing.T) {
	t.Parallel()

	h := handler.NewTestHandlerWithDeps(&config.ServerConfig{}, nil, &fakeService{}, nil, &fakeProxy{}, nil, nil)

	for _, tc := range []struct {
		name    string
		path    string
		body    string
		handler func(*handler.Handler, http.ResponseWriter, *http.Request)
		want    string
	}{
		{name: "embeddings invalid json", path: "/api/embeddings", body: "{", handler: (*handler.Handler).HandleEmbeddings, want: "invalid JSON payload"},
		{name: "embeddings missing model", path: "/api/embeddings", body: `{"model":""}`, handler: (*handler.Handler).HandleEmbeddings, want: "model is required"},
		{name: "completions invalid json", path: "/api/completions", body: "{", handler: (*handler.Handler).HandleCompletions, want: "invalid JSON payload"},
		{name: "responses invalid json", path: "/api/responses", body: "{", handler: (*handler.Handler).HandleResponsesCreate, want: "invalid JSON payload"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			tc.handler(h, rec, req)
			assertStatusContainsErr(t, rec, http.StatusBadRequest, tc.want)
		})
	}
}

func TestTokenHandlerValidationErrors(t *testing.T) {
	t.Parallel()

	issuer := testutil.MustNewTokenIssuer(t)
	userID := uuid.New()
	baseClaims := &auth.Claims{
		RegisteredClaims: jwtClaims(userID.String()),
		Type:             auth.TokenTypeSession,
		Scopes:           []string{"models:list"},
		Role:             "admin",
		Email:            "user@example.com",
	}
	h := handler.NewTestHandlerWithDeps(&config.ServerConfig{}, nil, &fakeService{}, nil, nopProxy{}, issuer, nil)

	rec := httptest.NewRecorder()
	req := withClaims(
		httptest.NewRequest(http.MethodGet, "/api/profile/tokens", nil),
		&auth.Claims{RegisteredClaims: jwtClaims(userID.String()), Type: auth.TokenTypePAT},
	)
	h.HandleListTokens(rec, req)
	assertStatusContainsErr(t, rec, http.StatusForbidden, "cannot manage other tokens")

	rec = httptest.NewRecorder()
	req = withClaims(httptest.NewRequest(http.MethodGet, "/api/profile/tokens/bad", nil), baseClaims)
	req.SetPathValue("tokenID", "bad")
	h.HandleGetToken(rec, req)
	assertStatusContainsErr(t, rec, http.StatusBadRequest, "invalid token id")

	for _, tc := range []struct {
		name string
		body string
		code int
		want string
	}{
		{name: "invalid json", body: "{", code: http.StatusBadRequest, want: "invalid JSON payload"},
		{name: "missing name", body: `{"name":"","scopes":["models:list"]}`, code: http.StatusBadRequest, want: "name is required"},
		{name: "missing scopes", body: `{"name":"cli","scopes":[]}`, code: http.StatusBadRequest, want: "at least one scope is required"},
		{name: "scope escalation", body: `{"name":"cli","scopes":["admin:all"]}`, code: http.StatusForbidden, want: "requested scopes exceed current permissions"},
		{name: "ttl too short", body: `{"name":"cli","scopes":["models:list"],"expires_in":10}`, code: http.StatusBadRequest, want: "at least 1 hour"},
		{name: "ttl too long", body: `{"name":"cli","scopes":["models:list"],"expires_in":99999999}`, code: http.StatusBadRequest, want: "cannot exceed 90 days"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec2 := httptest.NewRecorder()
			req2 := withClaims(
				httptest.NewRequest(http.MethodPost, "/api/profile/tokens", strings.NewReader(tc.body)),
				baseClaims,
			)
			h.HandleCreateToken(rec2, req2)
			assertStatusContainsErr(t, rec2, tc.code, tc.want)
		})
	}

	h = handler.NewTestHandlerWithDeps(&config.ServerConfig{}, nil, &fakeService{
		createTokenFn: func(context.Context, service.CreateTokenParams) (models.PersonalAccessToken, error) {
			return models.PersonalAccessToken{}, errors.New("boom")
		},
		revokeTokenFn: func(context.Context, uuid.UUID, uuid.UUID) error { return errors.New("boom") },
	}, nil, nopProxy{}, issuer, nil)
	rec = httptest.NewRecorder()
	req = withClaims(
		httptest.NewRequest(
			http.MethodPost,
			"/api/profile/tokens",
			strings.NewReader(`{"name":"cli","scopes":["models:list"]}`),
		),
		baseClaims,
	)
	h.HandleCreateToken(rec, req)
	assertStatusContainsErr(t, rec, http.StatusInternalServerError, "failed to create token")

	rec = httptest.NewRecorder()
	req = withClaims(
		httptest.NewRequest(http.MethodDelete, "/api/profile/tokens/bad", nil),
		&auth.Claims{RegisteredClaims: jwtClaims(userID.String()), Type: auth.TokenTypePAT},
	)
	h.HandleDeleteToken(rec, req)
	assertStatusContainsErr(t, rec, http.StatusForbidden, "cannot manage other tokens")

	rec = httptest.NewRecorder()
	req = withClaims(httptest.NewRequest(http.MethodDelete, "/api/profile/tokens/bad", nil), baseClaims)
	req.SetPathValue("tokenID", "bad")
	h.HandleDeleteToken(rec, req)
	assertStatusContainsErr(t, rec, http.StatusBadRequest, "invalid token id")

	rec = httptest.NewRecorder()
	tokenID := uuid.NewString()
	req = withClaims(httptest.NewRequest(http.MethodDelete, "/api/profile/tokens/"+tokenID, nil), baseClaims)
	req.SetPathValue("tokenID", tokenID)
	h.HandleDeleteToken(rec, req)
	assertStatusContainsErr(t, rec, http.StatusInternalServerError, "failed to revoke token")

	h = handler.NewTestHandlerWithDeps(&config.ServerConfig{}, nil, &fakeService{
		createTokenFn: func(_ context.Context, params service.CreateTokenParams) (models.PersonalAccessToken, error) {
			return models.PersonalAccessToken{
				ID:     uuid.New(),
				UserID: params.UserID,
				Name:   params.Name,
				Scopes: params.Scopes,
			}, nil
		},
	}, nil, nopProxy{}, testutil.MustNewTokenIssuer(t), nil)
	claimsNoEmail := &auth.Claims{
		RegisteredClaims: jwtClaims(userID.String()),
		Type:             auth.TokenTypeSession,
		Scopes:           []string{"models:list"},
		Role:             "admin",
	}
	rec = httptest.NewRecorder()
	req = withClaims(
		httptest.NewRequest(
			http.MethodPost,
			"/api/profile/tokens",
			strings.NewReader(`{"name":"cli","scopes":["models:list"]}`),
		),
		claimsNoEmail,
	)
	h.HandleCreateToken(rec, req)
	assertStatusContainsErr(t, rec, http.StatusInternalServerError, "failed to issue token")
}

func assertStatusContainsErr(t *testing.T, rec *httptest.ResponseRecorder, code int, text string) {
	t.Helper()
	if rec.Code != code || !strings.Contains(rec.Body.String(), text) {
		t.Fatalf("unexpected response: code=%d body=%s want=%q", rec.Code, rec.Body.String(), text)
	}
}

func mustLocationStateErr(t *testing.T, location string) string {
	t.Helper()
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatal("expected state in redirect location")
	}
	return state
}
