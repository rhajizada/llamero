package main

import (
	"log/slog"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestSetupDatabase(t *testing.T) {
	t.Parallel()

	ctx, dsn := testutil.MustStartPostgres(t)
	tests := []struct {
		name      string
		cfg       *config.ServerConfig
		wantErr   string
		checkPool bool
	}{
		{
			name:      "migrates and connects",
			cfg:       &config.ServerConfig{Database: config.DatabaseConfig{Postgres: mustPostgresConfig(t, dsn), MigrationsDir: testutil.MigrationsDir(t)}},
			checkPool: true,
		},
		{
			name:    "fails on missing migrations dir",
			cfg:     &config.ServerConfig{Database: config.DatabaseConfig{Postgres: mustPostgresConfig(t, dsn), MigrationsDir: filepath.Join(t.TempDir(), "missing")}},
			wantErr: "migrate database",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := SetupDatabase(ctx, tc.cfg)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			t.Cleanup(pool.Close)
			assert.NoError(t, pool.Ping(ctx))
		})
	}
}

func TestNewServerEnvironment(t *testing.T) {
	ctx, dsn := testutil.MustStartPostgres(t)
	_, redisAddr := testutil.MustStartRedis(t)
	baseCfg := newServerTestConfig(t, dsn, redisAddr)
	t.Chdir(repoRoot(t))

	tests := []struct {
		name    string
		mutate  func(*config.ServerConfig)
		wantErr string
	}{
		{name: "builds environment"},
		{
			name: "fails on missing backend definition file",
			mutate: func(cfg *config.ServerConfig) {
				cfg.Backends.FilePath = filepath.Join(t.TempDir(), "missing-backends.yaml")
			},
			wantErr: "load backends",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseCfg
			if tc.mutate != nil {
				tc.mutate(&cfg)
			}

			env, err := NewServerEnvironment(ctx, &cfg, slog.Default())
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, env)
			require.NotNil(t, env.service)
			require.NotNil(t, env.server)
			t.Cleanup(env.Close)
		})
	}
}

func newServerTestConfig(t *testing.T, dsn, redisAddr string) config.ServerConfig {
	t.Helper()

	jwtCfg := testutil.MustWriteEd25519JWTConfig(t)
	return config.ServerConfig{
		Address:     "127.0.0.1:0",
		ExternalURL: "http://localhost:8080",
		OAuth: config.OAuthConfig{
			ProviderName: "oauth",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			AuthorizeURL: "https://issuer.example.com/authorize",
			TokenURL:     "https://issuer.example.com/token",
			UserInfoURL:  "https://issuer.example.com/userinfo",
			RedirectURL:  "http://localhost:8080/auth/callback",
			Scopes:       []string{"openid", "email"},
		},
		JWT:   jwtCfg,
		Roles: config.RoleMappingConfig{Groups: map[string][]string{}},
		Database: config.DatabaseConfig{
			Postgres:      mustPostgresConfig(t, dsn),
			MigrationsDir: testutil.MigrationsDir(t),
		},
		Store:    config.RedisConfig{Addr: redisAddr},
		Backends: config.BackendsConfig{FilePath: filepath.Join("config", "backends.yaml")},
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	return filepath.Clean(filepath.Join(testutil.MigrationsDir(t), "..", "..", ".."))
}

func mustPostgresConfig(t *testing.T, dsn string) config.PostgresConfig {
	t.Helper()

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	port, err := strconv.Atoi(parsed.Port())
	require.NoError(t, err)
	password, _ := parsed.User.Password()

	return config.PostgresConfig{
		Host:     parsed.Hostname(),
		Port:     port,
		User:     parsed.User.Username(),
		Password: password,
		DBName:   parsed.Path[1:],
		SSLMode:  parsed.Query().Get("sslmode"),
	}
}
