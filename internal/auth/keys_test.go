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

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/config"
)

func TestRSATokenIssuerAndVerifierRoundTrip(t *testing.T) {
	t.Parallel()

	cfg := writeRSAJWTConfig(t)
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
	if claims.Email != "user@example.com" || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestNewTokenVerifierFallsBackToPrivateKey(t *testing.T) {
	t.Parallel()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey: %v", err)
	}
	path := filepath.Join(t.TempDir(), "private.der")
	if writeErr := os.WriteFile(path, privDER, 0o600); writeErr != nil {
		t.Fatalf("write private key: %v", writeErr)
	}
	cfg := config.JWTConfig{PrivateKeyPath: path, SigningMethod: "EdDSA"}
	verifier, err := auth.NewTokenVerifier(cfg)
	if err != nil {
		t.Fatalf("NewTokenVerifier returned error: %v", err)
	}
	if verifier == nil {
		t.Fatal("expected verifier")
	}
}

func TestInvalidKeyMaterialErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	privPath := filepath.Join(dir, "invalid-private.pem")
	pubPath := filepath.Join(dir, "invalid-public.pem")
	if err := os.WriteFile(privPath, []byte("not pem"), 0o600); err != nil {
		t.Fatalf("write private file: %v", err)
	}
	if err := os.WriteFile(pubPath, []byte("not pem"), 0o600); err != nil {
		t.Fatalf("write public file: %v", err)
	}

	issuerCfg := config.JWTConfig{PrivateKeyPath: privPath, SigningMethod: "EdDSA"}
	if _, err := auth.NewTokenIssuer(issuerCfg); err == nil || !strings.Contains(err.Error(), "invalid PEM block") {
		t.Fatalf("expected invalid private key error, got %v", err)
	}

	verifierCfg := config.JWTConfig{PublicKeyPath: pubPath, PrivateKeyPath: privPath, SigningMethod: "EdDSA"}
	if _, err := auth.NewTokenVerifier(verifierCfg); err == nil || !strings.Contains(err.Error(), "invalid PEM block") {
		t.Fatalf("expected invalid public key error, got %v", err)
	}
}

func TestLoadReader(t *testing.T) {
	t.Parallel()

	b, err := auth.LoadReader(strings.NewReader("hello"))
	if err != nil || string(b) != "hello" {
		t.Fatalf("unexpected LoadReader result: %q err=%v", string(b), err)
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
