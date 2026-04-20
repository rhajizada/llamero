package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/service"
	"github.com/rhajizada/llamero/internal/testutil"
	"github.com/rhajizada/llamero/internal/xslices"
)

type stubPATValidator struct {
	err    error
	called bool
}

func (s *stubPATValidator) ValidatePAT(_ context.Context, _ *auth.Claims) error {
	s.called = true
	return s.err
}

func TestAuthzHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "skips validation for session tokens",
			run: func(t *testing.T) {
				validator := &stubPATValidator{}
				err := middleware.ValidatePAT(context.Background(), &auth.Claims{Type: auth.TokenTypeSession}, validator)
				assert.NoError(t, err)
				assert.False(t, validator.called)
			},
		},
		{
			name: "fails when validator unavailable for PAT",
			run: func(t *testing.T) {
				err := middleware.ValidatePAT(context.Background(), &auth.Claims{Type: auth.TokenTypePAT}, nil)
				assert.ErrorIs(t, err, middleware.ErrPATValidationUnavailable)
			},
		},
		{
			name: "returns validator error for PAT",
			run: func(t *testing.T) {
				validator := &stubPATValidator{err: errors.New("revoked")}
				err := middleware.ValidatePAT(context.Background(), &auth.Claims{Type: auth.TokenTypePAT}, validator)
				assert.EqualError(t, err, "revoked")
				assert.True(t, validator.called)
			},
		},
		{
			name: "extracts bearer token",
			run: func(t *testing.T) {
				assert.Equal(t, "token-value", middleware.BearerToken("Bearer token-value"))
			},
		},
		{
			name: "rejects non bearer token",
			run: func(t *testing.T) {
				assert.Empty(t, middleware.BearerToken("Basic token-value"))
			},
		},
		{
			name: "rejects malformed bearer token",
			run: func(t *testing.T) {
				assert.Empty(t, middleware.BearerToken("Bearer"))
			},
		},
		{
			name: "deduplicates trimmed scopes",
			run: func(t *testing.T) {
				assert.Equal(t, []string{"read", "write"}, xslices.UniqueTrimmedStrings([]string{" read ", "write", "read", ""}))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

type stubPATValidatorHTTP struct {
	err error
}

func (s stubPATValidatorHTTP) ValidatePAT(_ context.Context, _ *auth.Claims) error {
	return s.err
}

func TestRequireWithSessionToken(t *testing.T) {
	t.Parallel()

	cfg := testutil.MustWriteEd25519JWTConfig(t)
	issuer, err := auth.NewTokenIssuer(cfg)
	require.NoError(t, err)
	verifier, err := auth.NewTokenVerifier(cfg)
	require.NoError(t, err)
	token, err := issuer.Issue(uuid.New(), "ext-1", "user@example.com", "admin", []string{"models:list", "profile:get"})
	require.NoError(t, err)

	a := middleware.NewAuthz(verifier, nil)
	h := a.Require(" models:list ", "models:list")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.ClaimsFromContext(r.Context())
		require.True(t, ok)
		assert.Equal(t, "user@example.com", claims.Email)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestRequireRejectsMissingScopeAndInvalidPAT(t *testing.T) {
	t.Parallel()

	cfg := testutil.MustWriteEd25519JWTConfig(t)
	issuer, err := auth.NewTokenIssuer(cfg)
	require.NoError(t, err)
	verifier, err := auth.NewTokenVerifier(cfg)
	require.NoError(t, err)

	sessionToken, err := issuer.Issue(uuid.New(), "ext-1", "user@example.com", "admin", []string{"models:list"})
	require.NoError(t, err)

	a := middleware.NewAuthz(verifier, nil)
	h := a.Require("profile:get")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "insufficient scope")

	patToken, err := issuer.IssuePAT(
		uuid.New(),
		"ext-1",
		"user@example.com",
		"admin",
		[]string{"models:list"},
		"jti-1",
		time.Now().Add(time.Hour),
	)
	require.NoError(t, err)

	a = middleware.NewAuthz(
		verifier,
		stubPATValidatorHTTP{err: &service.Error{Code: http.StatusUnauthorized, Message: "revoked token"}},
	)
	h = a.Require("models:list")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req = httptest.NewRequest(http.MethodGet, "/api/models", nil)
	req.Header.Set("Authorization", "Bearer "+patToken)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "revoked token")
}

func TestContextWithClaims(t *testing.T) {
	t.Parallel()

	claims := &auth.Claims{Email: "user@example.com"}
	ctx := middleware.ContextWithClaims(context.Background(), claims)
	got, ok := middleware.ClaimsFromContext(ctx)
	require.True(t, ok)
	assert.Same(t, claims, got)
}
