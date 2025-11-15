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

// Issue signs a standard session JWT with the supplied identity metadata.
func (i *TokenIssuer) Issue(userID uuid.UUID, externalSub, email, role string, scopes []string) (string, error) {
	payload := issuePayload{
		UserID:      userID,
		ExternalSub: externalSub,
		Email:       email,
		Role:        role,
		Scopes:      scopes,
		TokenType:   TokenTypeSession,
		JTI:         uuid.NewString(),
		ExpiresAt:   time.Now().Add(i.cfg.TTL),
	}
	return i.issue(payload)
}

// IssuePAT signs a personal access token using caller-provided expiry and identifier.
func (i *TokenIssuer) IssuePAT(
	userID uuid.UUID,
	externalSub, email, role string,
	scopes []string,
	jti string,
	expiresAt time.Time,
) (string, error) {
	if expiresAt.IsZero() {
		return "", errors.New("expires_at cannot be empty")
	}
	payload := issuePayload{
		UserID:      userID,
		ExternalSub: externalSub,
		Email:       email,
		Role:        role,
		Scopes:      scopes,
		TokenType:   TokenTypePAT,
		JTI:         jti,
		ExpiresAt:   expiresAt,
	}
	return i.issue(payload)
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

type issuePayload struct {
	UserID      uuid.UUID
	ExternalSub string
	Email       string
	Role        string
	Scopes      []string
	TokenType   string
	JTI         string
	ExpiresAt   time.Time
}

func (i *TokenIssuer) issue(payload issuePayload) (string, error) {
	if payload.UserID == uuid.Nil {
		return "", errors.New("user id cannot be empty")
	}
	if payload.ExternalSub == "" {
		return "", errors.New("external subject cannot be empty")
	}
	if payload.Email == "" {
		return "", errors.New("email cannot be empty")
	}
	if payload.Role == "" {
		return "", errors.New("role cannot be empty")
	}
	if len(payload.Scopes) == 0 {
		return "", errors.New("scopes cannot be empty")
	}
	if payload.TokenType == "" {
		return "", errors.New("token type cannot be empty")
	}
	if payload.JTI == "" {
		return "", errors.New("jti cannot be empty")
	}
	now := time.Now()
	if payload.ExpiresAt.Before(now) {
		return "", errors.New("expires_at must be in the future")
	}

	claims := jwt.MapClaims{
		"iss":     i.cfg.Issuer,
		"sub":     payload.UserID.String(),
		"ext_sub": payload.ExternalSub,
		"email":   payload.Email,
		"role":    payload.Role,
		"scopes":  payload.Scopes,
		"type":    payload.TokenType,
		"jti":     payload.JTI,
		"iat":     now.Unix(),
		"exp":     payload.ExpiresAt.Unix(),
		"aud":     i.cfg.Audience,
	}

	t := jwt.NewWithClaims(i.method, claims)
	return t.SignedString(i.privateKey)
}

// LoadReader is exposed for tests.
func LoadReader(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
