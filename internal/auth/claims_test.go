package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rhajizada/llamero/internal/auth"
)

func TestClaimsHasScopes(t *testing.T) {
	t.Parallel()

	claims := &auth.Claims{Scopes: []string{"models:read", "models:write"}}
	tests := []struct {
		name     string
		required []string
		want     bool
	}{
		{name: "allows empty requirements", required: nil, want: true},
		{name: "allows known scope", required: []string{"models:read"}, want: true},
		{name: "rejects unknown scope", required: []string{"models:delete"}, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, claims.HasScopes(tc.required))
		})
	}
}
