package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/repository"
)

func TestNewUserFromRepo(t *testing.T) {
	t.Parallel()

	now := time.Now()
	name := "Test User"
	tests := []struct {
		name   string
		user   repository.User
		assert func(*testing.T, models.User, repository.User)
	}{
		{
			name: "maps core fields",
			user: repository.User{
				ID:          uuid.New(),
				Sub:         "sub-1",
				Provider:    "oauth",
				Email:       "user@example.com",
				DisplayName: &name,
				Role:        "admin",
				Scopes:      []string{"models:read"},
				Groups:      []string{"admins"},
				LastLoginAt: &now,
			},
			assert: func(t *testing.T, got models.User, user repository.User) {
				assert.Equal(t, user.ID, got.ID)
				assert.Equal(t, user.Email, got.Email)
				assert.Equal(t, user.Role, got.Role)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := models.NewUserFromRepo(tc.user)
			tc.assert(t, got, tc.user)
		})
	}
}
