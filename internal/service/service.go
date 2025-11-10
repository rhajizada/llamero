package service

import (
	"context"
	"net/http"

	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/repository"
)

// Service contains the business logic that interacts with persistence.
type Service struct {
	repo  *repository.Queries
	store *redisstore.Store
}

// New creates a Service instance.
func New(repo *repository.Queries, store *redisstore.Store) *Service {
	return &Service{
		repo:  repo,
		store: store,
	}
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
