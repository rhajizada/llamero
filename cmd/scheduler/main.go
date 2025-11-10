package main

import (
	"log"

	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/workers"
)

func main() {
	cfg, err := config.LoadServer()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	connOpt := &asynq.RedisClientOpt{
		Addr:     cfg.Store.Addr,
		Username: cfg.Store.Username,
		Password: cfg.Store.Password,
		DB:       cfg.Store.DB,
	}

	scheduler := asynq.NewScheduler(connOpt, nil)
	task, err := workers.NewPingBackendsTask()
	if err != nil {
		log.Fatalf("create task: %v", err)
	}

	if _, err := scheduler.Register(cfg.Scheduler.BackendPingSpec, task); err != nil {
		log.Fatalf("register schedule: %v", err)
	}

	if err := scheduler.Run(); err != nil {
		log.Fatalf("scheduler stopped: %v", err)
	}
}
