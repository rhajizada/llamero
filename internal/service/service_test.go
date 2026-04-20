package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/service"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestBackendRoutingAndModelQueries(t *testing.T) {
	ctx := context.Background()
	store := newTestRedisStore(t)
	svc := service.New(nil, store)
	now := time.Unix(1_700_000_000, 0)

	for i, backend := range []redisstore.BackendStatus{
		{
			ID:           "backend-a",
			Address:      "http://backend-a:11434",
			Healthy:      true,
			LatencyMS:    10,
			Models:       []string{"llama3", "mistral"},
			LoadedModels: []string{"llama3"},
			ModelMeta:    []redisstore.ModelInfo{{Name: "llama3", CreatedAt: now, OwnedBy: "backend-a"}},
			UpdatedAt:    now,
		},
		{
			ID:        "backend-b",
			Address:   "http://backend-b:11434",
			Healthy:   true,
			LatencyMS: 20,
			Models:    []string{"mistral", "phi4"},
			ModelMeta: []redisstore.ModelInfo{{Name: "mistral", CreatedAt: now.Add(time.Hour), OwnedBy: "backend-b"}, {Name: "phi4", CreatedAt: now.Add(2 * time.Hour), OwnedBy: "backend-b"}},
			UpdatedAt: now.Add(time.Minute),
		},
		{
			ID:      "backend-c",
			Address: "http://backend-c:11434",
			Healthy: false,
			Models:  []string{"deepseek"},
		},
	} {
		require.NoError(t, store.SaveBackend(ctx, backend, float64(i)))
	}

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "routes loaded and available models",
			run: func(t *testing.T) {
				route, err := svc.RouteBackend(ctx, "llama3")
				require.NoError(t, err)
				assert.Equal(t, "backend-a", route.ID)

				route, err = svc.RouteBackend(ctx, "phi4")
				require.NoError(t, err)
				assert.Equal(t, "backend-b", route.ID)
			},
		},
		{
			name: "looks up backend route by id",
			run: func(t *testing.T) {
				route, err := svc.LookupBackendRoute(ctx, "backend-b")
				require.NoError(t, err)
				assert.Equal(t, "http://backend-b:11434", route.Address)
			},
		},
		{
			name: "lists models in sorted order",
			run: func(t *testing.T) {
				list, err := svc.ListModels(ctx)
				require.NoError(t, err)
				require.Len(t, list.Data, 4)
				assert.Equal(t, "list", list.Object)
				assert.Equal(
					t,
					[]string{"deepseek", "llama3", "mistral", "phi4"},
					[]string{list.Data[0].ID, list.Data[1].ID, list.Data[2].ID, list.Data[3].ID},
				)
			},
		},
		{
			name: "gets model metadata",
			run: func(t *testing.T) {
				model, err := svc.GetModel(ctx, "mistral")
				require.NoError(t, err)
				assert.Equal(t, "mistral", model.ID)
				assert.Equal(t, "backend-b", model.OwnedBy)
			},
		},
		{
			name: "falls back to first healthy backend when model empty",
			run: func(t *testing.T) {
				route, err := svc.RouteBackend(ctx, "")
				require.NoError(t, err)
				assert.Equal(t, "backend-a", route.ID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestBackendRouteAndLookupErrors(t *testing.T) {
	ctx := context.Background()
	store := newTestRedisStore(t)
	svc := service.New(nil, store)

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "returns no healthy backends when empty",
			run: func(t *testing.T) {
				_, err := svc.RouteBackend(ctx, "anything")
				assert.ErrorIs(t, err, service.ErrNoHealthyBackends)
			},
		},
		{
			name: "rejects empty backend id lookup",
			run: func(t *testing.T) {
				_, err := svc.LookupBackendRoute(ctx, "")
				assertServiceError(t, err, http.StatusNotFound)
			},
		},
		{
			name: "rejects empty and missing model ids",
			run: func(t *testing.T) {
				_, err := svc.GetModel(ctx, " ")
				assertServiceError(t, err, http.StatusBadRequest)

				_, err = svc.GetModel(ctx, "missing")
				assertServiceError(t, err, http.StatusNotFound)
			},
		},
		{
			name: "returns backend missing address",
			run: func(t *testing.T) {
				require.NoError(t, store.SaveBackend(ctx, redisstore.BackendStatus{ID: "broken", Healthy: true}, 0))
				_, err := svc.LookupBackendRoute(ctx, "broken")
				assertServiceError(t, err, http.StatusBadGateway)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestRegisterBackends(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "registers and prunes backends",
			run: func(t *testing.T) {
				store := newTestRedisStore(t)
				svc := service.New(nil, store)
				require.NoError(
					t,
					store.SaveBackend(ctx, redisstore.BackendStatus{ID: "stale", Address: "http://stale"}, 0),
				)

				defs := []config.BackendDefinition{
					{ID: "backend-a", Address: "http://backend-a:11434", Tags: []string{"gpu"}, Weight: 2},
					{ID: "backend-b", Address: "http://backend-b:11434", Tags: []string{"cpu"}, Weight: 1},
				}
				require.NoError(t, svc.RegisterBackends(ctx, defs))

				ids, err := store.ListBackendIDs(ctx, 0, -1)
				require.NoError(t, err)
				assert.Equal(t, []string{"backend-a", "backend-b"}, ids)
			},
		},
		{
			name: "rejects duplicate backend ids",
			run: func(t *testing.T) {
				store := newTestRedisStore(t)
				svc := service.New(nil, store)
				err := svc.RegisterBackends(
					ctx,
					[]config.BackendDefinition{{ID: "dup", Address: "http://a"}, {ID: "dup", Address: "http://b"}},
				)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "duplicate backend id")
			},
		},
		{
			name: "rejects reused backend addresses",
			run: func(t *testing.T) {
				store := newTestRedisStore(t)
				svc := service.New(nil, store)
				err := svc.RegisterBackends(
					ctx,
					[]config.BackendDefinition{
						{ID: "a", Address: "http://shared"},
						{ID: "b", Address: "http://shared"},
					},
				)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "reused")
			},
		},
		{
			name: "rejects missing id or address",
			run: func(t *testing.T) {
				store := newTestRedisStore(t)
				svc := service.New(nil, store)
				err := svc.RegisterBackends(ctx, []config.BackendDefinition{{ID: " ", Address: "http://backend"}})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "missing id or address")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestListBackends(t *testing.T) {
	ctx := context.Background()
	store := newTestRedisStore(t)
	svc := service.New(nil, store)
	now := time.Unix(1_700_000_000, 0)

	require.NoError(t, store.SaveBackend(ctx, redisstore.BackendStatus{
		ID:           "backend-a",
		Address:      "http://backend-a:11434",
		Healthy:      true,
		LatencyMS:    15,
		Tags:         []string{"gpu"},
		Models:       []string{"llama3"},
		LoadedModels: []string{"llama3"},
		UpdatedAt:    now,
	}, 0))

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "lists backend statuses",
			run: func(t *testing.T) {
				backends, err := svc.ListBackends(ctx)
				require.NoError(t, err)
				require.Len(t, backends, 1)
				assert.Equal(t, "backend-a", backends[0].ID)
				assert.Equal(t, int64(15), backends[0].LatencyMS)
				assert.Equal(t, []string{"llama3"}, backends[0].Models)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestSyncBackends(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "syncs healthy and unhealthy backends",
			run: func(t *testing.T) {
				store := newTestRedisStore(t)
				svc := service.New(nil, store)
				svc.SetBackendPinger(
					func(_ context.Context, baseURL string) ([]redisstore.ModelInfo, []string, []string, error) {
						if baseURL == "http://bad-backend" {
							return nil, nil, nil, errors.New("backend down")
						}
						now := time.Unix(1_700_000_000, 0)
						return []redisstore.ModelInfo{
								{Name: "llama3", CreatedAt: now, OwnedBy: "library"},
							}, []string{
								"llama3",
							}, []string{
								"llama3",
							}, nil
					},
				)

				for _, backend := range []redisstore.BackendStatus{{ID: "backend-a", Address: "http://backend-a:11434", Healthy: false}, {ID: "backend-b", Address: "http://bad-backend", Healthy: true}} {
					require.NoError(t, store.SaveBackend(ctx, backend, 0))
				}

				require.NoError(t, svc.SyncBackends(ctx))

				backendA, err := store.GetBackend(ctx, "backend-a")
				require.NoError(t, err)
				assert.True(t, backendA.Healthy)
				assert.Equal(t, []string{"llama3"}, backendA.Models)
				assert.Equal(t, []string{"llama3"}, backendA.LoadedModels)

				backendB, err := store.GetBackend(ctx, "backend-b")
				require.NoError(t, err)
				assert.False(t, backendB.Healthy)
			},
		},
		{
			name: "sync backend by id handles validation errors",
			run: func(t *testing.T) {
				store := newTestRedisStore(t)
				svc := service.New(nil, store)
				svc.SetBackendPinger(
					func(_ context.Context, _ string) ([]redisstore.ModelInfo, []string, []string, error) {
						return []redisstore.ModelInfo{
								{Name: "llama3", CreatedAt: time.Unix(1_700_000_000, 0), OwnedBy: "library"},
							}, []string{
								"llama3",
							}, []string{
								"llama3",
							}, nil
					},
				)

				require.NoError(
					t,
					store.SaveBackend(
						ctx,
						redisstore.BackendStatus{ID: "backend-a", Address: "http://backend-a:11434", Healthy: false},
						0,
					),
				)
				require.NoError(t, svc.SyncBackendByID(ctx, "backend-a"))

				require.Error(t, svc.SyncBackendByID(ctx, " "))
				require.Error(t, svc.SyncBackendByID(ctx, "missing"))

				require.NoError(t, store.SaveBackend(ctx, redisstore.BackendStatus{ID: "backend-c", Address: "  "}, 0))
				require.Error(t, svc.SyncBackendByID(ctx, "backend-c"))
			},
		},
		{
			name: "set backend pinger resets to default when nil",
			run: func(t *testing.T) {
				store := newTestRedisStore(t)
				svc := service.New(nil, store)
				svc.SetBackendPinger(func(context.Context, string) ([]redisstore.ModelInfo, []string, []string, error) {
					return nil, nil, nil, nil
				})
				svc.SetBackendPinger(nil)
				require.NoError(
					t,
					store.SaveBackend(ctx, redisstore.BackendStatus{ID: "backend-a", Address: "http://127.0.0.1:1"}, 0),
				)
				require.NoError(t, svc.SyncBackends(ctx))
				backend, err := store.GetBackend(ctx, "backend-a")
				require.NoError(t, err)
				assert.False(t, backend.Healthy)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestFetchInstalledModelsAndRouteResponsesCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			assert.NoError(
				t,
				json.NewEncoder(w).
					Encode(map[string]any{"models": []map[string]any{{"name": "llama3", "modified_at": "2026-04-05T12:00:00Z", "digest": "abc", "description": "ignored"}}}),
			)
		case "/api/ps":
			assert.NoError(
				t,
				json.NewEncoder(w).
					Encode(map[string]any{"models": []map[string]any{{"name": "llama3", "model": "llama3", "expires_at": "2026-04-05T13:00:00Z"}}}),
			)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	store := newTestRedisStore(t)
	svc := service.New(nil, store)
	require.NoError(
		t,
		store.SaveBackend(ctx, redisstore.BackendStatus{ID: "backend-a", Address: server.URL, Healthy: true}, 0),
	)

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "syncs remote models and routes responses create",
			run: func(t *testing.T) {
				require.NoError(t, svc.SyncBackendByID(ctx, "backend-a"))

				route, err := svc.RouteResponsesCreate(ctx, "llama3")
				require.NoError(t, err)
				assert.Equal(t, "backend-a", route.ID)
				assert.Equal(t, server.URL, route.Address)

				backend, err := store.GetBackend(ctx, "backend-a")
				require.NoError(t, err)
				assert.Equal(t, []string{"llama3"}, backend.Models)
				assert.Equal(t, []string{"llama3"}, backend.LoadedModels)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestUserServiceUsesDatabase(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	updatedAt := now.Add(5 * time.Minute)
	displayName := "Initial User"
	updatedName := "Updated User"

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{name: "upserts and fetches user", run: func(t *testing.T) {
			repo := repository.New(testutil.MustOpenMigratedPostgres(t))
			svc := service.New(repo, nil)
			provider := "oidc-" + uuid.NewString()
			sub := "sub-" + uuid.NewString()

			created, err := svc.UpsertUser(
				ctx,
				repository.UpsertUserParams{
					Sub:         sub,
					Provider:    provider,
					Email:       uuid.NewString() + "@example.com",
					DisplayName: &displayName,
					Role:        "admin",
					Scopes:      []string{"models:list"},
					Groups:      []string{"admins"},
					LastLoginAt: &now,
				},
			)
			require.NoError(t, err)

			fetched, getErr := svc.GetUser(ctx, created.ID)
			require.NoError(t, getErr)
			assert.Equal(t, created.ID, fetched.ID)
			assert.Equal(t, created.Email, fetched.Email)
		}},
		{name: "upsert updates existing user", run: func(t *testing.T) {
			repo := repository.New(testutil.MustOpenMigratedPostgres(t))
			svc := service.New(repo, nil)
			provider := "oidc-" + uuid.NewString()
			sub := "sub-" + uuid.NewString()
			email := uuid.NewString() + "@example.com"

			created, err := svc.UpsertUser(
				ctx,
				repository.UpsertUserParams{
					Sub:         sub,
					Provider:    provider,
					Email:       email,
					DisplayName: &displayName,
					Role:        "member",
					Scopes:      []string{"models:list"},
					Groups:      []string{"users"},
					LastLoginAt: &now,
				},
			)
			require.NoError(t, err)

			updated, updateErr := svc.UpsertUser(
				ctx,
				repository.UpsertUserParams{
					Sub:         sub,
					Provider:    provider,
					Email:       email,
					DisplayName: &updatedName,
					Role:        "admin",
					Scopes:      []string{"models:list", "tokens:write"},
					Groups:      []string{"admins"},
					LastLoginAt: &updatedAt,
				},
			)
			require.NoError(t, updateErr)
			assert.Equal(t, created.ID, updated.ID)
			assert.Equal(t, "admin", updated.Role)
			require.NotNil(t, updated.DisplayName)
			assert.Equal(t, updatedName, *updated.DisplayName)
		}},
		{name: "returns not found for missing user", run: func(t *testing.T) {
			repo := repository.New(testutil.MustOpenMigratedPostgres(t))
			svc := service.New(repo, nil)
			_, err := svc.GetUser(ctx, uuid.New())
			assertServiceError(t, err, http.StatusNotFound)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func seedServiceUser(ctx context.Context, t *testing.T, repo *repository.Queries) repository.User {
	t.Helper()

	displayName := "Test User"
	lastLoginAt := time.Now().UTC().Truncate(time.Second)
	user, err := repo.UpsertUser(
		ctx,
		repository.UpsertUserParams{
			Sub:         "sub-" + uuid.NewString(),
			Provider:    "oidc-" + uuid.NewString(),
			Email:       uuid.NewString() + "@example.com",
			DisplayName: &displayName,
			Role:        "member",
			Scopes:      []string{"models:list"},
			Groups:      []string{"users"},
			LastLoginAt: &lastLoginAt,
		},
	)
	require.NoError(t, err)
	return user
}

func assertServiceError(t *testing.T, err error, code int) {
	t.Helper()

	var serviceErr *service.Error
	require.Error(t, err)
	require.ErrorAs(t, err, &serviceErr)
	assert.Equal(t, code, serviceErr.Code)
}

func newTestRedisStore(t *testing.T) *redisstore.Store {
	t.Helper()

	return testutil.MustOpenRedisStore(t)
}
