package models

import (
	"time"

	"github.com/rhajizada/llamero/internal/repository"
)

// User represents the API view of a Llamero user.
type User struct {
	ID          string     `json:"id"`
	Provider    string     `json:"provider"`
	ExternalSub string     `json:"external_sub"`
	Email       string     `json:"email"`
	DisplayName *string    `json:"display_name,omitempty"`
	Role        string     `json:"role"`
	Scopes      []string   `json:"scopes"`
	Groups      []string   `json:"groups"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
} // @name User

// NewUserFromRepo builds a User model from a repository.User.
func NewUserFromRepo(u repository.User) User {
	return User{
		ID:          u.ID.String(),
		Provider:    u.Provider,
		ExternalSub: u.Sub,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		Scopes:      append([]string(nil), u.Scopes...),
		Groups:      append([]string(nil), u.Groups...),
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
