package handler_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/handler"
	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/models"
	oauthclient "github.com/rhajizada/llamero/internal/oauth"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/requestctx"
	"github.com/rhajizada/llamero/internal/service"
	"github.com/rhajizada/llamero/internal/testutil"
)

type fakeService struct {
	listModelsFn           func(context.Context) (models.ModelList, error)
	getModelFn             func(context.Context, string) (models.Model, error)
	listTokensFn           func(context.Context, uuid.UUID) ([]models.PersonalAccessToken, error)
	getTokenFn             func(context.Context, uuid.UUID, uuid.UUID) (models.PersonalAccessToken, error)
	createTokenFn          func(context.Context, service.CreateTokenParams) (models.PersonalAccessToken, error)
	revokeTokenFn          func(context.Context, uuid.UUID, uuid.UUID) error
	upsertUserFn           func(context.Context, repository.UpsertUserParams) (repository.User, error)
	getUserFn              func(context.Context, uuid.UUID) (models.User, error)
	listBackendsFn         func(context.Context) ([]models.Backend, error)
	lookupBackendRouteFn   func(context.Context, string) (service.BackendRoute, error)
	routeBackendFn         func(context.Context, string) (service.BackendRoute, error)
	routeResponsesCreateFn func(context.Context, string) (service.BackendRoute, error)
}

func (f *fakeService) ListModels(ctx context.Context) (models.ModelList, error) {
	return f.listModelsFn(ctx)
}
func (f *fakeService) GetModel(ctx context.Context, id string) (models.Model, error) {
	return f.getModelFn(ctx, id)
}

func (f *fakeService) ListPersonalAccessTokens(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.PersonalAccessToken, error) {
	return f.listTokensFn(ctx, userID)
}

func (f *fakeService) GetPersonalAccessToken(
	ctx context.Context,
	userID, tokenID uuid.UUID,
) (models.PersonalAccessToken, error) {
	return f.getTokenFn(ctx, userID, tokenID)
}

func (f *fakeService) CreatePersonalAccessToken(
	ctx context.Context,
	params service.CreateTokenParams,
) (models.PersonalAccessToken, error) {
	return f.createTokenFn(ctx, params)
}

func (f *fakeService) RevokePersonalAccessToken(ctx context.Context, userID, tokenID uuid.UUID) error {
	return f.revokeTokenFn(ctx, userID, tokenID)
}

func (f *fakeService) UpsertUser(ctx context.Context, params repository.UpsertUserParams) (repository.User, error) {
	return f.upsertUserFn(ctx, params)
}

func (f *fakeService) GetUser(ctx context.Context, id uuid.UUID) (models.User, error) {
	return f.getUserFn(ctx, id)
}

func (f *fakeService) ListBackends(ctx context.Context) ([]models.Backend, error) {
	if f.listBackendsFn == nil {
		return []models.Backend{}, nil
	}
	return f.listBackendsFn(ctx)
}

func (f *fakeService) LookupBackendRoute(ctx context.Context, backendID string) (service.BackendRoute, error) {
	return f.lookupBackendRouteFn(ctx, backendID)
}

func (f *fakeService) RouteBackend(ctx context.Context, model string) (service.BackendRoute, error) {
	return f.routeBackendFn(ctx, model)
}

func (f *fakeService) RouteResponsesCreate(ctx context.Context, model string) (service.BackendRoute, error) {
	return f.routeResponsesCreateFn(ctx, model)
}

type fakeOAuthClient struct {
	buildAuthorizeURLFn func(string) (string, error)
	exchangeCodeFn      func(context.Context, string) (*oauthclient.TokenResponse, error)
	fetchUserInfoFn     func(context.Context, string) (*oauthclient.UserInfo, error)
}

func (f *fakeOAuthClient) BuildAuthorizeURL(state string) (string, error) {
	return f.buildAuthorizeURLFn(state)
}
func (f *fakeOAuthClient) ExchangeCode(ctx context.Context, code string) (*oauthclient.TokenResponse, error) {
	return f.exchangeCodeFn(ctx, code)
}
func (f *fakeOAuthClient) FetchUserInfo(ctx context.Context, accessToken string) (*oauthclient.UserInfo, error) {
	return f.fetchUserInfoFn(ctx, accessToken)
}

