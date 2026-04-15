package server_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/roles"
	"github.com/rhajizada/llamero/internal/server"
	"github.com/rhajizada/llamero/internal/service"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestNewValidatesInputs(t *testing.T) {
	t.Parallel()

	roleStore := testutil.MustLoadRoles(t, `
default_role: viewer
roles:
  - name: viewer
    scopes: [models:list]
`, nil)
	svc := service.New(nil, nil)
	tasks := &asynq.Client{}

	for _, tc := range []struct {
		name string
		cfg  *config.ServerConfig
		role *roles.Store
		svc  *service.Service
		task *asynq.Client
	}{
		{name: "nil config", cfg: nil, role: roleStore, svc: svc, task: tasks},
		{name: "nil roles", cfg: &config.ServerConfig{}, role: nil, svc: svc, task: tasks},
		{name: "nil service", cfg: &config.ServerConfig{}, role: roleStore, svc: nil, task: tasks},
		{name: "nil task client", cfg: &config.ServerConfig{}, role: roleStore, svc: svc, task: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := server.New(tc.cfg, tc.role, tc.svc, tc.task, slog.Default()); err == nil {
				t.Fatalf("expected %s to fail", tc.name)
			}
		})
	}
}

func TestNewTestServerDefaults(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *config.ServerConfig
		router http.Handler
		logger *slog.Logger
	}{
		{name: "defaults router and logger", cfg: &config.ServerConfig{Address: "127.0.0.1:0"}},
		{
			name:   "preserves provided router and logger",
			cfg:    &config.ServerConfig{Address: "127.0.0.1:0"},
			router: http.NewServeMux(),
			logger: slog.Default(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := server.NewTestServer(tc.cfg, tc.router, tc.logger)
			if srv == nil {
				t.Fatal("expected server")
			}
			if err := srv.Run(runCanceledContext(t)); err != nil {
				t.Fatalf("unexpected Run error: %v", err)
			}
		})
	}
}

func runCanceledContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestNewRejectsInvalidJWTConfig(t *testing.T) {
	t.Parallel()

	roleStore := testutil.MustLoadRoles(t, `
default_role: viewer
roles:
  - name: viewer
    scopes: [models:list]
`, nil)
	_, err := server.New(
		&config.ServerConfig{JWT: config.JWTConfig{}},
		roleStore,
		service.New(nil, nil),
		&asynq.Client{},
		slog.Default(),
	)
	if err == nil {
		t.Fatal("expected invalid JWT config to fail")
	}
}

func TestNewSuccess(t *testing.T) {
	t.Parallel()

	roleStore := testutil.MustLoadRoles(t, `
default_role: viewer
roles:
  - name: viewer
    scopes: [models:list]
`, nil)
	cfg := &config.ServerConfig{JWT: testutil.MustWriteEd25519JWTConfig(t), Address: "127.0.0.1:0"}
	srv, err := server.New(cfg, roleStore, service.New(nil, nil), &asynq.Client{}, nil)
	if err != nil {
		t.Fatalf("expected server.New success, got %v", err)
	}
	if srv == nil {
		t.Fatal("expected server")
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("invalid address", func(t *testing.T) {
		t.Parallel()

		srv := server.NewTestServer(
			&config.ServerConfig{Address: "bad:addr:123"},
			http.NewServeMux(),
			slog.Default(),
		)
		err := srv.Run(context.Background())
		if err == nil {
			t.Fatal("expected invalid address to fail")
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		t.Parallel()

		srv := server.NewTestServer(
			&config.ServerConfig{Address: "127.0.0.1:0"},
			http.NewServeMux(),
			slog.Default(),
		)
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		if err := srv.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected Run error: %v", err)
		}
	})
}
