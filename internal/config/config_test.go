package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/config"
)

func TestConfigHelpers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "backends.yaml")
	raw := strings.TrimSpace(`
backends:
  - id: primary
    address: http://backend-1:11434
    tags: [gpu, us-east]
    weight: 2
  - id: secondary
    address: https://backend-2.example.com
`)
	require.NoError(t, os.WriteFile(path, []byte(raw), 0o600))

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "parses role groups",
			run: func(t *testing.T) {
				got, err := config.ParseRoleGroups("admin=ops, admins ; viewer = readers, readers")
				require.NoError(t, err)
				assert.Equal(t, map[string][]string{"admin": {"ops", "admins"}, "viewer": {"readers"}}, got)
			},
		},
		{
			name: "rejects invalid role groups entry",
			run: func(t *testing.T) {
				_, err := config.ParseRoleGroups("admin")
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid role mapping entry")
			},
		},
		{
			name: "builds postgres dsn",
			run: func(t *testing.T) {
				dsn := (config.PostgresConfig{
					Host:     "db",
					Port:     5432,
					User:     "user",
					Password: "secret",
					DBName:   "llamero",
					SSLMode:  "disable",
				}).DSN()
				assert.Equal(t, "postgres://user:secret@db:5432/llamero?sslmode=disable", dsn)
			},
		},
		{
			name: "loads backend definitions",
			run: func(t *testing.T) {
				backends, err := config.LoadBackendDefinitions(path)
				require.NoError(t, err)
				require.Len(t, backends, 2)
				assert.Equal(t, "primary", backends[0].ID)
				assert.Equal(t, 2, backends[0].Weight)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestLoaders(t *testing.T) {
	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	require.NoError(t, os.WriteFile(privPath, []byte("key"), 0o600))

	t.Setenv("LLAMERO_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("LLAMERO_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("LLAMERO_OAUTH_AUTHORIZE_URL", "https://issuer.example.com/authorize")
	t.Setenv("LLAMERO_OAUTH_TOKEN_URL", "https://issuer.example.com/token")
	t.Setenv("LLAMERO_OAUTH_USERINFO_URL", "https://issuer.example.com/userinfo")
	t.Setenv("LLAMERO_OAUTH_REDIRECT_URL", "https://llamero.example.com/callback")
	t.Setenv("LLAMERO_JWT_PRIVATE_KEY_PATH", privPath)
	t.Setenv("LLAMERO_JWT_TTL", "2h")
	t.Setenv("LLAMERO_ROLE_GROUPS", "admin=admins,ops")
	t.Setenv("LLAMERO_REDIS_ADDR", "localhost:6379")
	t.Setenv("LLAMERO_POSTGRES_HOST", "localhost")
	t.Setenv("LLAMERO_POSTGRES_USER", "postgres")
	t.Setenv("LLAMERO_POSTGRES_PASSWORD", "secret")
	t.Setenv("LLAMERO_POSTGRES_DBNAME", "llamero")

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "loads server config",
			run: func(t *testing.T) {
				serverCfg, err := config.LoadServer()
				require.NoError(t, err)
				assert.Equal(t, 2*time.Hour, serverCfg.JWT.TTL)
				assert.Equal(t, "admins", serverCfg.Roles.Groups["admin"][0])
			},
		},
		{
			name: "loads worker config",
			run: func(t *testing.T) {
				workerCfg, err := config.LoadWorker()
				require.NoError(t, err)
				assert.Equal(t, "localhost:6379", workerCfg.Store.Addr)
				assert.Equal(t, "llamero", workerCfg.Database.Postgres.DBName)
			},
		},
		{
			name: "loads scheduler config",
			run: func(t *testing.T) {
				schedulerCfg, err := config.LoadScheduler()
				require.NoError(t, err)
				assert.Equal(t, "localhost:6379", schedulerCfg.Store.Addr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t)
		})
	}
}
