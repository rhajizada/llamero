package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/logging"
	"github.com/rhajizada/llamero/internal/workers"
)

type schedulerRegistrar interface {
	Register(spec string, task *asynq.Task, opts ...asynq.Option) (string, error)
}

func main() {
	logger := logging.New()
	slog.SetDefault(logger)

	if err := Run(logger); err != nil {
		logger.Error("scheduler failed", "err", err)
		os.Exit(1)
	}
}

func Run(_ *slog.Logger) error {
	cfg, err := config.LoadScheduler()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	scheduler := asynq.NewScheduler(NewRedisClientOpt(cfg.Store), nil)
	if registerErr := RegisterBackendPingSchedule(scheduler, cfg.Scheduler.BackendPingSpec); registerErr != nil {
		return registerErr
	}

	return scheduler.Run()
}

func NewRedisClientOpt(cfg config.RedisConfig) *asynq.RedisClientOpt {
	return &asynq.RedisClientOpt{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
}

func RegisterBackendPingSchedule(scheduler schedulerRegistrar, spec string) error {
	task, err := workers.NewSyncBackendsTask()
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	if _, registerErr := scheduler.Register(spec, task); registerErr != nil {
		return fmt.Errorf("register schedule: %w", registerErr)
	}
	return nil
}
