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
	"log"
	"os/signal"
	"syscall"

	_ "github.com/rhajizada/llamero/docs"
	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/db"
	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/roles"
	"github.com/rhajizada/llamero/internal/server"
	"github.com/rhajizada/llamero/internal/service"
)

func main() {
	cfg, err := config.LoadServer()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	roleStore, err := roles.Load(roles.DefaultPath, cfg.Roles.Groups)
	if err != nil {
		log.Fatalf("load roles: %v", err)
	}

	dsn := cfg.Database.Postgres.DSN()
	if err := db.Migrate(ctx, dsn, cfg.Database.MigrationsDir); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	queries := repository.New(pool)
	cacheStore, err := redisstore.New(&cfg.Store)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	svc := service.New(queries, cacheStore)

	defs, err := config.LoadBackendDefinitions(cfg.Backends.FilePath)
	if err != nil {
		log.Fatalf("load backends: %v", err)
	}
	if err := svc.RegisterBackends(ctx, defs); err != nil {
		log.Fatalf("register backends: %v", err)
	}
	if err := svc.CheckBackends(ctx); err != nil {
		log.Printf("warn: initial backend health check failed: %v", err)
	}

	srv, err := server.New(cfg, roleStore, svc, log.Default())
	if err != nil {
		log.Fatalf("init server: %v", err)
	}

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
