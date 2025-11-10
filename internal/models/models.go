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

// Backend represents backend metadata returned by the admin API.
type Backend struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Healthy   bool      `json:"healthy"`
	LatencyMS int64     `json:"latency_ms"`
	Tags      []string  `json:"tags"`
	Models    []string  `json:"models"`
	UpdatedAt time.Time `json:"updated_at"`
} // @name Backend

// Model represents an OpenAI-compatible model resource.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
} // @name Model

// ModelList is the response envelope for listing models.
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
} // @name ModelList

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
