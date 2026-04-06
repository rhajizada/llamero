package auth_test

import (
	"testing"

	"github.com/rhajizada/llamero/internal/auth"
)

func TestClaimsHasScopes(t *testing.T) {
	t.Parallel()

	claims := &auth.Claims{Scopes: []string{"models:read", "models:write"}}
	if !claims.HasScopes(nil) {
		t.Fatal("expected empty scope requirement to pass")
	}
	if !claims.HasScopes([]string{"models:read"}) {
		t.Fatal("expected known scope to pass")
	}
	if claims.HasScopes([]string{"models:delete"}) {
		t.Fatal("expected missing scope to fail")
	}
}
