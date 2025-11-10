package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/db"
	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/service"
	"github.com/rhajizada/llamero/internal/workers"
)

func main() {
	cfg, err := config.LoadServer()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dsn := cfg.Database.Postgres.DSN()
	if err := db.Migrate(ctx, dsn, cfg.Database.MigrationsDir); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	cacheStore, err := redisstore.New(&cfg.Store)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
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
	mux.HandleFunc(workers.TypePingBackends, handler.HandlePingBackends)

	if err := server.Run(mux); err != nil {
		log.Fatalf("worker stopped: %v", err)
	}
}
