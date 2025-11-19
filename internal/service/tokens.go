package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/repository"
)

// CreateTokenParams captures the data required to persist a PAT.
type CreateTokenParams struct {
	UserID    uuid.UUID
	Name      string
	Scopes    []string
	TokenType string
	JTI       string
	ExpiresAt time.Time
}

// CreatePersonalAccessToken stores PAT metadata for the supplied user.
func (s *Service) CreatePersonalAccessToken(
	ctx context.Context,
	params CreateTokenParams,
) (models.PersonalAccessToken, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return models.PersonalAccessToken{}, &Error{
			Code:    http.StatusBadRequest,
			Message: "token name is required",
		}
	}
	if params.UserID == uuid.Nil {
		return models.PersonalAccessToken{}, &Error{
			Code:    http.StatusBadRequest,
			Message: "user id is required",
		}
	}
	if len(params.Scopes) == 0 {
		return models.PersonalAccessToken{}, &Error{
			Code:    http.StatusBadRequest,
			Message: "at least one scope is required",
		}
	}
	if params.ExpiresAt.Before(time.Now()) {
		return models.PersonalAccessToken{}, &Error{
			Code:    http.StatusBadRequest,
			Message: "expiration must be in the future",
		}
	}
	if params.JTI == "" {
		return models.PersonalAccessToken{}, &Error{
			Code:    http.StatusBadRequest,
			Message: "token identifier is required",
		}
	}
	tokenType := strings.TrimSpace(params.TokenType)
	if tokenType == "" {
		tokenType = auth.TokenTypePAT
	}

	record, err := s.repo.CreateToken(ctx, repository.CreateTokenParams{
		UserID:    params.UserID,
		Name:      name,
		Scopes:    params.Scopes,
		TokenType: tokenType,
		Jti:       params.JTI,
		ExpiresAt: params.ExpiresAt,
	})
	if err != nil {
		return models.PersonalAccessToken{}, &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to create token",
			Err:     err,
		}
	}
	return models.PersonalAccessToken(record), nil
}

// ListPersonalAccessTokens returns PAT metadata for a user ordered by creation time.
func (s *Service) ListPersonalAccessTokens(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.PersonalAccessToken, error) {
	if userID == uuid.Nil {
		return nil, &Error{
			Code:    http.StatusBadRequest,
			Message: "user id is required",
		}
	}
	records, err := s.repo.ListTokensByUser(ctx, userID)
	if err != nil {
		return nil, &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to list tokens",
			Err:     err,
		}
	}
	out := make([]models.PersonalAccessToken, 0, len(records))
	for _, rec := range records {
		out = append(out, models.PersonalAccessToken(rec))
	}
	return out, nil
}

// GetPersonalAccessToken fetches a specific PAT by identifier for a user.
func (s *Service) GetPersonalAccessToken(
	ctx context.Context,
	userID, tokenID uuid.UUID,
) (models.PersonalAccessToken, error) {
	if userID == uuid.Nil || tokenID == uuid.Nil {
		return models.PersonalAccessToken{}, &Error{
			Code:    http.StatusBadRequest,
			Message: "token id is required",
		}
	}
	record, err := s.repo.GetTokenByID(ctx, repository.GetTokenByIDParams{
		ID:     tokenID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.PersonalAccessToken{}, &Error{
				Code:    http.StatusNotFound,
				Message: "token not found",
				Err:     err,
			}
		}
		return models.PersonalAccessToken{}, &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to load token",
			Err:     err,
		}
	}
	return models.PersonalAccessToken(record), nil
}

// RevokePersonalAccessToken marks a PAT as revoked.
func (s *Service) RevokePersonalAccessToken(ctx context.Context, userID, tokenID uuid.UUID) error {
	if userID == uuid.Nil || tokenID == uuid.Nil {
		return &Error{
			Code:    http.StatusBadRequest,
			Message: "token id is required",
		}
	}
	_, err := s.repo.RevokeToken(ctx, repository.RevokeTokenParams{
		ID:     tokenID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &Error{
				Code:    http.StatusNotFound,
				Message: "token not found",
				Err:     err,
			}
		}
		return &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to revoke token",
			Err:     err,
		}
	}
	return nil
}

// ValidatePAT ensures the PAT represented by the claims is active and owned by the user.
func (s *Service) ValidatePAT(ctx context.Context, claims *auth.Claims) error {
	if claims == nil {
		return &Error{
			Code:    http.StatusUnauthorized,
			Message: "missing token claims",
		}
	}
	if strings.TrimSpace(claims.ID) == "" {
		return &Error{
			Code:    http.StatusUnauthorized,
			Message: "token identifier is missing",
		}
	}

	record, err := s.repo.GetTokenByJTI(ctx, claims.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &Error{
				Code:    http.StatusUnauthorized,
				Message: "token has been revoked",
				Err:     err,
			}
		}
		return &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to validate token",
			Err:     err,
		}
	}

	if record.TokenType != auth.TokenTypePAT {
		return &Error{
			Code:    http.StatusUnauthorized,
			Message: "token type mismatch",
		}
	}
	if record.Revoked {
		return &Error{
			Code:    http.StatusUnauthorized,
			Message: "token has been revoked",
		}
	}
	if record.UserID.String() != claims.Subject {
		return &Error{
			Code:    http.StatusUnauthorized,
			Message: "token subject mismatch",
		}
	}
	if time.Now().After(record.ExpiresAt) {
		return &Error{
			Code:    http.StatusUnauthorized,
			Message: "token has expired",
		}
	}
	if tuErr := s.repo.MarkTokenUsed(ctx, record.ID); tuErr != nil {
		return &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to record token usage",
			Err:     tuErr,
		}
	}
	return nil
}
