package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestTokenIssuerHelpers(t *testing.T) {
	t.Parallel()

	cfg := testutil.MustWriteEd25519JWTConfig(t)
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "round trips session token",
			run: func(t *testing.T) {
				issuer, err := auth.NewTokenIssuer(cfg)
				require.NoError(t, err)

				verifier, err := auth.NewTokenVerifier(cfg)
				require.NoError(t, err)

				token, err := issuer.Issue(uuid.New(), "ext-123", "user@example.com", "admin", []string{"models:read"})
				require.NoError(t, err)

				claims, err := verifier.Verify(context.Background(), token)
				require.NoError(t, err)
				assert.Equal(t, "user@example.com", claims.Email)
				assert.Equal(t, "admin", claims.Role)
				assert.Equal(t, auth.TokenTypeSession, claims.Type)
			},
		},
		{
			name: "rejects PAT without jti",
			run: func(t *testing.T) {
				issuer, err := auth.NewTokenIssuer(cfg)
				require.NoError(t, err)

				_, issueErr := issuer.IssuePAT(
					uuid.New(),
					"ext-123",
					"user@example.com",
					"admin",
					[]string{"models:read"},
					"",
					time.Now().Add(time.Hour),
				)
				assert.Error(t, issueErr)
			},
		},
		{
			name: "rejects PAT without expiry",
			run: func(t *testing.T) {
				issuer, err := auth.NewTokenIssuer(cfg)
				require.NoError(t, err)

				_, issueErr := issuer.IssuePAT(
					uuid.New(),
					"ext-123",
					"user@example.com",
					"admin",
					[]string{"models:read"},
					"jti-1",
					time.Time{},
				)
				assert.Error(t, issueErr)
			},
		},
		{
			name: "rejects unsupported signing method",
			run: func(t *testing.T) {
				invalidCfg := cfg
				invalidCfg.SigningMethod = "HS256"
				_, err := auth.NewTokenIssuer(invalidCfg)
				assert.Error(t, err)
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
