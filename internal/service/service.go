package service

import (
	"context"
	"net/http"

	"github.com/rhajizada/llamero/internal/repository"
)

// Service contains the business logic that interacts with persistence.
type Service struct {
	repo *repository.Queries
}

// New creates a Service instance.
func New(repo *repository.Queries) *Service {
	return &Service{repo: repo}
}

// UpsertUser creates or updates a user record based on provider/sub.
func (s *Service) UpsertUser(ctx context.Context, params repository.UpsertUserParams) (repository.User, error) {
	user, err := s.repo.UpsertUser(ctx, params)
	if err != nil {
		return repository.User{}, &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to upsert user",
			Err:     err,
		}
	}
	return user, nil
}
