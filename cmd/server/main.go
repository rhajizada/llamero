// @title Llamero API
// @version 1.0
// @description Llamero control plane API.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/rhajizada/llamero/docs"
	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/db"
	"github.com/rhajizada/llamero/internal/logging"
	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/roles"
	"github.com/rhajizada/llamero/internal/server"
	"github.com/rhajizada/llamero/internal/service"
)

func main() {
	logger := logging.New()
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.LoadServer()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	env, err := newServerEnvironment(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer env.Close()

	if syncErr := env.service.SyncBackends(ctx); syncErr != nil {
		logger.Warn("initial backend health check failed", "err", syncErr)
	}

	return env.server.Run(ctx)
}

type serverEnvironment struct {
	service *service.Service
	server  *server.Server
	closers []func()
}

func (e *serverEnvironment) Close() {
	for i := len(e.closers) - 1; i >= 0; i-- {
		if e.closers[i] != nil {
			e.closers[i]()
		}
	}
}

func (e *serverEnvironment) addCloser(fn func()) {
	if fn == nil {
		return
	}
	e.closers = append(e.closers, fn)
}

func newServerEnvironment(
	ctx context.Context,
	cfg *config.ServerConfig,
	logger *slog.Logger,
) (*serverEnvironment, error) {
	roleStore, err := roles.Load(roles.DefaultPath, cfg.Roles.Groups)
	if err != nil {
		return nil, fmt.Errorf("load roles: %w", err)
	}

	pool, err := setupDatabase(ctx, cfg)
	if err != nil {
		return nil, err
	}

	env := &serverEnvironment{}
	env.addCloser(pool.Close)

	cacheStore, err := redisstore.New(&cfg.Store)
	if err != nil {
		env.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	queries := repository.New(pool)
	svc := service.New(queries, cacheStore)

	defs, err := config.LoadBackendDefinitions(cfg.Backends.FilePath)
	if err != nil {
		env.Close()
		return nil, fmt.Errorf("load backends: %w", err)
	}
	if registerErr := svc.RegisterBackends(ctx, defs); registerErr != nil {
		env.Close()
		return nil, fmt.Errorf("register backends: %w", registerErr)
	}

	taskClient := asynq.NewClient(&asynq.RedisClientOpt{
		Addr:     cfg.Store.Addr,
		Username: cfg.Store.Username,
		Password: cfg.Store.Password,
		DB:       cfg.Store.DB,
	})
	env.addCloser(func() {
		if closeErr := taskClient.Close(); closeErr != nil {
			logger.Error("close task client", "err", closeErr)
		}
	})

	srv, err := server.New(cfg, roleStore, svc, taskClient, logger)
	if err != nil {
		env.Close()
		return nil, fmt.Errorf("init server: %w", err)
	}

	env.service = svc
	env.server = srv
	return env, nil
}

func setupDatabase(ctx context.Context, cfg *config.ServerConfig) (*pgxpool.Pool, error) {
	dsn := cfg.Database.Postgres.DSN()
	if err := db.Migrate(ctx, dsn, cfg.Database.MigrationsDir); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return pool, nil
}
