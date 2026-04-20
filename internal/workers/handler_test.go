package workers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestWorkerHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "handles sync backends task",
			run: func(t *testing.T) {
				called := false
				h := workers.NewHandler(&fakeSyncService{
					syncBackendsFn: func(context.Context) error {
						called = true
						return nil
					},
					syncBackendByIDFn: func(context.Context, string) error { return nil },
				})

				require.NoError(
					t,
					h.HandleSyncBackends(
						context.Background(),
						asynq.NewTask(workers.TypeSyncBackends, nil),
					),
				)
				assert.True(t, called)
			},
		},
		{
			name: "handles sync backend by id task",
			run: func(t *testing.T) {
				calledWith := ""
				h := workers.NewHandler(&fakeSyncService{
					syncBackendsFn: func(context.Context) error { return nil },
					syncBackendByIDFn: func(_ context.Context, backendID string) error {
						calledWith = backendID
						return nil
					},
				})
				task, err := workers.NewSyncBackendByIDTask(" backend-a ")
				require.NoError(t, err)

				require.NoError(t, h.HandleSyncBackendByID(context.Background(), task))
				assert.Equal(t, "backend-a", calledWith)
			},
		},
		{
			name: "returns decode payload error",
			run: func(t *testing.T) {
				h := workers.NewHandler(&fakeSyncService{
					syncBackendsFn: func(context.Context) error { return nil },
					syncBackendByIDFn: func(context.Context, string) error {
						return errors.New("sync failed")
					},
				})

				err := h.HandleSyncBackendByID(
					context.Background(),
					asynq.NewTask(workers.TypeSyncBackendByID, []byte("{")),
				)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "decode payload")
			},
		},
		{
			name: "returns backend id validation error",
			run: func(t *testing.T) {
				h := workers.NewHandler(&fakeSyncService{
					syncBackendsFn: func(context.Context) error { return nil },
					syncBackendByIDFn: func(context.Context, string) error {
						return errors.New("sync failed")
					},
				})

				err := h.HandleSyncBackendByID(
					context.Background(),
					asynq.NewTask(
						workers.TypeSyncBackendByID,
						[]byte(`{"backend_id":"   "}`),
					),
				)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "backend id is required")
			},
		},
		{
			name: "returns sync service error",
			run: func(t *testing.T) {
				h := workers.NewHandler(&fakeSyncService{
					syncBackendsFn: func(context.Context) error { return nil },
					syncBackendByIDFn: func(context.Context, string) error {
						return errors.New("sync failed")
					},
				})

				err := h.HandleSyncBackendByID(
					context.Background(),
					asynq.NewTask(
						workers.TypeSyncBackendByID,
						[]byte(`{"backend_id":"backend-a"}`),
					),
				)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "sync failed")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}
