package workers

import (
	"github.com/hibiken/asynq"
)

const (
	TypePingBackends = "backends:ping"
)

// NewPingBackendsTask enqueues a backend health check.
func NewPingBackendsTask() (*asynq.Task, error) {
	return asynq.NewTask(TypePingBackends, nil), nil
}