type nopProxy struct{}

var errUnexpectedProxyCall = errors.New("unexpected proxy call")

func (nopProxy) Forward(*http.Request, string, string, string, []byte) (*http.Response, error) {
	return nil, errUnexpectedProxyCall
}

func (nopProxy) ForwardGET(*http.Request, string, string) (*http.Response, error) {
	return nil, errUnexpectedProxyCall
}

func (nopProxy) ForwardLLM(*http.Request, string, []byte) (*http.Response, error) {
	return nil, errUnexpectedProxyCall
}

type fakeProxy struct {
	forwardFn    func(*http.Request, string, string, string, []byte) (*http.Response, error)
	forwardGETFn func(*http.Request, string, string) (*http.Response, error)
	forwardLLMFn func(*http.Request, string, []byte) (*http.Response, error)
}

func (f *fakeProxy) Forward(
	r *http.Request,
	method, baseAddress, targetPath string,
	body []byte,
) (*http.Response, error) {
	return f.forwardFn(r, method, baseAddress, targetPath, body)
}

func (f *fakeProxy) ForwardGET(r *http.Request, baseAddress, targetPath string) (*http.Response, error) {
	return f.forwardGETFn(r, baseAddress, targetPath)
}

func (f *fakeProxy) ForwardLLM(r *http.Request, baseAddress string, body []byte) (*http.Response, error) {
	return f.forwardLLMFn(r, baseAddress, body)
}

