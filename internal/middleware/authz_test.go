package middleware_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/middleware"
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

func TestValidatePAT(t *testing.T) {
	t.Parallel()

	claims := &auth.Claims{Type: auth.TokenTypeSession}
	validator := &stubPATValidator{}
	if err := middleware.ValidatePAT(context.Background(), claims, validator); err != nil {
		t.Fatalf("unexpected error for session token: %v", err)
	}
	if validator.called {
		t.Fatal("did not expect validator call for session token")
	}

	claims.Type = auth.TokenTypePAT
	if err := middleware.ValidatePAT(
		context.Background(),
		claims,
		nil,
	); !errors.Is(err, middleware.ErrPATValidationUnavailable) {
		t.Fatalf("expected validation unavailable error, got %v", err)
	}

	validator = &stubPATValidator{err: errors.New("revoked")}
	if err := middleware.ValidatePAT(context.Background(), claims, validator); err == nil || err.Error() != "revoked" {
		t.Fatalf("expected validator error, got %v", err)
	}
	if !validator.called {
		t.Fatal("expected validator to be called for PAT")
	}
}

func TestBearerTokenAndDedupe(t *testing.T) {
	t.Parallel()

	if got := middleware.BearerToken("Bearer token-value"); got != "token-value" {
		t.Fatalf("unexpected bearer token: %q", got)
	}
	if got := middleware.BearerToken("Basic token-value"); got != "" {
		t.Fatalf("expected empty token for invalid scheme, got %q", got)
	}
	if got := middleware.BearerToken("Bearer"); got != "" {
		t.Fatalf("expected empty token for malformed header, got %q", got)
	}

	got := xslices.UniqueTrimmedStrings([]string{" read ", "write", "read", ""})
	want := []string{"read", "write"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected dedupe result: got %#v want %#v", got, want)
	}
}
