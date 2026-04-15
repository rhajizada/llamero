package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestTokenQueries(t *testing.T) {
	ctx := context.Background()
	pool := testutil.MustOpenMigratedPostgres(t)
	queries := repository.New(pool)

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "creates and fetches token by id and jti",
			run: func(t *testing.T) {
				user := seedRepositoryUser(ctx, t, queries)
				expiresAt := time.Now().Add(2 * time.Hour)
				created, err := queries.CreateToken(ctx, repository.CreateTokenParams{
					UserID:    user.ID,
					Name:      "cli",
					Scopes:    []string{"models:list", "tokens:write"},
					TokenType: auth.TokenTypePAT,
					Jti:       uuid.NewString(),
					ExpiresAt: expiresAt,
				})
				require.NoError(t, err)

				byID, err := queries.GetTokenByID(ctx, repository.GetTokenByIDParams{
					ID:     created.ID,
					UserID: user.ID,
				})
				require.NoError(t, err)

				byJTI, err := queries.GetTokenByJTI(ctx, created.Jti)
				require.NoError(t, err)

				assert.Equal(t, created.ID, byID.ID)
				assert.Equal(t, created.ID, byJTI.ID)
				assert.Equal(t, expiresAt.Unix(), created.ExpiresAt.Unix())
				assert.False(t, created.Revoked)
			},
		},
		{
			name: "lists only active tokens in descending order",
			run: func(t *testing.T) {
				user := seedRepositoryUser(ctx, t, queries)
				older := mustCreateRepositoryToken(ctx, t, queries, user.ID, "older")
				time.Sleep(10 * time.Millisecond)
				newer := mustCreateRepositoryToken(ctx, t, queries, user.ID, "newer")
				revoked := mustCreateRepositoryToken(ctx, t, queries, user.ID, "revoked")

				_, err := queries.RevokeToken(ctx, repository.RevokeTokenParams{
					ID:     revoked.ID,
					UserID: user.ID,
				})
				require.NoError(t, err)

				listed, err := queries.ListTokensByUser(ctx, user.ID)
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(listed), 2)
				assert.Equal(t, newer.ID, listed[0].ID)
				assert.Contains(t, []uuid.UUID{listed[0].ID, listed[1].ID}, older.ID)

				for _, token := range listed {
					assert.False(t, token.Revoked)
					assert.NotEqual(t, revoked.ID, token.ID)
				}
			},
		},
		{
			name: "marks token used",
			run: func(t *testing.T) {
				user := seedRepositoryUser(ctx, t, queries)
				token := mustCreateRepositoryToken(ctx, t, queries, user.ID, "used")
				require.Nil(t, token.LastUsedAt)

				require.NoError(t, queries.MarkTokenUsed(ctx, token.ID))

				stored, err := queries.GetTokenByID(ctx, repository.GetTokenByIDParams{
					ID:     token.ID,
					UserID: user.ID,
				})
				require.NoError(t, err)
				assert.NotNil(t, stored.LastUsedAt)
			},
		},
		{
			name: "revokes token",
			run: func(t *testing.T) {
				user := seedRepositoryUser(ctx, t, queries)
				token := mustCreateRepositoryToken(ctx, t, queries, user.ID, "revoke")

				revoked, err := queries.RevokeToken(ctx, repository.RevokeTokenParams{
					ID:     token.ID,
					UserID: user.ID,
				})
				require.NoError(t, err)
				assert.True(t, revoked.Revoked)
			},
		},
		{
			name: "returns no rows for missing token lookups",
			run: func(t *testing.T) {
				user := seedRepositoryUser(ctx, t, queries)

				_, err := queries.GetTokenByID(ctx, repository.GetTokenByIDParams{
					ID:     uuid.New(),
					UserID: user.ID,
				})
				require.ErrorIs(t, err, pgx.ErrNoRows)

				_, err = queries.GetTokenByJTI(ctx, uuid.NewString())
				require.ErrorIs(t, err, pgx.ErrNoRows)

				_, err = queries.RevokeToken(ctx, repository.RevokeTokenParams{
					ID:     uuid.New(),
					UserID: user.ID,
				})
				require.ErrorIs(t, err, pgx.ErrNoRows)
			},
		},
		{
			name: "supports queries within transaction",
			run: func(t *testing.T) {
				user := seedRepositoryUser(ctx, t, queries)
				tx, err := pool.Begin(ctx)
				require.NoError(t, err)
				defer tx.Rollback(ctx)

				inTx := queries.WithTx(tx)
				token, err := inTx.CreateToken(ctx, repository.CreateTokenParams{
					UserID:    user.ID,
					Name:      "tx-token",
					Scopes:    []string{"models:list"},
					TokenType: auth.TokenTypePAT,
					Jti:       uuid.NewString(),
					ExpiresAt: time.Now().Add(time.Hour),
				})
				require.NoError(t, err)

				stored, err := inTx.GetTokenByJTI(ctx, token.Jti)
				require.NoError(t, err)
				assert.Equal(t, token.ID, stored.ID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func mustCreateRepositoryToken(
	ctx context.Context,
	t *testing.T,
	queries *repository.Queries,
	userID uuid.UUID,
	name string,
) repository.Token {
	t.Helper()

	token, err := queries.CreateToken(ctx, repository.CreateTokenParams{
		UserID:    userID,
		Name:      name,
		Scopes:    []string{"models:list"},
		TokenType: auth.TokenTypePAT,
		Jti:       uuid.NewString(),
		ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	return token
}
