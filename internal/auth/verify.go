package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/rhajizada/llamero/internal/config"
)

// TokenVerifier validates JWTs issued by Llamero.
type TokenVerifier struct {
	cfg    config.JWTConfig
	method jwt.SigningMethod
	key    any
}

// NewTokenVerifier loads the public key (or derives it from the private key) needed to verify tokens.
func NewTokenVerifier(cfg config.JWTConfig) (*TokenVerifier, error) {
	method, err := signingMethod(cfg.SigningMethod)
	if err != nil {
		return nil, err
	}

	key, err := loadVerificationKey(cfg, method)
	if err != nil {
		return nil, err
	}

	return &TokenVerifier{
		cfg:    cfg,
		method: method,
		key:    key,
	}, nil
}

// Verify parses and validates a token string into Claims.
func (v *TokenVerifier) Verify(ctx context.Context, tokenString string) (*Claims, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != v.method.Alg() {
			return nil, fmt.Errorf("unexpected signing method %s", token.Method.Alg())
		}
		return v.key, nil
	},
		jwt.WithIssuer(v.cfg.Issuer),
		jwt.WithAudience(v.cfg.Audience),
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func loadVerificationKey(cfg config.JWTConfig, method jwt.SigningMethod) (any, error) {
	if strings.TrimSpace(cfg.PublicKeyPath) != "" {
		pub, err := os.ReadFile(cfg.PublicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read public key: %w", err)
		}
		return parsePublicKey(pub, method)
	}

	// Fall back to private key if no public key was provided.
	privBytes, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	switch method {
	case jwt.SigningMethodEdDSA:
		privKey, parseErr := parseEd25519PrivateKey(privBytes)
		if parseErr != nil {
			return nil, parseErr
		}
		public, ok := privKey.Public().(ed25519.PublicKey)
		if !ok {
			return nil, errors.New("ed25519 private key missing public component")
		}
		return public, nil
	case jwt.SigningMethodRS256:
		privKey, parseErr := parseRSAPrivateKey(privBytes)
		if parseErr != nil {
			return nil, parseErr
		}
		return &privKey.PublicKey, nil
	default:
		return nil, fmt.Errorf("unsupported signing method %s", method.Alg())
	}
}

func parsePublicKey(raw []byte, method jwt.SigningMethod) (any, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("invalid PEM block for public key")
	}

	switch method {
	case jwt.SigningMethodEdDSA:
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse ed25519 public key: %w", err)
		}
		pub, ok := key.(ed25519.PublicKey)
		if !ok {
			return nil, errors.New("provided public key is not ed25519")
		}
		return pub, nil
	case jwt.SigningMethodRS256:
		key, parseErr := x509.ParsePKIXPublicKey(block.Bytes)
		if parseErr == nil {
			if rsaPub, ok := key.(*rsa.PublicKey); ok {
				return rsaPub, nil
			}
		}
		rsaPub, pkcs1Err := x509.ParsePKCS1PublicKey(block.Bytes)
		if pkcs1Err == nil {
			return rsaPub, nil
		}
		return nil, fmt.Errorf("parse rsa public key: %w", pkcs1Err)
	default:
		return nil, fmt.Errorf("unsupported signing method %s", method.Alg())
	}
}
