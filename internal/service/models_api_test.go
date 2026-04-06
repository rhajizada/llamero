package service_test

import (
	"testing"

	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/service"
)

func TestAddModel(t *testing.T) {
	t.Parallel()

	dest := map[string]models.Model{}
	service.AddModel(dest, " llama3 ", 100, "backend-a")
	service.AddModel(dest, "llama3", 50, "")
	service.AddModel(dest, "mistral", 0, "")

	if got := dest["llama3"]; got.Created != 50 || got.OwnedBy != "backend-a" || got.Object != "model" {
		t.Fatalf("unexpected merged model: %#v", got)
	}
	if got := dest["mistral"]; got.OwnedBy != service.DefaultModelOwner || got.Created == 0 {
		t.Fatalf("unexpected defaulted model: %#v", got)
	}
}
