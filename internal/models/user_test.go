package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/repository"
)

func TestNewUserFromRepo(t *testing.T) {
	t.Parallel()

	now := time.Now()
	name := "Test User"
	user := repository.User{
		ID:          uuid.New(),
		Sub:         "sub-1",
		Provider:    "oauth",
		Email:       "user@example.com",
		DisplayName: &name,
		Role:        "admin",
		Scopes:      []string{"models:read"},
		Groups:      []string{"admins"},
		LastLoginAt: &now,
	}

	got := models.NewUserFromRepo(user)
	if got.ID != user.ID || got.Email != user.Email || got.Role != user.Role {
		t.Fatalf("unexpected converted user: %#v", got)
	}
}
