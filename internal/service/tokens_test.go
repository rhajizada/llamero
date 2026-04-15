package service_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/service"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestCreatePersonalAccessToken(t *testing.T) {
	ctx := context.Background()
	repo := repository.New(testutil.MustOpenMigratedPostgres(t))
	user := seedServiceUser(t, ctx, repo)
	svc := service.New(repo, nil)

	tests := []struct {
		name     string
		params   service.CreateTokenParams
		wantCode int
		wantType string
	}{
		{name: "creates token with default type", params: service.CreateTokenParams{UserID: user.ID, Name: " cli ", Scopes: []string{"models:list"}, JTI: uuid.NewString(), ExpiresAt: time.Now().Add(time.Hour)}, wantType: auth.TokenTypePAT},
		{name: "creates token with explicit type", params: service.CreateTokenParams{UserID: user.ID, Name: "automation", Scopes: []string{"models:list", "tokens:read"}, TokenType: auth.TokenTypePAT, JTI: uuid.NewString(), ExpiresAt: time.Now().Add(2 * time.Hour)}, wantType: auth.TokenTypePAT},
		{name: "rejects missing name", params: service.CreateTokenParams{UserID: user.ID, Scopes: []string{"models:list"}, JTI: uuid.NewString(), ExpiresAt: time.Now().Add(time.Hour)}, wantCode: http.StatusBadRequest},
		{name: "rejects missing user id", params: service.CreateTokenParams{Name: "cli", Scopes: []string{"models:list"}, JTI: uuid.NewString(), ExpiresAt: time.Now().Add(time.Hour)}, wantCode: http.StatusBadRequest},
		{name: "rejects missing scopes", params: service.CreateTokenParams{UserID: user.ID, Name: "cli", JTI: uuid.NewString(), ExpiresAt: time.Now().Add(time.Hour)}, wantCode: http.StatusBadRequest},
		{name: "rejects missing jti", params: service.CreateTokenParams{UserID: user.ID, Name: "cli", Scopes: []string{"models:list"}, ExpiresAt: time.Now().Add(time.Hour)}, wantCode: http.StatusBadRequest},
		{name: "rejects expired token", params: service.CreateTokenParams{UserID: user.ID, Name: "cli", Scopes: []string{"models:list"}, JTI: uuid.NewString(), ExpiresAt: time.Now().Add(-time.Hour)}, wantCode: http.StatusBadRequest},
		{name: "surfaces repository failure for missing user", params: service.CreateTokenParams{UserID: uuid.New(), Name: "cli", Scopes: []string{"models:list"}, JTI: uuid.NewString(), ExpiresAt: time.Now().Add(time.Hour)}, wantCode: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := svc.CreatePersonalAccessToken(ctx, tc.params)
			if tc.wantCode != 0 {
				assertServiceError(t, err, tc.wantCode)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(tc.params.Name), token.Name)
			assert.Equal(t, tc.wantType, token.TokenType)
			assert.Equal(t, tc.params.UserID, token.UserID)
			assert.Equal(t, tc.params.Scopes, token.Scopes)
			stored, getErr := repo.GetTokenByJTI(ctx, tc.params.JTI)
			require.NoError(t, getErr)
			assert.Equal(t, token.ID, stored.ID)
		})
	}
}

func TestPersonalAccessTokenLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := repository.New(testutil.MustOpenMigratedPostgres(t))
	user := seedServiceUser(t, ctx, repo)
	svc := service.New(repo, nil)

	older := mustCreateServiceToken(t, ctx, repo, user.ID, "older")
	time.Sleep(10 * time.Millisecond)
	newer := mustCreateServiceToken(t, ctx, repo, user.ID, "newer")
	revoked := mustCreateServiceToken(t, ctx, repo, user.ID, "revoked")
	_, err := repo.RevokeToken(ctx, repository.RevokeTokenParams{ID: revoked.ID, UserID: user.ID})
	require.NoError(t, err)

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{name: "lists active tokens in descending order", run: func(t *testing.T) {
			tokens, listErr := svc.ListPersonalAccessTokens(ctx, user.ID)
			require.NoError(t, listErr)
			require.Len(t, tokens, 2)
			assert.Equal(t, []uuid.UUID{newer.ID, older.ID}, []uuid.UUID{tokens[0].ID, tokens[1].ID})
		}},
		{name: "gets token by id", run: func(t *testing.T) {
			token, getErr := svc.GetPersonalAccessToken(ctx, user.ID, newer.ID)
			require.NoError(t, getErr)
			assert.Equal(t, newer.ID, token.ID)
		}},
		{name: "returns not found for missing token", run: func(t *testing.T) {
			_, getErr := svc.GetPersonalAccessToken(ctx, user.ID, uuid.New())
			assertServiceError(t, getErr, http.StatusNotFound)
		}},
		{name: "rejects missing ids", run: func(t *testing.T) {
			_, listErr := svc.ListPersonalAccessTokens(ctx, uuid.Nil)
			assertServiceError(t, listErr, http.StatusBadRequest)
			_, getErr := svc.GetPersonalAccessToken(ctx, uuid.Nil, newer.ID)
			assertServiceError(t, getErr, http.StatusBadRequest)
			revokeErr := svc.RevokePersonalAccessToken(ctx, uuid.Nil, newer.ID)
			assertServiceError(t, revokeErr, http.StatusBadRequest)
		}},
		{name: "revokes token", run: func(t *testing.T) {
			token := mustCreateServiceToken(t, ctx, repo, user.ID, "revoke-me")
			require.NoError(t, svc.RevokePersonalAccessToken(ctx, user.ID, token.ID))
			stored, getErr := repo.GetTokenByID(ctx, repository.GetTokenByIDParams{ID: token.ID, UserID: user.ID})
			require.NoError(t, getErr)
			assert.True(t, stored.Revoked)
		}},
		{name: "returns not found when revoking missing token", run: func(t *testing.T) {
			revokeErr := svc.RevokePersonalAccessToken(ctx, user.ID, uuid.New())
			assertServiceError(t, revokeErr, http.StatusNotFound)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestValidatePAT(t *testing.T) {
	ctx := context.Background()
	repo := repository.New(testutil.MustOpenMigratedPostgres(t))
	user := seedServiceUser(t, ctx, repo)
	svc := service.New(repo, nil)

	valid := mustCreateServiceToken(t, ctx, repo, user.ID, "valid")
	revoked := mustCreateServiceToken(t, ctx, repo, user.ID, "revoked")
	_, err := repo.RevokeToken(ctx, repository.RevokeTokenParams{ID: revoked.ID, UserID: user.ID})
	require.NoError(t, err)
	expired := mustCreateServiceTokenWithParams(t, ctx, repo, repository.CreateTokenParams{UserID: user.ID, Name: "expired", Scopes: []string{"models:list"}, TokenType: auth.TokenTypePAT, Jti: uuid.NewString(), ExpiresAt: time.Now().Add(-time.Hour)})
	wrongType := mustCreateServiceTokenWithParams(t, ctx, repo, repository.CreateTokenParams{UserID: user.ID, Name: "session", Scopes: []string{"models:list"}, TokenType: auth.TokenTypeSession, Jti: uuid.NewString(), ExpiresAt: time.Now().Add(time.Hour)})

	tests := []struct {
		name     string
		claims   *auth.Claims
		wantCode int
		check    func(*testing.T)
	}{
		{name: "rejects missing claims", wantCode: http.StatusUnauthorized},
		{name: "rejects missing jti", claims: &auth.Claims{}, wantCode: http.StatusUnauthorized},
		{name: "rejects missing token record", claims: &auth.Claims{RegisteredClaims: jwtClaims(user.ID.String(), uuid.NewString())}, wantCode: http.StatusUnauthorized},
		{name: "rejects token type mismatch", claims: &auth.Claims{RegisteredClaims: jwtClaims(user.ID.String(), wrongType.Jti)}, wantCode: http.StatusUnauthorized},
		{name: "rejects revoked token", claims: &auth.Claims{RegisteredClaims: jwtClaims(user.ID.String(), revoked.Jti)}, wantCode: http.StatusUnauthorized},
		{name: "rejects subject mismatch", claims: &auth.Claims{RegisteredClaims: jwtClaims(uuid.NewString(), valid.Jti)}, wantCode: http.StatusUnauthorized},
		{name: "rejects expired token", claims: &auth.Claims{RegisteredClaims: jwtClaims(user.ID.String(), expired.Jti)}, wantCode: http.StatusUnauthorized},
		{name: "marks valid token used", claims: &auth.Claims{RegisteredClaims: jwtClaims(user.ID.String(), valid.Jti)}, check: func(t *testing.T) {
			stored, getErr := repo.GetTokenByJTI(ctx, valid.Jti)
			require.NoError(t, getErr)
			assert.NotNil(t, stored.LastUsedAt)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.ValidatePAT(ctx, tc.claims)
			if tc.wantCode != 0 {
				assertServiceError(t, err, tc.wantCode)
				return
			}
			require.NoError(t, err)
			if tc.check != nil {
				tc.check(t)
			}
		})
	}
}

func mustCreateServiceToken(t *testing.T, ctx context.Context, repo *repository.Queries, userID uuid.UUID, name string) repository.Token {
	t.Helper()
	return mustCreateServiceTokenWithParams(t, ctx, repo, repository.CreateTokenParams{UserID: userID, Name: name, Scopes: []string{"models:list"}, TokenType: auth.TokenTypePAT, Jti: uuid.NewString(), ExpiresAt: time.Now().Add(time.Hour)})
}

func mustCreateServiceTokenWithParams(t *testing.T, ctx context.Context, repo *repository.Queries, params repository.CreateTokenParams) repository.Token {
	t.Helper()
	token, err := repo.CreateToken(ctx, params)
	require.NoError(t, err)
	return token
}

func jwtClaims(subject, jti string) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{Subject: subject, ID: jti}
}

func assertServiceError(t *testing.T, err error, code int) {
	t.Helper()
	var serviceErr *service.Error
	require.Error(t, err)
	require.True(t, errors.As(err, &serviceErr), "expected *service.Error, got %T (%v)", err, err)
	assert.Equal(t, code, serviceErr.Code)
}
