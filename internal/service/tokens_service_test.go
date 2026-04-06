package service_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/service"
)

type fakeRepo struct {
	createTokenFn   func(context.Context, repository.CreateTokenParams) (repository.Token, error)
	getTokenByIDFn  func(context.Context, repository.GetTokenByIDParams) (repository.Token, error)
	getTokenByJTIFn func(context.Context, string) (repository.Token, error)
	listTokensFn    func(context.Context, uuid.UUID) ([]repository.Token, error)
	markTokenUsedFn func(context.Context, uuid.UUID) error
	revokeTokenFn   func(context.Context, repository.RevokeTokenParams) (repository.Token, error)
	upsertUserFn    func(context.Context, repository.UpsertUserParams) (repository.User, error)
	getUserByIDFn   func(context.Context, uuid.UUID) (repository.User, error)
}

func (f *fakeRepo) CreateToken(ctx context.Context, arg repository.CreateTokenParams) (repository.Token, error) {
	return f.createTokenFn(ctx, arg)
}
func (f *fakeRepo) GetTokenByID(ctx context.Context, arg repository.GetTokenByIDParams) (repository.Token, error) {
	return f.getTokenByIDFn(ctx, arg)
}
func (f *fakeRepo) GetTokenByJTI(ctx context.Context, jti string) (repository.Token, error) {
	return f.getTokenByJTIFn(ctx, jti)
}
func (f *fakeRepo) ListTokensByUser(ctx context.Context, userID uuid.UUID) ([]repository.Token, error) {
	return f.listTokensFn(ctx, userID)
}
func (f *fakeRepo) MarkTokenUsed(ctx context.Context, id uuid.UUID) error {
	return f.markTokenUsedFn(ctx, id)
}
func (f *fakeRepo) RevokeToken(ctx context.Context, arg repository.RevokeTokenParams) (repository.Token, error) {
	return f.revokeTokenFn(ctx, arg)
}
func (f *fakeRepo) UpsertUser(ctx context.Context, arg repository.UpsertUserParams) (repository.User, error) {
	return f.upsertUserFn(ctx, arg)
}
func (f *fakeRepo) GetUserByID(ctx context.Context, id uuid.UUID) (repository.User, error) {
	return f.getUserByIDFn(ctx, id)
}

func TestCreatePersonalAccessToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tokenID := uuid.New()
	expiresAt := time.Now().Add(time.Hour)
	called := false
	svc := service.New(&fakeRepo{
		createTokenFn: func(_ context.Context, arg repository.CreateTokenParams) (repository.Token, error) {
			called = true
			if arg.UserID != userID || arg.Name != "cli" || arg.TokenType != auth.TokenTypePAT || arg.Jti != "jti-1" {
				t.Fatalf("unexpected create token params: %#v", arg)
			}
			return repository.Token{
				ID:        tokenID,
				UserID:    userID,
				Name:      arg.Name,
				Scopes:    arg.Scopes,
				TokenType: arg.TokenType,
				Jti:       arg.Jti,
				ExpiresAt: expiresAt,
			}, nil
		},
	}, nil)

	token, err := svc.CreatePersonalAccessToken(context.Background(), service.CreateTokenParams{
		UserID:    userID,
		Name:      " cli ",
		Scopes:    []string{"models:list"},
		JTI:       "jti-1",
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("CreatePersonalAccessToken returned error: %v", err)
	}
	if !called || token.ID != tokenID || token.Name != "cli" {
		t.Fatalf("unexpected token: %#v called=%v", token, called)
	}
}

