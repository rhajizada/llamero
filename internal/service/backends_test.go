package service_test

import (
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"

	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/service"
)

func TestModelRegistryHelpers(t *testing.T) {
	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "adds list and process models",
			run: func(t *testing.T) {
				registry := map[string]redisstore.ModelInfo{}

				service.AddListModels(registry, []api.ListModelResponse{{
					Name:       "llama3",
					Model:      "ignored",
					ModifiedAt: now,
					RemoteHost: "registry.example.com",
				}})
				service.AddProcessModels(registry, []api.ProcessModelResponse{{
					Name:      "llama3",
					ExpiresAt: later,
				}, {
					Name:      "mistral",
					ExpiresAt: later,
				}})

				assert.Len(t, registry, 2)
				assert.Equal(t, redisstore.ModelInfo{Name: "llama3", CreatedAt: now, OwnedBy: "registry.example.com"}, registry["llama3"])
				assert.Equal(t, redisstore.ModelInfo{Name: "mistral", CreatedAt: later, OwnedBy: service.DefaultModelOwner}, registry["mistral"])
			},
		},
		{
			name: "uses defaults for zero values",
			run: func(t *testing.T) {
				registry := map[string]redisstore.ModelInfo{}

				service.AddListModels(registry, []api.ListModelResponse{{Name: "  ", Model: ""}})
				service.AddProcessModels(registry, []api.ProcessModelResponse{{Name: "phi4"}})

				assert.Len(t, registry, 1)
				assert.Equal(t, "phi4", registry["phi4"].Name)
				assert.Equal(t, service.DefaultModelOwner, registry["phi4"].OwnedBy)
				assert.False(t, registry["phi4"].CreatedAt.IsZero())
			},
		},
		{
			name: "prefers first non-empty model name",
			run: func(t *testing.T) {
				registry := map[string]redisstore.ModelInfo{}

				service.AddListModels(registry, []api.ListModelResponse{{Name: " ", Model: "llama3", ModifiedAt: now}})

				assert.Contains(t, registry, "llama3")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestModelNameAndOwnerHelpers(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "first model name trims values",
			run: func(t *testing.T) {
				assert.Equal(t, "llama3", service.FirstModelName("", " llama3 ", "mistral"))
			},
		},
		{
			name: "infer owner defaults to library",
			run: func(t *testing.T) {
				assert.Equal(t, service.DefaultModelOwner, service.InferOwner(""))
			},
		},
		{
			name: "infer owner preserves explicit host",
			run: func(t *testing.T) {
				assert.Equal(t, "remote", service.InferOwner("remote"))
			},
		},
		{
			name: "contains finds existing value",
			run: func(t *testing.T) {
				assert.True(t, service.Contains([]string{"llama3", "mistral"}, "mistral"))
			},
		},
		{
			name: "contains rejects empty target",
			run: func(t *testing.T) {
				assert.False(t, service.Contains([]string{"llama3"}, ""))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}
