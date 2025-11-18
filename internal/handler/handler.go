package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/config"
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
	svc    *service.Service
	client *http.Client
	state  *auth.StateStore
	issuer *auth.TokenIssuer
	tasks  *asynq.Client
	logger *slog.Logger
}

// New builds a Handler with the provided dependencies.
func New(
	cfg *config.ServerConfig,
	roleStore *roles.Store,
	svc *service.Service,
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

	return &Handler{
		cfg:    cfg,
		roles:  roleStore,
		svc:    svc,
		client: &http.Client{Timeout: backendHTTPTimeout},
		state:  auth.NewStateStore(stateStoreTTL),
		issuer: issuer,
		tasks:  tasks,
		logger: logger,
	}, nil
}
