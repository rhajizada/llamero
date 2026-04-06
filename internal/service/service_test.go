package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/service"
)

func TestServiceBackendAndModelQueries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newTestRedisStore(t)
	svc := service.New(nil, store)

	now := time.Unix(1_700_000_000, 0)
	backendA := redisstore.BackendStatus{
		ID:           "backend-a",
		Address:      "http://backend-a:11434",
		Healthy:      true,
		LatencyMS:    10,
		Models:       []string{"llama3", "mistral"},
		LoadedModels: []string{"llama3"},
		ModelMeta: []redisstore.ModelInfo{
			{Name: "llama3", CreatedAt: now, OwnedBy: "backend-a"},
		},
		UpdatedAt: now,
	}
	backendB := redisstore.BackendStatus{
		ID:        "backend-b",
		Address:   "http://backend-b:11434",
		Healthy:   true,
		LatencyMS: 20,
		Models:    []string{"mistral", "phi4"},
		ModelMeta: []redisstore.ModelInfo{
			{Name: "mistral", CreatedAt: now.Add(time.Hour), OwnedBy: "backend-b"},
			{Name: "phi4", CreatedAt: now.Add(2 * time.Hour), OwnedBy: "backend-b"},
		},
		UpdatedAt: now.Add(time.Minute),
	}
	backendC := redisstore.BackendStatus{
		ID:      "backend-c",
		Address: "http://backend-c:11434",
		Healthy: false,
		Models:  []string{"deepseek"},
	}

	for i, backend := range []redisstore.BackendStatus{backendA, backendB, backendC} {
		if err := store.SaveBackend(ctx, backend, float64(i)); err != nil {
			t.Fatalf("SaveBackend(%s): %v", backend.ID, err)
		}
	}

	route, err := svc.RouteBackend(ctx, "llama3")
	if err != nil {
		t.Fatalf("RouteBackend llama3: %v", err)
	}
	if route.ID != "backend-a" {
		t.Fatalf("unexpected llama3 route: %#v", route)
	}

	route, err = svc.RouteBackend(ctx, "phi4")
	if err != nil {
		t.Fatalf("RouteBackend phi4: %v", err)
	}
	if route.ID != "backend-b" {
		t.Fatalf("unexpected phi4 route: %#v", route)
	}

	route, err = svc.LookupBackendRoute(ctx, "backend-b")
	if err != nil {
		t.Fatalf("LookupBackendRoute: %v", err)
	}
	if route.Address != "http://backend-b:11434" {
		t.Fatalf("unexpected lookup route: %#v", route)
	}

	list, err := svc.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if list.Object != "list" || len(list.Data) != 4 {
		t.Fatalf("unexpected model list: %#v", list)
	}
	if gotIDs := []string{
		list.Data[0].ID,
		list.Data[1].ID,
		list.Data[2].ID,
		list.Data[3].ID,
	}; !reflect.DeepEqual(gotIDs, []string{"deepseek", "llama3", "mistral", "phi4"}) {
		t.Fatalf("unexpected model ids: %#v", gotIDs)
	}

	model, err := svc.GetModel(ctx, "mistral")
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	if model.ID != "mistral" || model.OwnedBy != "backend-b" {
		t.Fatalf("unexpected model: %#v", model)
	}
}

func TestServiceRouteAndLookupErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newTestRedisStore(t)
	svc := service.New(nil, store)

	if _, err := svc.RouteBackend(ctx, "anything"); !errors.Is(err, service.ErrNoHealthyBackends) {
		t.Fatalf("expected ErrNoHealthyBackends, got %v", err)
	}

	if _, err := svc.LookupBackendRoute(ctx, ""); err == nil {
		t.Fatal("expected empty backend id lookup to fail")
	}
	if _, err := svc.GetModel(ctx, " "); err == nil {
		t.Fatal("expected empty model id to fail")
	}
	if _, err := svc.GetModel(ctx, "missing"); err == nil {
		t.Fatal("expected missing model to fail")
	}
}

func TestRegisterAndDeleteBackends(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newTestRedisStore(t)
	svc := service.New(nil, store)

	defs := []config.BackendDefinition{
		{ID: "backend-a", Address: "http://backend-a:11434", Tags: []string{"gpu"}, Weight: 2},
		{ID: "backend-b", Address: "http://backend-b:11434", Tags: []string{"cpu"}, Weight: 1},
	}
	if err := svc.RegisterBackends(ctx, defs); err != nil {
		t.Fatalf("RegisterBackends: %v", err)
	}

	ids, err := store.ListBackendIDs(ctx, 0, -1)
	if err != nil {
		t.Fatalf("ListBackendIDs: %v", err)
	}
	if !reflect.DeepEqual(ids, []string{"backend-a", "backend-b"}) {
		t.Fatalf("unexpected backend ids: %#v", ids)
	}

	if deleteErr := store.DeleteBackend(ctx, "backend-a"); deleteErr != nil {
		t.Fatalf("DeleteBackend: %v", deleteErr)
	}
	status, err := store.GetBackend(ctx, "backend-a")
	if err != nil {
		t.Fatalf("GetBackend after delete: %v", err)
	}
	if status.ID != "" {
		t.Fatalf("expected deleted backend to be missing, got %#v", status)
	}
}

