package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/hajizar/llamero/internal/config"
	"github.com/hajizar/llamero/internal/roles"
	"github.com/hajizar/llamero/internal/server"
)

func main() {
	cfg, err := config.LoadServer()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	roleStore, err := roles.Load(roles.DefaultPath, cfg.Roles.Groups)
	if err != nil {
		log.Fatalf("load roles: %v", err)
	}

	srv, err := server.New(cfg, roleStore, log.Default())
	if err != nil {
		log.Fatalf("init server: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
