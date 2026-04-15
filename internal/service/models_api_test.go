package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/service"
)

func TestAddModel(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "merges duplicate model with earliest created time",
			run: func(t *testing.T) {
				dest := map[string]models.Model{}

				service.AddModel(dest, " llama3 ", 100, "backend-a")
				service.AddModel(dest, "llama3", 50, "")

				assert.Equal(
					t,
					models.Model{ID: "llama3", Object: "model", Created: 50, OwnedBy: "backend-a"},
					dest["llama3"],
				)
			},
		},
		{
			name: "defaults owner and created timestamp",
			run: func(t *testing.T) {
				dest := map[string]models.Model{}

				service.AddModel(dest, "mistral", 0, "")

				assert.Equal(t, "mistral", dest["mistral"].ID)
				assert.Equal(t, service.DefaultModelOwner, dest["mistral"].OwnedBy)
				assert.Equal(t, "model", dest["mistral"].Object)
				assert.NotZero(t, dest["mistral"].Created)
			},
		},
		{
			name: "ignores empty model names",
			run: func(t *testing.T) {
				dest := map[string]models.Model{}

				service.AddModel(dest, " ", 123, "backend-a")

				assert.Empty(t, dest)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}
