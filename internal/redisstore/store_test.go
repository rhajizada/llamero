package redisstore_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/redisstore"
)

func TestStoreBackendLifecycle(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	store, err := redisstore.New(&config.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatalf("redisstore.New returned error: %v", err)
	}
	if store.Client() == nil {
		t.Fatal("expected redis client")
	}

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
	if saveErr := store.SaveBackend(ctx, status, 1); saveErr != nil {
		t.Fatalf("SaveBackend returned error: %v", saveErr)
	}

	ids, err := store.ListBackendIDs(ctx, 0, -1)
	if err != nil {
		t.Fatalf("ListBackendIDs returned error: %v", err)
	}
	if !reflect.DeepEqual(ids, []string{"backend-a"}) {
		t.Fatalf("unexpected ids: %#v", ids)
	}

	backends, err := store.ListBackends(ctx)
	if err != nil {
		t.Fatalf("ListBackends returned error: %v", err)
	}
	if len(backends) != 1 || backends[0].Address != status.Address {
		t.Fatalf("unexpected backends: %#v", backends)
	}

	loaded, err := store.GetBackend(ctx, "backend-a")
	if err != nil {
		t.Fatalf("GetBackend returned error: %v", err)
	}
	if loaded.ID != "backend-a" || !loaded.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected loaded backend: %#v", loaded)
	}

	if deleteErr := store.DeleteBackend(ctx, "backend-a"); deleteErr != nil {
		t.Fatalf("DeleteBackend returned error: %v", deleteErr)
	}
	loaded, err = store.GetBackend(ctx, "backend-a")
	if err != nil {
		t.Fatalf("GetBackend after delete returned error: %v", err)
	}
	if loaded.ID != "" {
		t.Fatalf("expected missing backend after delete, got %#v", loaded)
	}
}

func TestNewStoreRequiresConfig(t *testing.T) {
	t.Parallel()

	if _, err := redisstore.New(nil); err == nil {
		t.Fatal("expected nil config to fail")
	}
}
