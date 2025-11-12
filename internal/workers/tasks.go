package workers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
)

const (
	TypeSyncBackends    = "backends:sync"
	TypeSyncBackendByID = "backends:sync_by_id"
)

// SyncBackendPayload defines the task payload for syncing a single backend.
type SyncBackendPayload struct {
	BackendID string `json:"backend_id"`
}

// NewSyncBackendsTask enqueues a full backend sync.
func NewSyncBackendsTask() (*asynq.Task, error) {
	return asynq.NewTask(TypeSyncBackends, nil), nil
}

// NewSyncBackendByIDTask enqueues a sync for a specific backend.
func NewSyncBackendByIDTask(backendID string) (*asynq.Task, error) {
	backendID = strings.TrimSpace(backendID)
	if backendID == "" {
		return nil, fmt.Errorf("backend id is required")
	}
	payload, err := json.Marshal(SyncBackendPayload{BackendID: backendID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeSyncBackendByID, payload), nil
}
