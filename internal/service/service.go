package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rhajizada/llamero/internal/models"
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

// GetUser returns the API-facing user model for the provided ID.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (models.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, &Error{
				Code:    http.StatusNotFound,
				Message: "user not found",
				Err:     err,
			}
		}
		return models.User{}, &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to load user",
			Err:     err,
		}
	}
	return models.NewUserFromRepo(user), nil
}
