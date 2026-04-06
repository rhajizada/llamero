package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/models"
	oauthclient "github.com/rhajizada/llamero/internal/oauth"
	backendproxy "github.com/rhajizada/llamero/internal/proxy"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/roles"
	"github.com/rhajizada/llamero/internal/service"
)

const (
	// backendHTTPTimeout controls how long we wait for backend responses. Set to
	// zero to allow long-running streaming requests to complete.
	backendHTTPTimeout = 0
	stateStoreTTL      = 5 * time.Minute
)

// Handler coordinates OAuth flow endpoints and JWT issuance.
type Handler struct {
	cfg    *config.ServerConfig
	roles  *roles.Store
	svc    serviceAPI
	oauth  oauthclient.Client
	proxy  backendproxy.Forwarder
	state  *auth.StateStore
	issuer *auth.TokenIssuer
	tasks  *asynq.Client
	logger *slog.Logger
}

type serviceAPI interface {
	ListModels(ctx context.Context) (models.ModelList, error)
	GetModel(ctx context.Context, id string) (models.Model, error)
	ListBackends(ctx context.Context) ([]models.Backend, error)
	LookupBackendRoute(ctx context.Context, backendID string) (service.BackendRoute, error)
	RouteBackend(ctx context.Context, model string) (service.BackendRoute, error)
	RouteResponsesCreate(ctx context.Context, model string) (service.BackendRoute, error)
	UpsertUser(ctx context.Context, params repository.UpsertUserParams) (repository.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (models.User, error)
	ListPersonalAccessTokens(ctx context.Context, userID uuid.UUID) ([]models.PersonalAccessToken, error)
	GetPersonalAccessToken(ctx context.Context, userID, tokenID uuid.UUID) (models.PersonalAccessToken, error)
	CreatePersonalAccessToken(ctx context.Context, params service.CreateTokenParams) (models.PersonalAccessToken, error)
	RevokePersonalAccessToken(ctx context.Context, userID, tokenID uuid.UUID) error
}

// New builds a Handler with the provided dependencies.
func New(
	cfg *config.ServerConfig,
	roleStore *roles.Store,
	svc serviceAPI,
	tasks *asynq.Client,
	logger *slog.Logger,
) (*Handler, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if roleStore == nil {
		return nil, errors.New("roles store is required")
	}
	if svc == nil {
		return nil, errors.New("service is required")
	}
	if tasks == nil {
		return nil, errors.New("task client is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	issuer, err := auth.NewTokenIssuer(cfg.JWT)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: backendHTTPTimeout}

	return &Handler{
		cfg:    cfg,
		roles:  roleStore,
		svc:    svc,
		oauth:  oauthclient.New(cfg, client),
		proxy:  backendproxy.New(client),
		state:  auth.NewStateStore(stateStoreTTL),
		issuer: issuer,
		tasks:  tasks,
		logger: logger,
	}, nil
}

func NewTestHandler(cfg *config.ServerConfig, roleStore *roles.Store) *Handler {
	client := &http.Client{Timeout: backendHTTPTimeout}
	return &Handler{
		cfg:    cfg,
		roles:  roleStore,
		oauth:  oauthclient.New(cfg, client),
		proxy:  backendproxy.New(client),
		state:  auth.NewStateStore(stateStoreTTL),
		logger: slog.Default(),
	}
}

func NewTestHandlerWithDeps(
	cfg *config.ServerConfig,
	roleStore *roles.Store,
	svc serviceAPI,
	oauth oauthclient.Client,
	proxy backendproxy.Forwarder,
	issuer *auth.TokenIssuer,
	logger *slog.Logger,
) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		cfg:    cfg,
		roles:  roleStore,
		svc:    svc,
		oauth:  oauth,
		proxy:  proxy,
		state:  auth.NewStateStore(stateStoreTTL),
		issuer: issuer,
		logger: logger,
	}
}
