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
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	cfg, err := config.LoadServer()
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	roleStore, err := roles.Load(roles.DefaultPath, cfg.Roles.Groups)
	if err != nil {
		logger.Error("load roles", "err", err)
		os.Exit(1)
	}

	dsn := cfg.Database.Postgres.DSN()
	if err := db.Migrate(ctx, dsn, cfg.Database.MigrationsDir); err != nil {
		logger.Error("migrate database", "err", err)
		os.Exit(1)
	}

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		logger.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := repository.New(pool)
	cacheStore, err := redisstore.New(&cfg.Store)
	if err != nil {
		logger.Error("connect redis", "err", err)
		os.Exit(1)
	}
	svc := service.New(queries, cacheStore)

	defs, err := config.LoadBackendDefinitions(cfg.Backends.FilePath)
	if err != nil {
		logger.Error("load backends", "err", err)
		os.Exit(1)
	}
	if err := svc.RegisterBackends(ctx, defs); err != nil {
		logger.Error("register backends", "err", err)
		os.Exit(1)
	}
	if err := svc.CheckBackends(ctx); err != nil {
		logger.Warn("initial backend health check failed", "err", err)
	}

	srv, err := server.New(cfg, roleStore, svc, logger)
	if err != nil {
		logger.Error("init server", "err", err)
		os.Exit(1)
	}

	if err := srv.Run(ctx); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}
