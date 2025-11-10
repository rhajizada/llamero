package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/hajizar/llamero/internal/auth"
	"github.com/hajizar/llamero/internal/config"
	"github.com/hajizar/llamero/internal/handler"
	"github.com/hajizar/llamero/internal/middleware"
	"github.com/hajizar/llamero/internal/roles"
	"github.com/hajizar/llamero/internal/router"
	"github.com/hajizar/llamero/internal/service"
)

// Server wraps the HTTP server lifecycle.
type Server struct {
	cfg     *config.ServerConfig
	handler *handler.Handler
	router  http.Handler
	authz   *middleware.Authz
	logger  *log.Logger
}

// New constructs the handler, router, and server wiring.
func New(cfg *config.ServerConfig, roleStore *roles.Store, svc *service.Service, logger *log.Logger) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if roleStore == nil {
		return nil, errors.New("role store is required")
	}
	if svc == nil {
		return nil, errors.New("service is required")
	}
	if logger == nil {
		logger = log.Default()
	}

	h, err := handler.New(cfg, roleStore, svc, logger)
	if err != nil {
		return nil, err
	}
	verifier, err := auth.NewTokenVerifier(cfg.JWT)
	if err != nil {
		return nil, err
	}
	authz := middleware.NewAuthz(verifier)
	r := router.New(h, authz)

	return &Server{
		cfg:     cfg,
		handler: h,
		router:  r,
		authz:   authz,
		logger:  logger,
	}, nil
}

// Run starts the HTTP listener until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:    s.cfg.Address,
		Handler: s.router,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Printf("llamero server listening on %s", s.cfg.Address)
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
