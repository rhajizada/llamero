package auth_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/config"
)

func TestKeyLoadingAndJWTHelpers(t *testing.T) {
	t.Parallel()

	cfg := writeRSAJWTConfig(t)
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "round trips RSA token",
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
			},
		},
		{
			name: "falls back to private key when public key missing",
			run: func(t *testing.T) {
				_, priv, err := ed25519.GenerateKey(rand.Reader)
				require.NoError(t, err)
				privDER, err := x509.MarshalPKCS8PrivateKey(priv)
				require.NoError(t, err)

				path := filepath.Join(t.TempDir(), "private.der")
				require.NoError(t, os.WriteFile(path, privDER, 0o600))

				verifier, err := auth.NewTokenVerifier(config.JWTConfig{
					PrivateKeyPath: path,
					SigningMethod:  "EdDSA",
				})
				require.NoError(t, err)
				assert.NotNil(t, verifier)
			},
		},
		{
			name: "rejects invalid private key material",
			run: func(t *testing.T) {
				dir := t.TempDir()
				privPath := filepath.Join(dir, "invalid-private.pem")
				require.NoError(t, os.WriteFile(privPath, []byte("not pem"), 0o600))

				_, err := auth.NewTokenIssuer(config.JWTConfig{
					PrivateKeyPath: privPath,
					SigningMethod:  "EdDSA",
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid PEM block")
			},
		},
		{
			name: "rejects invalid public key material",
			run: func(t *testing.T) {
				dir := t.TempDir()
				privPath := filepath.Join(dir, "invalid-private.pem")
				pubPath := filepath.Join(dir, "invalid-public.pem")
				require.NoError(t, os.WriteFile(privPath, []byte("not pem"), 0o600))
				require.NoError(t, os.WriteFile(pubPath, []byte("not pem"), 0o600))

				_, err := auth.NewTokenVerifier(config.JWTConfig{
					PublicKeyPath:  pubPath,
					PrivateKeyPath: privPath,
					SigningMethod:  "EdDSA",
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid PEM block")
			},
		},
		{
			name: "loads bytes from reader",
			run: func(t *testing.T) {
				b, err := auth.LoadReader(strings.NewReader("hello"))
				require.NoError(t, err)
				assert.Equal(t, "hello", string(b))
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

func writeRSAJWTConfig(t *testing.T) config.JWTConfig {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(key)
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}

	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	pubPath := filepath.Join(dir, "public.pem")
	if writeErr := os.WriteFile(
		privPath,
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER}),
		0o600,
	); writeErr != nil {
		t.Fatalf("write private key: %v", writeErr)
	}
	if writeErr := os.WriteFile(
		pubPath,
		pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}),
		0o600,
	); writeErr != nil {
		t.Fatalf("write public key: %v", writeErr)
	}

	return config.JWTConfig{
		Issuer:         "llamero-test",
		Audience:       "clients",
		PrivateKeyPath: privPath,
		PublicKeyPath:  pubPath,
		SigningMethod:  "RS256",
		TTL:            time.Hour,
	}
}
