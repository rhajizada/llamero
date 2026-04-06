package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestTokenIssuerAndVerifierRoundTrip(t *testing.T) {
	t.Parallel()

	cfg := testutil.MustWriteEd25519JWTConfig(t)

	issuer, err := auth.NewTokenIssuer(cfg)
	if err != nil {
		t.Fatalf("NewTokenIssuer returned error: %v", err)
	}

	verifier, err := auth.NewTokenVerifier(cfg)
	if err != nil {
		t.Fatalf("NewTokenVerifier returned error: %v", err)
	}

	token, err := issuer.Issue(uuid.New(), "ext-123", "user@example.com", "admin", []string{"models:read"})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	claims, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if claims.Email != "user@example.com" || claims.Role != "admin" || claims.Type != auth.TokenTypeSession {
		t.Fatalf("unexpected verified claims: %#v", claims)
	}
}

func TestIssuePATValidatesInputs(t *testing.T) {
	t.Parallel()

	cfg := testutil.MustWriteEd25519JWTConfig(t)
	issuer, err := auth.NewTokenIssuer(cfg)
	if err != nil {
		t.Fatalf("NewTokenIssuer returned error: %v", err)
	}

	if _, issueErr := issuer.IssuePAT(
		uuid.New(),
		"ext-123",
		"user@example.com",
		"admin",
		[]string{"models:read"},
		"",
		time.Now().Add(time.Hour),
	); issueErr == nil {
		t.Fatal("expected empty jti to fail")
	}
	if _, issueErr := issuer.IssuePAT(
		uuid.New(),
		"ext-123",
		"user@example.com",
		"admin",
		[]string{"models:read"},
		"jti-1",
		time.Time{},
	); issueErr == nil {
		t.Fatal("expected zero expiry to fail")
	}
}

func TestNewTokenIssuerRejectsUnsupportedSigningMethod(t *testing.T) {
	t.Parallel()

	cfg := testutil.MustWriteEd25519JWTConfig(t)
	cfg.SigningMethod = "HS256"
	if _, err := auth.NewTokenIssuer(cfg); err == nil {
		t.Fatal("expected unsupported signing method to fail")
	}
}
