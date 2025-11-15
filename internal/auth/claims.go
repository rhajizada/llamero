package auth

import "github.com/golang-jwt/jwt/v5"

const (
	// TokenTypeSession identifies interactive session tokens.
	TokenTypeSession = "session"
	// TokenTypePAT identifies personal access tokens.
	TokenTypePAT = "pat"
)

// Claims represents the JWT payload used across Llamero services.
type Claims struct {
	jwt.RegisteredClaims

	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Scopes      []string `json:"scopes"`
	Type        string   `json:"type"`
	ExternalSub string   `json:"ext_sub,omitempty"`
}

// HasScopes returns true when all required scopes exist in the claim set.
func (c *Claims) HasScopes(required []string) bool {
	if len(required) == 0 {
		return true
	}
	available := make(map[string]struct{}, len(c.Scopes))
	for _, scope := range c.Scopes {
		available[scope] = struct{}{}
	}
	for _, scope := range required {
		if _, ok := available[scope]; !ok {
			return false
		}
	}
	return true
}