func TestHandleListModelsAndGetModel(t *testing.T) {
	t.Parallel()

	fakeSvc := &fakeService{
		listModelsFn: func(context.Context) (models.ModelList, error) {
			return models.ModelList{Object: "list", Data: []models.Model{{ID: "llama3", Object: "model"}}}, nil
		},
		getModelFn: func(_ context.Context, id string) (models.Model, error) {
			return models.Model{ID: id, Object: "model", OwnedBy: "library"}, nil
		},
	}
	h := handler.NewTestHandlerWithDeps(&config.ServerConfig{}, nil, fakeSvc, nil, nopProxy{}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	h.HandleListModels(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"id":"llama3"`) {
		t.Fatalf("unexpected list models response: code=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/models/llama3", nil)
	req.SetPathValue("modelID", "llama3")
	h.HandleGetModel(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"id":"llama3"`) {
		t.Fatalf("unexpected get model response: code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTokenHandlers(t *testing.T) {
	t.Parallel()

	issuer := testutil.MustNewTokenIssuer(t)
	userID := uuid.New()
	tokenID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	fakeSvc := &fakeService{
		listTokensFn: func(context.Context, uuid.UUID) ([]models.PersonalAccessToken, error) {
			return []models.PersonalAccessToken{{
				ID:        tokenID,
				UserID:    userID,
				Name:      "cli",
				Scopes:    []string{"models:list"},
				CreatedAt: now,
				UpdatedAt: now,
			}}, nil
		},
		getTokenFn: func(context.Context, uuid.UUID, uuid.UUID) (models.PersonalAccessToken, error) {
			return models.PersonalAccessToken{
				ID:        tokenID,
				UserID:    userID,
				Name:      "cli",
				Scopes:    []string{"models:list"},
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
		createTokenFn: func(_ context.Context, params service.CreateTokenParams) (models.PersonalAccessToken, error) {
			return models.PersonalAccessToken{
				ID:        tokenID,
				UserID:    params.UserID,
				Name:      params.Name,
				Scopes:    params.Scopes,
				TokenType: params.TokenType,
				Jti:       params.JTI,
				ExpiresAt: params.ExpiresAt,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
		revokeTokenFn: func(context.Context, uuid.UUID, uuid.UUID) error { return nil },
	}
	h := handler.NewTestHandlerWithDeps(&config.ServerConfig{}, nil, fakeSvc, nil, nopProxy{}, issuer, nil)
	claims := &auth.Claims{
		RegisteredClaims: jwtClaims(userID.String()),
		Type:             auth.TokenTypeSession,
		Scopes:           []string{"models:list", "profile:get"},
		Role:             "admin",
		Email:            "user@example.com",
	}

	rec := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/profile/tokens", nil), claims)
	h.HandleListTokens(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"cli"`) {
		t.Fatalf("unexpected list tokens response: code=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = withClaims(httptest.NewRequest(http.MethodGet, "/api/profile/tokens/"+tokenID.String(), nil), claims)
	req.SetPathValue("tokenID", tokenID.String())
	h.HandleGetToken(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tokenID.String()) {
		t.Fatalf("unexpected get token response: code=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = withClaims(
		httptest.NewRequest(
			http.MethodPost,
			"/api/profile/tokens",
			strings.NewReader(`{"name":"cli","scopes":["models:list"],"expires_in":7200}`),
		),
		claims,
	)
	h.HandleCreateToken(rec, req)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"token":"`) {
		t.Fatalf("unexpected create token response: code=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = withClaims(httptest.NewRequest(http.MethodDelete, "/api/profile/tokens/"+tokenID.String(), nil), claims)
	req.SetPathValue("tokenID", tokenID.String())
	h.HandleDeleteToken(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected delete token status: %d", rec.Code)
	}
}

func TestOAuthLoginAndCallback(t *testing.T) {
	t.Parallel()

	roleStore := testutil.MustLoadRoles(t, `
default_role: viewer
roles:
  - name: viewer
    scopes: [models:list]
  - name: admin
    scopes: [models:list, profile:get, models:write]
`, map[string][]string{"admin": {"admins"}})
	issuer := testutil.MustNewTokenIssuer(t)
	userID := uuid.New()
	var upserted repository.UpsertUserParams

	fakeOAuth := &fakeOAuthClient{
		buildAuthorizeURLFn: func(state string) (string, error) {
			return "https://issuer.example.com/authorize?state=" + state, nil
		},
		exchangeCodeFn: func(context.Context, string) (*oauthclient.TokenResponse, error) {
			return &oauthclient.TokenResponse{AccessToken: "token-123"}, nil
		},
		fetchUserInfoFn: func(context.Context, string) (*oauthclient.UserInfo, error) {
			return &oauthclient.UserInfo{
				Subject: "sub-1",
				Email:   "user@example.com",
				Name:    "User",
				Groups:  []string{"admins", "admins"},
			}, nil
		},
	}
	fakeSvc := &fakeService{
		upsertUserFn: func(_ context.Context, params repository.UpsertUserParams) (repository.User, error) {
			upserted = params
			return repository.User{ID: userID, Email: params.Email, Role: params.Role, Scopes: params.Scopes}, nil
		},
		getUserFn: func(context.Context, uuid.UUID) (models.User, error) { return models.User{}, nil },
	}
	cfg := &config.ServerConfig{
		ExternalURL: "https://llamero.example.com/",
		OAuth:       config.OAuthConfig{ProviderName: "auth0"},
		JWT:         config.JWTConfig{TTL: 2 * time.Hour},
	}
	h := handler.NewTestHandlerWithDeps(cfg, roleStore, fakeSvc, fakeOAuth, nopProxy{}, issuer, nil)

	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	h.Login(loginRec, loginReq)
	if loginRec.Code != http.StatusFound {
		t.Fatalf("unexpected login status: %d", loginRec.Code)
	}
	redirectURL, err := url.Parse(loginRec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse login location: %v", err)
	}
	state := redirectURL.Query().Get("state")
	if state == "" {
		t.Fatal("expected oauth state in login redirect")
	}

	callbackRec := httptest.NewRecorder()
	callbackReq := httptest.NewRequest(
		http.MethodPost,
		"/auth/callback",
		strings.NewReader("code=code-123&state="+state),
	)
	callbackReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Callback(callbackRec, callbackReq)
	if callbackRec.Code != http.StatusFound {
		t.Fatalf("unexpected callback status: %d body=%s", callbackRec.Code, callbackRec.Body.String())
	}
	if upserted.Provider != "auth0" || upserted.Email != "user@example.com" || upserted.Role != "admin" {
		t.Fatalf("unexpected upserted user params: %#v", upserted)
	}
	if !strings.Contains(callbackRec.Header().Get("Location"), "token=") {
		t.Fatalf("expected token in callback redirect: %s", callbackRec.Header().Get("Location"))
	}
}

func TestHealth(t *testing.T) {
	t.Parallel()

	h := handler.NewTestHandler(&config.ServerConfig{}, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.Health(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("unexpected health response: code=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestBackendHandlers(t *testing.T) {
	t.Parallel()

	fakeSvc := &fakeService{
		listBackendsFn: func(context.Context) ([]models.Backend, error) {
			return []models.Backend{{ID: "backend-a", Address: "http://backend-a:11434", Healthy: true}}, nil
		},
		lookupBackendRouteFn: func(_ context.Context, backendID string) (service.BackendRoute, error) {
			return service.BackendRoute{ID: backendID, Address: "http://backend-a:11434"}, nil
		},
	}
	proxy := &fakeProxy{
		forwardGETFn: func(r *http.Request, baseAddress, targetPath string) (*http.Response, error) {
			if baseAddress != "http://backend-a:11434" || targetPath != "/api/tags" {
				t.Fatalf("unexpected GET proxy target: %s %s", baseAddress, targetPath)
			}
			if backendID, _ := requestctx.BackendID(r.Context()); backendID != "backend-a" {
				t.Fatalf("expected backend id in context, got %q", backendID)
			}
			return responseOK(`{"models":[{"name":"llama3"}]}`), nil
		},
		forwardFn: func(_ *http.Request, method, baseAddress, targetPath string, body []byte) (*http.Response, error) {
			if method != http.MethodPost || baseAddress != "http://backend-a:11434" || targetPath != "/api/create" {
				t.Fatalf("unexpected proxy target: %s %s %s", method, baseAddress, targetPath)
			}
			if string(body) != `{"name":"llama3"}` {
				t.Fatalf("unexpected proxy body: %s", string(body))
			}
			return responseOK(`{"status":"success"}`), nil
		},
	}
	h := handler.NewTestHandlerWithDeps(&config.ServerConfig{}, nil, fakeSvc, nil, proxy, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/backends", nil)
	h.HandleListBackends(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"id":"backend-a"`) {
		t.Fatalf("unexpected list backends response: code=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/backends/backend-a/tags", nil)
	req.SetPathValue("backendID", "backend-a")
	h.HandleBackendTags(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"llama3"`) {
		t.Fatalf("unexpected backend tags response: code=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/backends/backend-a/create", strings.NewReader(`{"name":"llama3"}`))
	req.SetPathValue("backendID", "backend-a")
	h.HandleBackendCreate(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"success"`) {
		t.Fatalf("unexpected backend create response: code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLLMProxyHandlers(t *testing.T) {
	t.Parallel()

	fakeSvc := &fakeService{
		routeBackendFn: func(_ context.Context, model string) (service.BackendRoute, error) {
			if model != "llama3" {
				t.Fatalf("unexpected model route request: %s", model)
			}
			return service.BackendRoute{ID: "backend-a", Address: "http://backend-a:11434"}, nil
		},
		routeResponsesCreateFn: func(_ context.Context, _ string) (service.BackendRoute, error) {
			return service.BackendRoute{ID: "backend-b", Address: "http://backend-b:11434"}, nil
		},
	}
	proxy := &fakeProxy{
		forwardLLMFn: func(r *http.Request, _ string, body []byte) (*http.Response, error) {
			if backendID, _ := requestctx.BackendID(r.Context()); backendID == "" {
				t.Fatal("expected backend id in proxied context")
			}
			if !strings.Contains(string(body), `"model":"llama3"`) {
				t.Fatalf("unexpected llm body: %s", string(body))
			}
			return responseOK(`{"object":"chat.completion","model":"llama3"}`), nil
		},
	}
	h := handler.NewTestHandlerWithDeps(&config.ServerConfig{}, nil, fakeSvc, nil, proxy, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat/completions", strings.NewReader(`{"model":"llama3"}`))
	h.HandleChatCompletions(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"model":"llama3"`) {
		t.Fatalf("unexpected chat completions response: code=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/completions", strings.NewReader(`{"model":"llama3"}`))
	h.HandleCompletions(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected completions status: %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/responses", strings.NewReader(`{"model":"llama3"}`))
	h.HandleResponsesCreate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected responses status: %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/chat/completions", strings.NewReader(`{"model":""}`))
	h.HandleChatCompletions(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for empty model, got %d", rec.Code)
	}
}

func TestRemainingBackendGETHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		path        string
		handler     func(*handler.Handler, http.ResponseWriter, *http.Request)
		backendPath string
	}{
		{
			name:        "processes",
			path:        "/api/backends/backend-a/ps",
			handler:     (*handler.Handler).HandleBackendProcesses,
			backendPath: "/api/ps",
		},
		{
			name:        "version",
			path:        "/api/backends/backend-a/version",
			handler:     (*handler.Handler).HandleBackendVersion,
			backendPath: "/api/version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newBackendTestHandler(t, http.MethodGet, tt.backendPath, "")
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.SetPathValue("backendID", "backend-a")
			rec := httptest.NewRecorder()
			tt.handler(h, rec, req)
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
				t.Fatalf("unexpected response: code=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRemainingBackendMutationHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		path        string
		handler     func(*handler.Handler, http.ResponseWriter, *http.Request)
		backendPath string
		body        string
	}{
		{
			name:        "copy",
			method:      http.MethodPost,
			path:        "/api/backends/backend-a/copy",
			handler:     (*handler.Handler).HandleBackendCopy,
			backendPath: "/api/copy",
			body:        `{"source":"a","destination":"b"}`,
		},
		{
			name:        "pull",
			method:      http.MethodPost,
			path:        "/api/backends/backend-a/pull",
			handler:     (*handler.Handler).HandleBackendPull,
			backendPath: "/api/pull",
			body:        `{"name":"llama3"}`,
		},
		{
			name:        "push",
			method:      http.MethodPost,
			path:        "/api/backends/backend-a/push",
			handler:     (*handler.Handler).HandleBackendPush,
			backendPath: "/api/push",
			body:        `{"name":"llama3"}`,
		},
		{
			name:        "delete",
			method:      http.MethodDelete,
			path:        "/api/backends/backend-a/delete",
			handler:     (*handler.Handler).HandleBackendDelete,
			backendPath: "/api/delete",
			body:        `{"name":"llama3"}`,
		},
		{
			name:        "show",
			method:      http.MethodPost,
			path:        "/api/backends/backend-a/show",
			handler:     (*handler.Handler).HandleBackendShow,
			backendPath: "/api/show",
			body:        `{"name":"llama3"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newBackendTestHandler(t, tt.method, tt.backendPath, tt.body)
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.SetPathValue("backendID", "backend-a")
			rec := httptest.NewRecorder()
			tt.handler(h, rec, req)
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
				t.Fatalf("unexpected response: code=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestBackendAndProxyErrorPaths(t *testing.T) {
	t.Parallel()

	h := handler.NewTestHandlerWithDeps(
		&config.ServerConfig{},
		nil,
		&fakeService{
			lookupBackendRouteFn: func(context.Context, string) (service.BackendRoute, error) {
				return service.BackendRoute{}, &service.Error{Code: http.StatusNotFound, Message: "backend not found"}
			},
			routeBackendFn: func(context.Context, string) (service.BackendRoute, error) {
				return service.BackendRoute{}, service.ErrNoHealthyBackends
			},
		},
		nil,
		&fakeProxy{
			forwardLLMFn: func(*http.Request, string, []byte) (*http.Response, error) {
				return nil, errors.New("boom")
			},
		},
		nil,
		nil,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/backends/backend-a/tags", nil)
	req.SetPathValue("backendID", "backend-a")
	h.HandleBackendTags(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected backend not found status, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/chat/completions", strings.NewReader(`{"model":"llama3"}`))
	h.HandleChatCompletions(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected service unavailable, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestEmbeddingsAndProfileHandlers(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	h := handler.NewTestHandlerWithDeps(
		&config.ServerConfig{},
		nil,
		&fakeService{
			routeBackendFn: func(_ context.Context, _ string) (service.BackendRoute, error) {
				return service.BackendRoute{ID: "backend-a", Address: "http://backend-a:11434"}, nil
			},
			getUserFn: func(_ context.Context, id uuid.UUID) (models.User, error) {
				return models.User{ID: id, Email: "user@example.com", Role: "admin"}, nil
			},
		},
		nil,
		&fakeProxy{
			forwardLLMFn: func(r *http.Request, _ string, body []byte) (*http.Response, error) {
				if !strings.Contains(string(body), `"model":"llama3"`) {
					t.Fatalf("unexpected embeddings body: %s", string(body))
				}
				if backendID, _ := requestctx.BackendID(r.Context()); backendID != "backend-a" {
					t.Fatalf("expected backend id in context, got %q", backendID)
				}
				return responseOK(`{"object":"list","model":"llama3","data":[]}`), nil
			},
		},
		nil,
		nil,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/embeddings",
		strings.NewReader(`{"model":"llama3","input":"hello"}`),
	)
	h.HandleEmbeddings(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"model":"llama3"`) {
		t.Fatalf("unexpected embeddings response: code=%d body=%s", rec.Code, rec.Body.String())
	}

	claims := &auth.Claims{RegisteredClaims: jwtClaims(userID.String())}
	rec = httptest.NewRecorder()
	req = withClaims(httptest.NewRequest(http.MethodGet, "/api/profile", nil), claims)
	h.Profile(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"email":"user@example.com"`) {
		t.Fatalf("unexpected profile response: code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func withClaims(req *http.Request, claims *auth.Claims) *http.Request {
	ctx := middleware.ContextWithClaims(req.Context(), claims)
	return req.WithContext(ctx)
}

func jwtClaims(subject string) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{Subject: subject}
}

func newBackendTestHandler(t *testing.T, wantMethod, wantPath, wantBody string) *handler.Handler {
	t.Helper()

	fakeSvc := &fakeService{
		lookupBackendRouteFn: func(_ context.Context, backendID string) (service.BackendRoute, error) {
			return service.BackendRoute{ID: backendID, Address: "http://backend-a:11434"}, nil
		},
	}
	proxy := &fakeProxy{
		forwardGETFn: func(r *http.Request, baseAddress, targetPath string) (*http.Response, error) {
			if wantMethod != http.MethodGet {
				t.Fatal("unexpected GET proxy call")
			}
			if baseAddress != "http://backend-a:11434" || targetPath != wantPath {
				t.Fatalf("unexpected GET target: %s %s", baseAddress, targetPath)
			}
			if backendID, _ := requestctx.BackendID(r.Context()); backendID != "backend-a" {
				t.Fatalf("expected backend id in context, got %q", backendID)
			}
			return responseOK(`{"ok":true}`), nil
		},
		forwardFn: func(r *http.Request, method, baseAddress, targetPath string, body []byte) (*http.Response, error) {
			if wantMethod == http.MethodGet {
				t.Fatal("unexpected body proxy call")
			}
			if method != wantMethod || baseAddress != "http://backend-a:11434" || targetPath != wantPath {
				t.Fatalf("unexpected proxy target: %s %s %s", method, baseAddress, targetPath)
			}
			if string(body) != wantBody {
				t.Fatalf("unexpected proxy body: %s", string(body))
			}
			if backendID, _ := requestctx.BackendID(r.Context()); backendID != "backend-a" {
				t.Fatalf("expected backend id in context, got %q", backendID)
			}
			return responseOK(`{"ok":true}`), nil
		},
	}
	return handler.NewTestHandlerWithDeps(&config.ServerConfig{}, nil, fakeSvc, nil, proxy, nil, nil)
}

func responseOK(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
