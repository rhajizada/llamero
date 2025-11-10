package workers

import (
	"context"

	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/service"
)

// Handler defines task handlers for Asynq.
type Handler struct {
	svc *service.Service
}

// NewHandler creates a handler instance.
func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// HandlePingBackends refreshes backend health info.
func (h *Handler) HandlePingBackends(ctx context.Context, _ *asynq.Task) error {
	return h.svc.CheckBackends(ctx)
}
