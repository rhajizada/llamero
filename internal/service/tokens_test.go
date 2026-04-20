package service_test

import (
	"context"
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

	tests := []struct {
		name     string
		params   service.CreateTokenParams
		wantCode int
		wantType string
	}{
		{
			name: "creates token with default type",
			params: service.CreateTokenParams{
				Name:      " cli ",
				Scopes:    []string{"models:list"},
				JTI:       uuid.NewString(),
				ExpiresAt: time.Now().Add(time.Hour),
			},
			wantType: auth.TokenTypePAT,
		},
		{
			name: "creates token with explicit type",
			params: service.CreateTokenParams{
				Name:      "automation",
				Scopes:    []string{"models:list", "tokens:read"},
				TokenType: auth.TokenTypePAT,
				JTI:       uuid.NewString(),
				ExpiresAt: time.Now().Add(2 * time.Hour),
			},
			wantType: auth.TokenTypePAT,
		},
		{
			name: "rejects missing name",
			params: service.CreateTokenParams{
				Scopes:    []string{"models:list"},
				JTI:       uuid.NewString(),
				ExpiresAt: time.Now().Add(time.Hour),
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "rejects missing user id",
			params: service.CreateTokenParams{
				Name:      "cli",
				Scopes:    []string{"models:list"},
				JTI:       uuid.NewString(),
				ExpiresAt: time.Now().Add(time.Hour),
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "rejects missing scopes",
			params: service.CreateTokenParams{
				Name:      "cli",
				JTI:       uuid.NewString(),
				ExpiresAt: time.Now().Add(time.Hour),
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "rejects missing jti",
			params: service.CreateTokenParams{
				Name:      "cli",
				Scopes:    []string{"models:list"},
				ExpiresAt: time.Now().Add(time.Hour),
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "rejects expired token",
			params: service.CreateTokenParams{
				Name:      "cli",
				Scopes:    []string{"models:list"},
				JTI:       uuid.NewString(),
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "surfaces repository failure for missing user",
			params: service.CreateTokenParams{
				UserID:    uuid.New(),
				Name:      "cli",
				Scopes:    []string{"models:list"},
				JTI:       uuid.NewString(),
				ExpiresAt: time.Now().Add(time.Hour),
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := repository.New(testutil.MustOpenMigratedPostgres(t))
			user := seedServiceUser(ctx, t, repo)
			svc := service.New(repo, nil)

			params := tc.params
			if params.UserID == uuid.Nil &&
				tc.name != "rejects missing user id" &&
				tc.name != "surfaces repository failure for missing user" {
				params.UserID = user.ID
			}

			token, err := svc.CreatePersonalAccessToken(ctx, params)
			if tc.wantCode != 0 {
				assertServiceError(t, err, tc.wantCode)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(params.Name), token.Name)
			assert.Equal(t, tc.wantType, token.TokenType)
			assert.Equal(t, params.UserID, token.UserID)
			assert.Equal(t, params.Scopes, token.Scopes)

			stored, getErr := repo.GetTokenByJTI(ctx, params.JTI)
			require.NoError(t, getErr)
			assert.Equal(t, token.ID, stored.ID)
		})
	}
}

func TestPersonalAccessTokenLifecycle(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "lists active tokens in descending order",
			run: func(t *testing.T) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				svc := service.New(repo, nil)
				older := mustCreateServiceToken(ctx, t, repo, user.ID, "older")
				time.Sleep(10 * time.Millisecond)
				newer := mustCreateServiceToken(ctx, t, repo, user.ID, "newer")
				revoked := mustCreateServiceToken(ctx, t, repo, user.ID, "revoked")
				_, err := repo.RevokeToken(ctx, repository.RevokeTokenParams{ID: revoked.ID, UserID: user.ID})
				require.NoError(t, err)

				tokens, listErr := svc.ListPersonalAccessTokens(ctx, user.ID)
				require.NoError(t, listErr)
				require.Len(t, tokens, 2)
				assert.Equal(t, []uuid.UUID{newer.ID, older.ID}, []uuid.UUID{tokens[0].ID, tokens[1].ID})
			},
		},
		{
			name: "gets token by id",
			run: func(t *testing.T) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				svc := service.New(repo, nil)
				newer := mustCreateServiceToken(ctx, t, repo, user.ID, "newer")

				token, getErr := svc.GetPersonalAccessToken(ctx, user.ID, newer.ID)
				require.NoError(t, getErr)
				assert.Equal(t, newer.ID, token.ID)
			},
		},
		{
			name: "returns not found for missing token",
			run: func(t *testing.T) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				svc := service.New(repo, nil)

				_, getErr := svc.GetPersonalAccessToken(ctx, user.ID, uuid.New())
				assertServiceError(t, getErr, http.StatusNotFound)
			},
		},
		{
			name: "rejects missing ids",
			run: func(t *testing.T) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				svc := service.New(repo, nil)
				newerID := uuid.New()

				_, listErr := svc.ListPersonalAccessTokens(ctx, uuid.Nil)
				assertServiceError(t, listErr, http.StatusBadRequest)

				_, getErr := svc.GetPersonalAccessToken(ctx, uuid.Nil, newerID)
				assertServiceError(t, getErr, http.StatusBadRequest)

				revokeErr := svc.RevokePersonalAccessToken(ctx, uuid.Nil, newerID)
				assertServiceError(t, revokeErr, http.StatusBadRequest)
			},
		},
		{
			name: "revokes token",
			run: func(t *testing.T) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				svc := service.New(repo, nil)
				token := mustCreateServiceToken(ctx, t, repo, user.ID, "revoke-me")
				require.NoError(t, svc.RevokePersonalAccessToken(ctx, user.ID, token.ID))

				stored, getErr := repo.GetTokenByID(ctx, repository.GetTokenByIDParams{
					ID:     token.ID,
					UserID: user.ID,
				})
				require.NoError(t, getErr)
				assert.True(t, stored.Revoked)
			},
		},
		{
			name: "returns not found when revoking missing token",
			run: func(t *testing.T) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				svc := service.New(repo, nil)

				revokeErr := svc.RevokePersonalAccessToken(ctx, user.ID, uuid.New())
				assertServiceError(t, revokeErr, http.StatusNotFound)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestValidatePAT(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		prepare  func(*testing.T) (*service.Service, *repository.Queries, repository.User, *auth.Claims)
		wantCode int
		check    func(*testing.T, *repository.Queries, repository.User)
	}{
		{
			name: "rejects missing claims",
			prepare: func(t *testing.T) (
				*service.Service,
				*repository.Queries,
				repository.User,
				*auth.Claims,
			) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				return service.New(repo, nil), repo, user, nil
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "rejects missing jti",
			prepare: func(t *testing.T) (
				*service.Service,
				*repository.Queries,
				repository.User,
				*auth.Claims,
			) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				return service.New(repo, nil), repo, user, &auth.Claims{}
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "rejects missing token record",
			prepare: func(t *testing.T) (*service.Service, *repository.Queries, repository.User, *auth.Claims) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				return service.New(repo, nil), repo, user, &auth.Claims{
					RegisteredClaims: jwtClaims(user.ID.String(), uuid.NewString()),
				}
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "rejects token type mismatch",
			prepare: func(t *testing.T) (*service.Service, *repository.Queries, repository.User, *auth.Claims) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				wrongType := mustCreateServiceTokenWithParams(
					ctx,
					t,
					repo,
					repository.CreateTokenParams{
						UserID:    user.ID,
						Name:      "session",
						Scopes:    []string{"models:list"},
						TokenType: auth.TokenTypeSession,
						Jti:       uuid.NewString(),
						ExpiresAt: time.Now().Add(time.Hour),
					},
				)
				return service.New(repo, nil), repo, user, &auth.Claims{
					RegisteredClaims: jwtClaims(user.ID.String(), wrongType.Jti),
				}
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "rejects revoked token",
			prepare: func(t *testing.T) (*service.Service, *repository.Queries, repository.User, *auth.Claims) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				revoked := mustCreateServiceToken(ctx, t, repo, user.ID, "revoked")
				_, err := repo.RevokeToken(ctx, repository.RevokeTokenParams{ID: revoked.ID, UserID: user.ID})
				require.NoError(t, err)
				return service.New(repo, nil), repo, user, &auth.Claims{
					RegisteredClaims: jwtClaims(user.ID.String(), revoked.Jti),
				}
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "rejects subject mismatch",
			prepare: func(t *testing.T) (*service.Service, *repository.Queries, repository.User, *auth.Claims) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				valid := mustCreateServiceToken(ctx, t, repo, user.ID, "valid")
				return service.New(repo, nil), repo, user, &auth.Claims{
					RegisteredClaims: jwtClaims(uuid.NewString(), valid.Jti),
				}
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "rejects expired token",
			prepare: func(t *testing.T) (*service.Service, *repository.Queries, repository.User, *auth.Claims) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				expired := mustCreateServiceTokenWithParams(
					ctx,
					t,
					repo,
					repository.CreateTokenParams{
						UserID:    user.ID,
						Name:      "expired",
						Scopes:    []string{"models:list"},
						TokenType: auth.TokenTypePAT,
						Jti:       uuid.NewString(),
						ExpiresAt: time.Now().Add(-time.Hour),
					},
				)
				return service.New(repo, nil), repo, user, &auth.Claims{
					RegisteredClaims: jwtClaims(user.ID.String(), expired.Jti),
				}
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "marks valid token used",
			prepare: func(t *testing.T) (*service.Service, *repository.Queries, repository.User, *auth.Claims) {
				repo := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedServiceUser(ctx, t, repo)
				valid := mustCreateServiceToken(ctx, t, repo, user.ID, "valid")
				return service.New(repo, nil), repo, user, &auth.Claims{
					RegisteredClaims: jwtClaims(user.ID.String(), valid.Jti),
				}
			},
			check: func(t *testing.T, repo *repository.Queries, user repository.User) {
				stored, getErr := repo.ListTokensByUser(ctx, user.ID)
				require.NoError(t, getErr)
				require.NotEmpty(t, stored)
				assert.NotNil(t, stored[0].LastUsedAt)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, user, claims := tc.prepare(t)
			validateErr := svc.ValidatePAT(ctx, claims)
			if tc.wantCode != 0 {
				assertServiceError(t, validateErr, tc.wantCode)
				return
			}

			require.NoError(t, validateErr)
			if tc.check != nil {
				tc.check(t, repo, user)
			}
		})
	}
}

func mustCreateServiceToken(
	ctx context.Context,
	t *testing.T,
	repo *repository.Queries,
	userID uuid.UUID,
	name string,
) repository.Token {
	t.Helper()

	return mustCreateServiceTokenWithParams(ctx, t, repo, repository.CreateTokenParams{
		UserID:    userID,
		Name:      name,
		Scopes:    []string{"models:list"},
		TokenType: auth.TokenTypePAT,
		Jti:       uuid.NewString(),
		ExpiresAt: time.Now().Add(time.Hour),
	})
}

func mustCreateServiceTokenWithParams(
	ctx context.Context,
	t *testing.T,
	repo *repository.Queries,
	params repository.CreateTokenParams,
) repository.Token {
	t.Helper()

	token, err := repo.CreateToken(ctx, params)
	require.NoError(t, err)

	return token
}

func jwtClaims(subject, jti string) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{Subject: subject, ID: jti}
}
