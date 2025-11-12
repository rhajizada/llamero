package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/handler"
	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/roles"
	"github.com/rhajizada/llamero/internal/router"
	"github.com/rhajizada/llamero/internal/service"
)

const (
	shutdownTimeout   = 5 * time.Second
	readHeaderTimeout = 10 * time.Second
)

// Server wraps the HTTP server lifecycle.
type Server struct {
	cfg     *config.ServerConfig
	handler *handler.Handler
	router  http.Handler
	authz   *middleware.Authz
	logger  *slog.Logger
}

// New constructs the handler, router, and server wiring.
func New(
	cfg *config.ServerConfig,
	roleStore *roles.Store,
	svc *service.Service,
	tasks *asynq.Client,
	logger *slog.Logger,
) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if roleStore == nil {
		return nil, errors.New("role store is required")
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

	h, err := handler.New(cfg, roleStore, svc, tasks, logger)
	if err != nil {
		return nil, err
	}
	verifier, err := auth.NewTokenVerifier(cfg.JWT)
	if err != nil {
		return nil, err
	}
	authz := middleware.NewAuthz(verifier)
	r := router.New(h, authz)
	handlerWithLogging := middleware.Logging(logger)(r)

	return &Server{
		cfg:     cfg,
		handler: h,
		router:  handlerWithLogging,
		authz:   authz,
		logger:  logger,
	}, nil
}

// Run starts the HTTP listener until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.cfg.Address,
		Handler:           s.router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("llamero server listening", "addr", s.cfg.Address)
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
