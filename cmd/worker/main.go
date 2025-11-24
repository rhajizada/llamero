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

	if err := run(); err != nil {
		logger.Error("worker failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadWorker()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	env, err := newWorkerEnvironment(ctx, cfg)
	if err != nil {
		return err
	}
	defer env.Close()

	handler := workers.NewHandler(env.service)
	env.mux.HandleFunc(workers.TypeSyncBackends, handler.HandleSyncBackends)
	env.mux.HandleFunc(workers.TypeSyncBackendByID, handler.HandleSyncBackendByID)

	return env.server.Run(env.mux)
}

type workerEnvironment struct {
	server  *asynq.Server
	mux     *asynq.ServeMux
	service *service.Service
	closers []func()
}

func (e *workerEnvironment) Close() {
	for i := len(e.closers) - 1; i >= 0; i-- {
		if e.closers[i] != nil {
			e.closers[i]()
		}
	}
}

func (e *workerEnvironment) addCloser(fn func()) {
	if fn == nil {
		return
	}
	e.closers = append(e.closers, fn)
}

func newWorkerEnvironment(ctx context.Context, cfg *config.WorkerConfig) (*workerEnvironment, error) {
	pool, err := prepareDatabase(ctx, cfg)
	if err != nil {
		return nil, err
	}

	env := &workerEnvironment{}
	env.addCloser(pool.Close)

	cacheStore, err := redisstore.New(&cfg.Store)
	if err != nil {
		env.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
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

	env.server = server
	env.mux = asynq.NewServeMux()
	env.service = svc
	return env, nil
}

func prepareDatabase(ctx context.Context, cfg *config.WorkerConfig) (*pgxpool.Pool, error) {
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
