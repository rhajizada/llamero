package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/service"
	"github.com/rhajizada/llamero/internal/testutil"
)

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
	if err != nil {
		t.Fatalf("new token issuer: %v", err)
	}
	verifier, err := auth.NewTokenVerifier(cfg)
	if err != nil {
		t.Fatalf("new token verifier: %v", err)
	}
	token, err := issuer.Issue(uuid.New(), "ext-1", "user@example.com", "admin", []string{"models:list", "profile:get"})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	a := middleware.NewAuthz(verifier, nil)
	h := a.Require(" models:list ", "models:list")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.ClaimsFromContext(r.Context())
		if !ok || claims.Email != "user@example.com" {
			t.Fatalf("expected claims in context, got %#v %v", claims, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRequireRejectsMissingScopeAndInvalidPAT(t *testing.T) {
	t.Parallel()

	cfg := testutil.MustWriteEd25519JWTConfig(t)
	issuer, err := auth.NewTokenIssuer(cfg)
	if err != nil {
		t.Fatalf("new token issuer: %v", err)
	}
	verifier, err := auth.NewTokenVerifier(cfg)
	if err != nil {
		t.Fatalf("new token verifier: %v", err)
	}

	sessionToken, err := issuer.Issue(uuid.New(), "ext-1", "user@example.com", "admin", []string{"models:list"})
	if err != nil {
		t.Fatalf("issue session token: %v", err)
	}

	a := middleware.NewAuthz(verifier, nil)
	h := a.Require("profile:get")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "insufficient scope") {
		t.Fatalf("unexpected insufficient scope response: code=%d body=%s", rec.Code, rec.Body.String())
	}

	patToken, err := issuer.IssuePAT(
		uuid.New(),
		"ext-1",
		"user@example.com",
		"admin",
		[]string{"models:list"},
		"jti-1",
		time.Now().Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("issue pat token: %v", err)
	}

	a = middleware.NewAuthz(
		verifier,
		stubPATValidatorHTTP{
			err: &service.Error{Code: http.StatusUnauthorized, Message: "revoked token"},
		},
	)
	h = a.Require("models:list")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req = httptest.NewRequest(http.MethodGet, "/api/models", nil)
	req.Header.Set("Authorization", "Bearer "+patToken)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "revoked token") {
		t.Fatalf("unexpected revoked pat response: code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestContextWithClaims(t *testing.T) {
	t.Parallel()

	claims := &auth.Claims{Email: "user@example.com"}
	ctx := middleware.ContextWithClaims(context.Background(), claims)
	got, ok := middleware.ClaimsFromContext(ctx)
	if !ok || got != claims {
		t.Fatalf("unexpected claims from context: %#v %v", got, ok)
	}
}
