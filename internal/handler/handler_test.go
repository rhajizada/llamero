package handler_test

import (
	"log/slog"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/handler"
	"github.com/rhajizada/llamero/internal/roles"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestNewValidationAndDefaults(t *testing.T) {
	roleStore := testutil.MustLoadRoles(t, `
default_role: viewer
roles:
  - name: viewer
    scopes: [models:list]
`, nil)
	validCfg := &config.ServerConfig{JWT: testutil.MustWriteEd25519JWTConfig(t)}

	tests := []struct {
		name    string
		cfg     *config.ServerConfig
		roles   *roles.Store
		svc     *fakeService
		nilSvc  bool
		tasks   *asynq.Client
		logger  *slog.Logger
		wantErr string
	}{
		{name: "rejects nil config", roles: roleStore, svc: &fakeService{}, tasks: &asynq.Client{}, wantErr: "config is required"},
		{name: "rejects nil roles", cfg: validCfg, svc: &fakeService{}, tasks: &asynq.Client{}, wantErr: "roles store is required"},
		{name: "rejects nil service", cfg: validCfg, roles: roleStore, nilSvc: true, tasks: &asynq.Client{}, wantErr: "service is required"},
		{name: "rejects nil task client", cfg: validCfg, roles: roleStore, svc: &fakeService{}, wantErr: "task client is required"},
		{name: "builds handler with default logger", cfg: validCfg, roles: roleStore, svc: &fakeService{}, tasks: &asynq.Client{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var (
				h   *handler.Handler
				err error
			)
			if tc.nilSvc {
				h, err = handler.New(tc.cfg, tc.roles, nil, tc.tasks, tc.logger)
			} else {
				h, err = handler.New(tc.cfg, tc.roles, tc.svc, tc.tasks, tc.logger)
			}
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, h)
		})
	}
}
