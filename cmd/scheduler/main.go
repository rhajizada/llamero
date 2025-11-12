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

	if _, regErr := scheduler.Register(cfg.Scheduler.BackendPingSpec, task); regErr != nil {
		logger.Error("register schedule", "err", regErr)
		os.Exit(1)
	}

	if runErr := scheduler.Run(); runErr != nil {
		logger.Error("scheduler stopped", "err", runErr)
		os.Exit(1)
	}
}
