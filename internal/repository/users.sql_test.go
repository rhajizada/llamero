package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestUserQueries(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	updatedAt := now.Add(10 * time.Minute)
	displayName := "Initial User"
	updatedName := "Updated User"

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "upserts new and existing user",
			run: func(t *testing.T) {
				queries := repository.New(testutil.MustOpenMigratedPostgres(t))
				provider := "oidc-" + uuid.NewString()
				sub := "sub-" + uuid.NewString()
				email := uuid.NewString() + "@example.com"

				created, err := queries.UpsertUser(ctx, repository.UpsertUserParams{
					Sub:         sub,
					Provider:    provider,
					Email:       email,
					DisplayName: &displayName,
					Role:        "member",
					Scopes:      []string{"models:list"},
					Groups:      []string{"users"},
					LastLoginAt: &now,
				})
				require.NoError(t, err)

				updated, err := queries.UpsertUser(ctx, repository.UpsertUserParams{
					Sub:         sub,
					Provider:    provider,
					Email:       email,
					DisplayName: &updatedName,
					Role:        "admin",
					Scopes:      []string{"models:list", "tokens:write"},
					Groups:      []string{"admins"},
					LastLoginAt: &updatedAt,
				})
				require.NoError(t, err)

				assert.Equal(t, created.ID, updated.ID)
				assert.Equal(t, "admin", updated.Role)
				require.NotNil(t, updated.DisplayName)
				assert.Equal(t, updatedName, *updated.DisplayName)
				require.NotNil(t, updated.LastLoginAt)
				assert.Equal(t, updatedAt.Unix(), updated.LastLoginAt.Unix())
			},
		},
		{
			name: "gets user by id and provider sub",
			run: func(t *testing.T) {
				queries := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedRepositoryUser(ctx, t, queries)

				byID, err := queries.GetUserByID(ctx, user.ID)
				require.NoError(t, err)

				byProviderSub, err := queries.GetUserByProviderSub(ctx, repository.GetUserByProviderSubParams{
					Provider: user.Provider,
					Sub:      user.Sub,
				})
				require.NoError(t, err)

				assert.Equal(t, user.ID, byID.ID)
				assert.Equal(t, user.ID, byProviderSub.ID)
			},
		},
		{
			name: "lists users with pagination",
			run: func(t *testing.T) {
				queries := repository.New(testutil.MustOpenMigratedPostgres(t))
				userA := seedRepositoryUser(ctx, t, queries)
				time.Sleep(10 * time.Millisecond)
				userB := seedRepositoryUser(ctx, t, queries)

				listed, err := queries.ListUsers(ctx, repository.ListUsersParams{Limit: 1, Offset: 0})
				require.NoError(t, err)
				require.Len(t, listed, 1)
				assert.Equal(t, userB.ID, listed[0].ID)

				listed, err = queries.ListUsers(ctx, repository.ListUsersParams{Limit: 2, Offset: 1})
				require.NoError(t, err)
				require.NotEmpty(t, listed)
				assert.Equal(t, userA.ID, listed[0].ID)
			},
		},
		{
			name: "deletes user",
			run: func(t *testing.T) {
				queries := repository.New(testutil.MustOpenMigratedPostgres(t))
				user := seedRepositoryUser(ctx, t, queries)

				require.NoError(t, queries.DeleteUser(ctx, user.ID))

				_, err := queries.GetUserByID(ctx, user.ID)
				require.ErrorIs(t, err, pgx.ErrNoRows)
			},
		},
		{
			name: "returns no rows for missing lookups",
			run: func(t *testing.T) {
				queries := repository.New(testutil.MustOpenMigratedPostgres(t))

				_, err := queries.GetUserByID(ctx, uuid.New())
				require.ErrorIs(t, err, pgx.ErrNoRows)

				_, err = queries.GetUserByProviderSub(ctx, repository.GetUserByProviderSubParams{
					Provider: "missing",
					Sub:      "missing",
				})
				require.ErrorIs(t, err, pgx.ErrNoRows)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func seedRepositoryUser(
	ctx context.Context,
	t *testing.T,
	queries *repository.Queries,
) repository.User {
	t.Helper()

	displayName := "Repo User"
	lastLoginAt := time.Now().UTC().Truncate(time.Second)
	user, err := queries.UpsertUser(ctx, repository.UpsertUserParams{
		Sub:         "sub-" + uuid.NewString(),
		Provider:    "oidc-" + uuid.NewString(),
		Email:       uuid.NewString() + "@example.com",
		DisplayName: &displayName,
		Role:        "member",
		Scopes:      []string{"models:list"},
		Groups:      []string{"users"},
		LastLoginAt: &lastLoginAt,
	})
	require.NoError(t, err)

	return user
}
