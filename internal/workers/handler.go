package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

// HandleSyncBackends refreshes metadata for all backends.
func (h *Handler) HandleSyncBackends(ctx context.Context, _ *asynq.Task) error {
	return h.svc.SyncBackends(ctx)
}

// HandleSyncBackendByID refreshes metadata for a specific backend.
func (h *Handler) HandleSyncBackendByID(ctx context.Context, task *asynq.Task) error {
	var payload SyncBackendPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	backendID := strings.TrimSpace(payload.BackendID)
	if backendID == "" {
		return fmt.Errorf("backend id is required")
	}
	return h.svc.SyncBackendByID(ctx, backendID)
}
