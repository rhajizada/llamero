package main

import (
	"log/slog"
	"os"

	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/logging"
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

	connOpt := &asynq.RedisClientOpt{
		Addr:     cfg.Store.Addr,
		Username: cfg.Store.Username,
		Password: cfg.Store.Password,
		DB:       cfg.Store.DB,
	}

	scheduler := asynq.NewScheduler(connOpt, nil)
	task, err := workers.NewSyncBackendsTask()
	if err != nil {
		logger.Error("create task", "err", err)
		os.Exit(1)
	}

	if _, err := scheduler.Register(cfg.Scheduler.BackendPingSpec, task); err != nil {
		logger.Error("register schedule", "err", err)
		os.Exit(1)
	}

	if err := scheduler.Run(); err != nil {
		logger.Error("scheduler stopped", "err", err)
		os.Exit(1)
	}
}
