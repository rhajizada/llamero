package models

import "github.com/rhajizada/llamero/internal/repository"

// User re-exports the repository user model for Swagger docs.
type User = repository.User // @name User

// NewUserFromRepo returns the repository user unchanged.
func NewUserFromRepo(u repository.User) User {
	return u
}
