package workers_test

import (
	"encoding/json"
	"testing"

	"github.com/rhajizada/llamero/internal/workers"
)

func TestNewSyncBackendsTask(t *testing.T) {
	t.Parallel()

	task, err := workers.NewSyncBackendsTask()
	if err != nil {
		t.Fatalf("NewSyncBackendsTask returned error: %v", err)
	}
	if task.Type() != workers.TypeSyncBackends {
		t.Fatalf("unexpected task type: %s", task.Type())
	}
	if len(task.Payload()) != 0 {
		t.Fatalf("expected empty payload, got %q", string(task.Payload()))
	}
}

func TestNewSyncBackendByIDTask(t *testing.T) {
	t.Parallel()

	task, err := workers.NewSyncBackendByIDTask(" backend-1 ")
	if err != nil {
		t.Fatalf("NewSyncBackendByIDTask returned error: %v", err)
	}
	if task.Type() != workers.TypeSyncBackendByID {
		t.Fatalf("unexpected task type: %s", task.Type())
	}

	var payload workers.SyncBackendPayload
	if unmarshalErr := json.Unmarshal(task.Payload(), &payload); unmarshalErr != nil {
		t.Fatalf("unmarshal payload: %v", unmarshalErr)
	}
	if payload.BackendID != "backend-1" {
		t.Fatalf("unexpected backend id: %s", payload.BackendID)
	}
}

func TestNewSyncBackendByIDTaskRejectsEmptyID(t *testing.T) {
	t.Parallel()

	if _, err := workers.NewSyncBackendByIDTask("   "); err == nil {
		t.Fatal("expected empty backend id to fail")
	}
}