func TestCreatePersonalAccessTokenValidationAndErrors(t *testing.T) {
	t.Parallel()

	svc := service.New(&fakeRepo{
		createTokenFn: func(context.Context, repository.CreateTokenParams) (repository.Token, error) {
			return repository.Token{}, errors.New("db down")
		},
	}, nil)

	for _, tc := range []struct {
		name   string
		params service.CreateTokenParams
	}{
		{name: "missing everything", params: service.CreateTokenParams{}},
		{name: "missing name", params: service.CreateTokenParams{UserID: uuid.New(), Name: "", Scopes: []string{"models:list"}, JTI: "jti", ExpiresAt: time.Now().Add(time.Hour)}},
		{name: "missing scopes", params: service.CreateTokenParams{UserID: uuid.New(), Name: "cli", Scopes: nil, JTI: "jti", ExpiresAt: time.Now().Add(time.Hour)}},
		{name: "missing jti", params: service.CreateTokenParams{UserID: uuid.New(), Name: "cli", Scopes: []string{"models:list"}, JTI: "", ExpiresAt: time.Now().Add(time.Hour)}},
		{name: "expired token", params: service.CreateTokenParams{UserID: uuid.New(), Name: "cli", Scopes: []string{"models:list"}, JTI: "jti", ExpiresAt: time.Now().Add(-time.Hour)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := svc.CreatePersonalAccessToken(context.Background(), tc.params); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	_, err := svc.CreatePersonalAccessToken(context.Background(), service.CreateTokenParams{
		UserID:    uuid.New(),
		Name:      "cli",
		Scopes:    []string{"models:list"},
		JTI:       "jti",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	assertServiceError(t, err, http.StatusInternalServerError)
}

func TestListGetAndRevokePersonalAccessTokens(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tokenID := uuid.New()
	now := time.Now().UTC()
	repo := &fakeRepo{
		listTokensFn: func(context.Context, uuid.UUID) ([]repository.Token, error) {
			return []repository.Token{{
				ID:        tokenID,
				UserID:    userID,
				Name:      "cli",
				Scopes:    []string{"models:list"},
				CreatedAt: now,
				UpdatedAt: now,
			}}, nil
		},
		getTokenByIDFn: func(_ context.Context, arg repository.GetTokenByIDParams) (repository.Token, error) {
			if arg.ID != tokenID || arg.UserID != userID {
				t.Fatalf("unexpected get token params: %#v", arg)
			}
			return repository.Token{
				ID:        tokenID,
				UserID:    userID,
				Name:      "cli",
				Scopes:    []string{"models:list"},
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
		revokeTokenFn: func(_ context.Context, arg repository.RevokeTokenParams) (repository.Token, error) {
			if arg.ID != tokenID || arg.UserID != userID {
				t.Fatalf("unexpected revoke token params: %#v", arg)
			}
			return repository.Token{ID: tokenID, UserID: userID}, nil
		},
	}
	svc := service.New(repo, nil)

	tokens, err := svc.ListPersonalAccessTokens(context.Background(), userID)
	if err != nil || len(tokens) != 1 || tokens[0].ID != tokenID {
		t.Fatalf("unexpected listed tokens: %#v err=%v", tokens, err)
	}

	token, err := svc.GetPersonalAccessToken(context.Background(), userID, tokenID)
	if err != nil || token.ID != tokenID {
		t.Fatalf("unexpected token: %#v err=%v", token, err)
	}

	if revokeErr := svc.RevokePersonalAccessToken(context.Background(), userID, tokenID); revokeErr != nil {
		t.Fatalf("unexpected revoke error: %v", revokeErr)
	}
}

func TestTokenLookupAndRevokeErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tokenID := uuid.New()
	svc := service.New(&fakeRepo{
		listTokensFn: func(context.Context, uuid.UUID) ([]repository.Token, error) { return nil, errors.New("db down") },
		getTokenByIDFn: func(context.Context, repository.GetTokenByIDParams) (repository.Token, error) {
			return repository.Token{}, pgx.ErrNoRows
		},
		revokeTokenFn: func(context.Context, repository.RevokeTokenParams) (repository.Token, error) {
			return repository.Token{}, pgx.ErrNoRows
		},
	}, nil)

	_, err := svc.ListPersonalAccessTokens(context.Background(), uuid.Nil)
	assertServiceError(t, err, http.StatusBadRequest)
	_, err = svc.ListPersonalAccessTokens(context.Background(), userID)
	assertServiceError(t, err, http.StatusInternalServerError)

	_, err = svc.GetPersonalAccessToken(context.Background(), uuid.Nil, tokenID)
	assertServiceError(t, err, http.StatusBadRequest)
	_, err = svc.GetPersonalAccessToken(context.Background(), userID, tokenID)
	assertServiceError(t, err, http.StatusNotFound)

	err = svc.RevokePersonalAccessToken(context.Background(), uuid.Nil, tokenID)
	assertServiceError(t, err, http.StatusBadRequest)
	err = svc.RevokePersonalAccessToken(context.Background(), userID, tokenID)
	assertServiceError(t, err, http.StatusNotFound)
}

func TestValidatePAT(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tokenID := uuid.New()
	repo := &fakeRepo{
		getTokenByJTIFn: func(_ context.Context, jti string) (repository.Token, error) {
			if jti != "jti-1" {
				t.Fatalf("unexpected jti: %s", jti)
			}
			return repository.Token{
				ID:        tokenID,
				UserID:    userID,
				TokenType: auth.TokenTypePAT,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
		markTokenUsedFn: func(_ context.Context, id uuid.UUID) error {
			if id != tokenID {
				t.Fatalf("unexpected mark-token-used id: %s", id)
			}
			return nil
		},
	}
	svc := service.New(repo, nil)

	err := svc.ValidatePAT(context.Background(), &auth.Claims{RegisteredClaims: jwtClaims(userID.String(), "jti-1")})
	if err != nil {
		t.Fatalf("unexpected ValidatePAT error: %v", err)
	}
}

func TestValidatePATErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	baseClaims := &auth.Claims{RegisteredClaims: jwtClaims(userID.String(), "jti-1")}

	svc := service.New(&fakeRepo{
		getTokenByJTIFn: func(context.Context, string) (repository.Token, error) {
			return repository.Token{}, pgx.ErrNoRows
		},
		markTokenUsedFn: func(context.Context, uuid.UUID) error { return errors.New("write failed") },
	}, nil)

	assertServiceError(t, svc.ValidatePAT(context.Background(), nil), http.StatusUnauthorized)
	assertServiceError(t, svc.ValidatePAT(context.Background(), &auth.Claims{}), http.StatusUnauthorized)
	assertServiceError(t, svc.ValidatePAT(context.Background(), baseClaims), http.StatusUnauthorized)

	svc = service.New(&fakeRepo{
		getTokenByJTIFn: func(context.Context, string) (repository.Token, error) {
			return repository.Token{
				UserID:    userID,
				TokenType: auth.TokenTypeSession,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
		markTokenUsedFn: func(context.Context, uuid.UUID) error { return nil },
	}, nil)
	assertServiceError(t, svc.ValidatePAT(context.Background(), baseClaims), http.StatusUnauthorized)

	svc = service.New(&fakeRepo{
		getTokenByJTIFn: func(context.Context, string) (repository.Token, error) {
			return repository.Token{
				UserID:    userID,
				TokenType: auth.TokenTypePAT,
				Revoked:   true,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
		markTokenUsedFn: func(context.Context, uuid.UUID) error { return nil },
	}, nil)
	assertServiceError(t, svc.ValidatePAT(context.Background(), baseClaims), http.StatusUnauthorized)

	svc = service.New(&fakeRepo{
		getTokenByJTIFn: func(context.Context, string) (repository.Token, error) {
			return repository.Token{
				UserID:    uuid.New(),
				TokenType: auth.TokenTypePAT,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
		markTokenUsedFn: func(context.Context, uuid.UUID) error { return nil },
	}, nil)
	assertServiceError(t, svc.ValidatePAT(context.Background(), baseClaims), http.StatusUnauthorized)

	svc = service.New(&fakeRepo{
		getTokenByJTIFn: func(context.Context, string) (repository.Token, error) {
			return repository.Token{
				UserID:    userID,
				TokenType: auth.TokenTypePAT,
				ExpiresAt: time.Now().Add(-time.Hour),
			}, nil
		},
		markTokenUsedFn: func(context.Context, uuid.UUID) error { return nil },
	}, nil)
	assertServiceError(t, svc.ValidatePAT(context.Background(), baseClaims), http.StatusUnauthorized)

	svc = service.New(&fakeRepo{
		getTokenByJTIFn: func(context.Context, string) (repository.Token, error) {
			return repository.Token{
				ID:        uuid.New(),
				UserID:    userID,
				TokenType: auth.TokenTypePAT,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
		markTokenUsedFn: func(context.Context, uuid.UUID) error { return errors.New("write failed") },
	}, nil)
	assertServiceError(t, svc.ValidatePAT(context.Background(), baseClaims), http.StatusInternalServerError)
}

func TestUpsertUserAndGetUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	now := time.Now().UTC()
	repo := &fakeRepo{
		upsertUserFn: func(_ context.Context, arg repository.UpsertUserParams) (repository.User, error) {
			return repository.User{ID: userID, Email: arg.Email, Role: arg.Role}, nil
		},
		getUserByIDFn: func(_ context.Context, id uuid.UUID) (repository.User, error) {
			return repository.User{
				ID:        id,
				Email:     "user@example.com",
				Role:      "admin",
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}
	svc := service.New(repo, nil)

	user, err := svc.UpsertUser(
		context.Background(),
		repository.UpsertUserParams{Email: "user@example.com", Role: "admin"},
	)
	if err != nil || user.ID != userID {
		t.Fatalf("unexpected upsert result: %#v err=%v", user, err)
	}

	modelUser, err := svc.GetUser(context.Background(), userID)
	if err != nil || modelUser.ID != userID || modelUser.Email != "user@example.com" {
		t.Fatalf("unexpected get user result: %#v err=%v", modelUser, err)
	}
}

func TestUserServiceErrors(t *testing.T) {
	t.Parallel()

	svc := service.New(&fakeRepo{
		upsertUserFn: func(context.Context, repository.UpsertUserParams) (repository.User, error) {
			return repository.User{}, errors.New("db down")
		},
		getUserByIDFn: func(context.Context, uuid.UUID) (repository.User, error) {
			return repository.User{}, pgx.ErrNoRows
		},
	}, nil)

	_, err := svc.UpsertUser(context.Background(), repository.UpsertUserParams{})
	assertServiceError(t, err, http.StatusInternalServerError)
	_, err = svc.GetUser(context.Background(), uuid.New())
	assertServiceError(t, err, http.StatusNotFound)
}

func jwtClaims(subject, jti string) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{Subject: subject, ID: jti}
}

func assertServiceError(t *testing.T, err error, code int) {
	t.Helper()
	var serviceErr *service.Error
	if !errors.As(err, &serviceErr) {
		t.Fatalf("expected *service.Error, got %T (%v)", err, err)
	}
	if serviceErr.Code != code {
		t.Fatalf("unexpected service error code: got %d want %d", serviceErr.Code, code)
	}
}
