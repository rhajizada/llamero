package handler

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/roles"
	"github.com/rhajizada/llamero/internal/service"
)

// Handler coordinates OAuth flow endpoints and JWT issuance.
type Handler struct {
	cfg    *config.ServerConfig
	roles  *roles.Store
	svc    *service.Service
	client *http.Client
	state  *auth.StateStore
	issuer *auth.TokenIssuer
	logger *log.Logger
}

// New builds a Handler with the provided dependencies.
func New(cfg *config.ServerConfig, roleStore *roles.Store, svc *service.Service, logger *log.Logger) (*Handler, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if roleStore == nil {
		return nil, errors.New("roles store is required")
	}
	if svc == nil {
		return nil, errors.New("service is required")
	}
	if logger == nil {
		logger = log.Default()
	}

	issuer, err := auth.NewTokenIssuer(cfg.JWT)
	if err != nil {
		return nil, err
	}

	return &Handler{
		cfg:    cfg,
		roles:  roleStore,
		svc:    svc,
		client: &http.Client{Timeout: 10 * time.Second},
		state:  auth.NewStateStore(5 * time.Minute),
		issuer: issuer,
		logger: logger,
	}, nil
}
