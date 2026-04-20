package main

import (
	"net/url"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestPrepareDatabase(t *testing.T) {
	t.Parallel()

	ctx, dsn := testutil.MustStartPostgres(t)
	tests := []struct {
		name      string
		cfg       *config.WorkerConfig
		wantErr   string
		checkPool bool
	}{
		{
			name:      "migrates and connects",
			cfg:       &config.WorkerConfig{Database: config.DatabaseConfig{Postgres: mustWorkerPostgresConfig(t, dsn), MigrationsDir: testutil.MigrationsDir(t)}},
			checkPool: true,
		},
		{
			name:    "fails on missing migrations dir",
			cfg:     &config.WorkerConfig{Database: config.DatabaseConfig{Postgres: mustWorkerPostgresConfig(t, dsn), MigrationsDir: filepath.Join(t.TempDir(), "missing")}},
			wantErr: "migrate database",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := PrepareDatabase(ctx, tc.cfg)
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

func TestNewWorkerEnvironment(t *testing.T) {
	t.Parallel()

	ctx, dsn := testutil.MustStartPostgres(t)
	_, redisAddr := testutil.MustStartRedis(t)
	baseCfg := config.WorkerConfig{
		Database: config.DatabaseConfig{Postgres: mustWorkerPostgresConfig(t, dsn), MigrationsDir: testutil.MigrationsDir(t)},
		Store:    config.RedisConfig{Addr: redisAddr},
		Worker:   config.WorkerSettings{Concurrency: 2},
	}

	tests := []struct {
		name    string
		mutate  func(*config.WorkerConfig)
		wantErr string
	}{
		{name: "builds environment"},
		{
			name: "fails on invalid redis address",
			mutate: func(cfg *config.WorkerConfig) {
				cfg.Store.Addr = "127.0.0.1:0"
			},
			wantErr: "connect redis",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseCfg
			if tc.mutate != nil {
				tc.mutate(&cfg)
			}

			env, err := NewWorkerEnvironment(ctx, &cfg)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, env)
			require.NotNil(t, env.server)
			require.NotNil(t, env.mux)
			require.NotNil(t, env.service)
			t.Cleanup(env.Close)
		})
	}
}

func mustWorkerPostgresConfig(t *testing.T, dsn string) config.PostgresConfig {
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
