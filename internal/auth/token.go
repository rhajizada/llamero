package auth

import (
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/rhajizada/llamero/internal/config"
)

// TokenIssuer signs internal Llamero JWTs.
type TokenIssuer struct {
	cfg        config.JWTConfig
	method     jwt.SigningMethod
	privateKey any
}

// NewTokenIssuer loads the signing key from disk and prepares the issuer.
func NewTokenIssuer(cfg config.JWTConfig) (*TokenIssuer, error) {
	method, err := signingMethod(cfg.SigningMethod)
	if err != nil {
		return nil, err
	}

	keyBytes, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	key, err := parsePrivateKey(keyBytes, method)
	if err != nil {
		return nil, err
	}

	return &TokenIssuer{
		cfg:        cfg,
		method:     method,
		privateKey: key,
	}, nil
}

// Issue signs a JWT with the supplied identity metadata.
func (i *TokenIssuer) Issue(userID uuid.UUID, externalSub, email, role string, scopes []string) (string, error) {
	if userID == uuid.Nil {
		return "", errors.New("user id cannot be empty")
	}
	if externalSub == "" {
		return "", errors.New("external subject cannot be empty")
	}
	if len(scopes) == 0 {
		return "", errors.New("scopes cannot be empty")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":     i.cfg.Issuer,
		"sub":     userID.String(),
		"ext_sub": externalSub,
		"email":   email,
		"role":    role,
		"scopes":  scopes,
		"type":    "session",
		"jti":     uuid.NewString(),
		"iat":     now.Unix(),
		"exp":     now.Add(i.cfg.TTL).Unix(),
		"aud":     i.cfg.Audience,
	}

	t := jwt.NewWithClaims(i.method, claims)
	return t.SignedString(i.privateKey)
}

func signingMethod(name string) (jwt.SigningMethod, error) {
	switch strings.ToUpper(name) {
	case "EDDSA":
		return jwt.SigningMethodEdDSA, nil
	case "RS256":
		return jwt.SigningMethodRS256, nil
	default:
		return nil, fmt.Errorf("unsupported signing method %q", name)
	}
}

func parsePrivateKey(raw []byte, method jwt.SigningMethod) (any, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("invalid PEM block for private key")
	}

	switch method {
	case jwt.SigningMethodEdDSA:
		key, err := parseEd25519PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return key, nil
	case jwt.SigningMethodRS256:
		key, err := parseRSAPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return key, nil
	default:
		return nil, fmt.Errorf("method %s not implemented", method.Alg())
	}
}

func parseEd25519PrivateKey(der []byte) (ed25519.PrivateKey, error) {
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse ed25519 pkcs8: %w", err)
	}
	edKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("pkcs8 key is not ed25519")
	}
	return edKey, nil
}

func parseRSAPrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse rsa pkcs8: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("pkcs8 key is not rsa")
	}
	return rsaKey, nil
}

// LoadReader is exposed for tests.
func LoadReader(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