func TestListBackends(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newTestRedisStore(t)
	svc := service.New(nil, store)
	now := time.Unix(1_700_000_000, 0)

	if err := store.SaveBackend(ctx, redisstore.BackendStatus{
		ID:           "backend-a",
		Address:      "http://backend-a:11434",
		Healthy:      true,
		LatencyMS:    15,
		Tags:         []string{"gpu"},
		Models:       []string{"llama3"},
		LoadedModels: []string{"llama3"},
		UpdatedAt:    now,
	}, 0); err != nil {
		t.Fatalf("SaveBackend: %v", err)
	}

	backends, err := svc.ListBackends(ctx)
	if err != nil {
		t.Fatalf("ListBackends returned error: %v", err)
	}
	if len(backends) != 1 {
		t.Fatalf("unexpected backends length: %d", len(backends))
	}
	if backends[0].ID != "backend-a" ||
		backends[0].LatencyMS != 15 ||
		!reflect.DeepEqual(backends[0].Models, []string{"llama3"}) {
		t.Fatalf("unexpected backend entry: %#v", backends[0])
	}
}

func TestSyncBackendsAndSyncBackendByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newTestRedisStore(t)
	svc := service.New(nil, store)
	svc.SetBackendPinger(func(_ context.Context, baseURL string) ([]redisstore.ModelInfo, []string, []string, error) {
		if baseURL == "http://bad-backend" {
			return nil, nil, nil, errors.New("backend down")
		}
		now := time.Unix(1_700_000_000, 0)
		return []redisstore.ModelInfo{{
			Name:      "llama3",
			CreatedAt: now,
			OwnedBy:   "library",
		}}, []string{"llama3"}, []string{"llama3"}, nil
	})

	for _, backend := range []redisstore.BackendStatus{
		{ID: "backend-a", Address: "http://backend-a:11434", Healthy: false},
		{ID: "backend-b", Address: "http://bad-backend", Healthy: true},
	} {
		if err := store.SaveBackend(ctx, backend, 0); err != nil {
			t.Fatalf("SaveBackend(%s): %v", backend.ID, err)
		}
	}

	if err := svc.SyncBackends(ctx); err != nil {
		t.Fatalf("SyncBackends returned error: %v", err)
	}

	backendA, err := store.GetBackend(ctx, "backend-a")
	if err != nil {
		t.Fatalf("GetBackend backend-a: %v", err)
	}
	if !backendA.Healthy ||
		!reflect.DeepEqual(backendA.Models, []string{"llama3"}) ||
		!reflect.DeepEqual(backendA.LoadedModels, []string{"llama3"}) {
		t.Fatalf("unexpected synced backend-a: %#v", backendA)
	}

	backendB, err := store.GetBackend(ctx, "backend-b")
	if err != nil {
		t.Fatalf("GetBackend backend-b: %v", err)
	}
	if backendB.Healthy {
		t.Fatalf("expected unhealthy backend-b after failed ping: %#v", backendB)
	}

	if syncErr := svc.SyncBackendByID(ctx, "backend-a"); syncErr != nil {
		t.Fatalf("SyncBackendByID returned error: %v", syncErr)
	}
	if syncErr := svc.SyncBackendByID(ctx, " "); syncErr == nil {
		t.Fatal("expected empty backend id to fail")
	}
	if syncErr := svc.SyncBackendByID(ctx, "missing"); syncErr == nil {
		t.Fatal("expected missing backend id to fail")
	}

	if saveErr := store.SaveBackend(ctx, redisstore.BackendStatus{ID: "backend-c", Address: "  "}, 0); saveErr != nil {
		t.Fatalf("SaveBackend backend-c: %v", saveErr)
	}
	if syncErr := svc.SyncBackendByID(ctx, "backend-c"); syncErr == nil {
		t.Fatal("expected missing backend address to fail")
	}
}

func TestFetchInstalledModelsAndRouteResponsesCreate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{{
					"name":        "llama3",
					"modified_at": "2026-04-05T12:00:00Z",
					"digest":      "abc",
					"description": "ignored",
				}},
			})
		case "/api/ps":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{{
					"name":       "llama3",
					"model":      "llama3",
					"expires_at": "2026-04-05T13:00:00Z",
				}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	store := newTestRedisStore(t)
	svc := service.New(nil, store)

	if err := store.SaveBackend(ctx, redisstore.BackendStatus{
		ID:      "backend-a",
		Address: server.URL,
		Healthy: true,
	}, 0); err != nil {
		t.Fatalf("SaveBackend: %v", err)
	}

	if err := svc.SyncBackendByID(ctx, "backend-a"); err != nil {
		t.Fatalf("SyncBackendByID returned error: %v", err)
	}
	route, err := svc.RouteResponsesCreate(ctx, "llama3")
	if err != nil {
		t.Fatalf("RouteResponsesCreate returned error: %v", err)
	}
	if route.ID != "backend-a" || route.Address != server.URL {
		t.Fatalf("unexpected responses route: %#v", route)
	}
	backend, err := store.GetBackend(ctx, "backend-a")
	if err != nil {
		t.Fatalf("GetBackend: %v", err)
	}
	if !reflect.DeepEqual(backend.Models, []string{"llama3"}) ||
		!reflect.DeepEqual(backend.LoadedModels, []string{"llama3"}) {
		t.Fatalf("unexpected fetched models: %#v", backend)
	}
}

func newTestRedisStore(t *testing.T) *redisstore.Store {
	t.Helper()

	mr := miniredis.RunT(t)
	store, err := redisstore.New(&config.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatalf("redisstore.New: %v", err)
	}
	return store
}
