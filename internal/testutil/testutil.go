package testutil

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/roles"
)

func MustWriteEd25519JWTConfig(t *testing.T) config.JWTConfig {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}

	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	pubPath := filepath.Join(dir, "public.pem")
	if writeErr := os.WriteFile(
		privPath,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}),
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
		SigningMethod:  "EdDSA",
		TTL:            time.Hour,
	}
}

func MustNewTokenIssuer(t *testing.T) *auth.TokenIssuer {
	t.Helper()

	issuer, err := auth.NewTokenIssuer(MustWriteEd25519JWTConfig(t))
	if err != nil {
		t.Fatalf("new token issuer: %v", err)
	}
	return issuer
}

func MustLoadRoles(t *testing.T, raw string, groups map[string][]string) *roles.Store {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "roles.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(raw)), 0o600); err != nil {
		t.Fatalf("write roles file: %v", err)
	}
	store, err := roles.Load(path, groups)
	if err != nil {
		t.Fatalf("load roles: %v", err)
	}
	return store
}
