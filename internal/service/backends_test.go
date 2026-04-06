package service_test

import (
	"testing"
	"time"

	"github.com/ollama/ollama/api"

	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/service"
)

func TestAddListModelsAndAddProcessModels(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)
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

	if len(registry) != 2 {
		t.Fatalf("unexpected registry size: %d", len(registry))
	}
	if got := registry["llama3"]; got.OwnedBy != "registry.example.com" || !got.CreatedAt.Equal(now) {
		t.Fatalf("unexpected registry entry for llama3: %#v", got)
	}
	if got := registry["mistral"]; got.OwnedBy != service.DefaultModelOwner || !got.CreatedAt.Equal(later) {
		t.Fatalf("unexpected registry entry for mistral: %#v", got)
	}
}

func TestModelNameAndOwnerHelpers(t *testing.T) {
	t.Parallel()

	if got := service.FirstModelName("", " llama3 ", "mistral"); got != "llama3" {
		t.Fatalf("unexpected first model name: %q", got)
	}
	if got := service.InferOwner(""); got != service.DefaultModelOwner {
		t.Fatalf("unexpected default owner: %q", got)
	}
	if got := service.InferOwner("remote"); got != "remote" {
		t.Fatalf("unexpected explicit owner: %q", got)
	}
	if !service.Contains([]string{"llama3", "mistral"}, "mistral") {
		t.Fatal("expected contains to find existing value")
	}
	if service.Contains([]string{"llama3"}, "") {
		t.Fatal("expected empty target to fail contains")
	}
}
