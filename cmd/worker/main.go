package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/db"
	"github.com/rhajizada/llamero/internal/logging"
	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/service"
	"github.com/rhajizada/llamero/internal/workers"
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

	cacheStore, err := redisstore.New(&cfg.Store)
	if err != nil {
		logger.Error("connect redis", "err", err)
		os.Exit(1)
	}

	queries := repository.New(pool)
	svc := service.New(queries, cacheStore)

	connOpt := &asynq.RedisClientOpt{
		Addr:     cfg.Store.Addr,
		Username: cfg.Store.Username,
		Password: cfg.Store.Password,
		DB:       cfg.Store.DB,
	}

	server := asynq.NewServer(connOpt, asynq.Config{
		Concurrency: cfg.Worker.Concurrency,
		Queues: map[string]int{
			"default": 1,
		},
	})

	handler := workers.NewHandler(svc)
	mux := asynq.NewServeMux()
	mux.HandleFunc(workers.TypeSyncBackends, handler.HandleSyncBackends)
	mux.HandleFunc(workers.TypeSyncBackendByID, handler.HandleSyncBackendByID)

	if err := server.Run(mux); err != nil {
		logger.Error("worker stopped", "err", err)
		os.Exit(1)
	}
}
