package redisstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestRedisStoreLifecycle(t *testing.T) {
	t.Parallel()

	store := newRedisStore(t)

	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0)
	status := redisstore.BackendStatus{
		ID:           "backend-a",
		Address:      "http://backend-a:11434",
		Healthy:      true,
		LatencyMS:    12,
		Tags:         []string{"gpu"},
		Models:       []string{"llama3"},
		LoadedModels: []string{"llama3"},
		ModelMeta: []redisstore.ModelInfo{{
			Name:      "llama3",
			CreatedAt: now,
			OwnedBy:   "library",
		}},
		UpdatedAt: now,
	}
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "saves lists loads and deletes backend",
			run: func(t *testing.T) {
				require.NotNil(t, store.Client())
				require.NoError(t, store.SaveBackend(ctx, status, 1))

				ids, err := store.ListBackendIDs(ctx, 0, -1)
				require.NoError(t, err)
				assert.Equal(t, []string{"backend-a"}, ids)

				backends, err := store.ListBackends(ctx)
				require.NoError(t, err)
				require.Len(t, backends, 1)
				assert.Equal(t, status.Address, backends[0].Address)

				loaded, err := store.GetBackend(ctx, "backend-a")
				require.NoError(t, err)
				assert.Equal(t, "backend-a", loaded.ID)
				assert.True(t, loaded.UpdatedAt.Equal(now))

				require.NoError(t, store.DeleteBackend(ctx, "backend-a"))
				loaded, err = store.GetBackend(ctx, "backend-a")
				require.NoError(t, err)
				assert.Empty(t, loaded.ID)
			},
		},
		{
			name: "requires config",
			run: func(t *testing.T) {
				_, err := redisstore.New(nil)
				assert.Error(t, err)
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

func newRedisStore(t *testing.T) *redisstore.Store {
	t.Helper()

	return testutil.MustOpenRedisStore(t)
}
