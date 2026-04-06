package workers_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hibiken/asynq"

	"github.com/rhajizada/llamero/internal/workers"
)

type fakeSyncService struct {
	syncBackendsFn    func(context.Context) error
	syncBackendByIDFn func(context.Context, string) error
}

func (f *fakeSyncService) SyncBackends(ctx context.Context) error {
	return f.syncBackendsFn(ctx)
}

func (f *fakeSyncService) SyncBackendByID(ctx context.Context, backendID string) error {
	return f.syncBackendByIDFn(ctx, backendID)
}

func TestNewHandlerAndHandleSyncBackends(t *testing.T) {
	t.Parallel()

	called := false
	h := workers.NewHandler(&fakeSyncService{
		syncBackendsFn: func(context.Context) error {
			called = true
			return nil
		},
		syncBackendByIDFn: func(context.Context, string) error { return nil },
	})

	if err := h.HandleSyncBackends(context.Background(), asynq.NewTask(workers.TypeSyncBackends, nil)); err != nil {
		t.Fatalf("HandleSyncBackends returned error: %v", err)
	}
	if !called {
		t.Fatal("expected SyncBackends to be called")
	}
}

func TestHandleSyncBackendByID(t *testing.T) {
	t.Parallel()

	calledWith := ""
	h := workers.NewHandler(&fakeSyncService{
		syncBackendsFn: func(context.Context) error { return nil },
		syncBackendByIDFn: func(_ context.Context, backendID string) error {
			calledWith = backendID
			return nil
		},
	})
	task, err := workers.NewSyncBackendByIDTask(" backend-a ")
	if err != nil {
		t.Fatalf("NewSyncBackendByIDTask returned error: %v", err)
	}

	if handleErr := h.HandleSyncBackendByID(context.Background(), task); handleErr != nil {
		t.Fatalf("HandleSyncBackendByID returned error: %v", handleErr)
	}
	if calledWith != "backend-a" {
		t.Fatalf("unexpected backend id: %q", calledWith)
	}
}

func TestHandleSyncBackendByIDErrors(t *testing.T) {
	t.Parallel()

	h := workers.NewHandler(&fakeSyncService{
		syncBackendsFn:    func(context.Context) error { return nil },
		syncBackendByIDFn: func(context.Context, string) error { return errors.New("sync failed") },
	})

	if handleErr := h.HandleSyncBackendByID(
		context.Background(),
		asynq.NewTask(workers.TypeSyncBackendByID, []byte("{")),
	); handleErr == nil || !strings.Contains(handleErr.Error(), "decode payload") {
		t.Fatalf("expected decode payload error, got %v", handleErr)
	}

	task := asynq.NewTask(workers.TypeSyncBackendByID, []byte(`{"backend_id":"   "}`))
	if handleErr := h.HandleSyncBackendByID(context.Background(), task); handleErr == nil ||
		!strings.Contains(handleErr.Error(), "backend id is required") {
		t.Fatalf("expected backend id error, got %v", handleErr)
	}

	task = asynq.NewTask(workers.TypeSyncBackendByID, []byte(`{"backend_id":"backend-a"}`))
	if handleErr := h.HandleSyncBackendByID(context.Background(), task); handleErr == nil ||
		!strings.Contains(handleErr.Error(), "sync failed") {
		t.Fatalf("expected sync error, got %v", handleErr)
	}
}
