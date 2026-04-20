package server_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			_, err := server.New(tc.cfg, tc.role, tc.svc, tc.task, slog.Default())
			assert.Error(t, err)
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
			t.Parallel()

			srv := server.NewTestServer(tc.cfg, tc.router, tc.logger)
			require.NotNil(t, srv)
			assert.NoError(t, srv.Run(runCanceledContext(t)))
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
	assert.Error(t, err)
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
	require.NoError(t, err)
	assert.NotNil(t, srv)
}

func TestRun(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "invalid address",
			run: func(t *testing.T) {
				srv := server.NewTestServer(
					&config.ServerConfig{Address: "bad:addr:123"},
					http.NewServeMux(),
					slog.Default(),
				)
				assert.Error(t, srv.Run(context.Background()))
			},
		},
		{
			name: "context canceled",
			run: func(t *testing.T) {
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
				err := srv.Run(ctx)
				assert.True(t, err == nil || errors.Is(err, context.Canceled))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}
