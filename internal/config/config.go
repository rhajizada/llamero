package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// ServerConfig holds every runtime option for the HTTP server.
type ServerConfig struct {
	Address     string `env:"LLAMERO_SERVER_ADDRESS" envDefault:":8080"`
	ExternalURL string `env:"LLAMERO_SERVER_EXTERNAL_URL" envDefault:"http://localhost:8080"`
	OAuth       OAuthConfig
	JWT         JWTConfig
	Roles       RoleMappingConfig
}

// OAuthConfig captures the OAuth2 provider integration points.
type OAuthConfig struct {
	ProviderName string   `env:"LLAMERO_OAUTH_PROVIDER_NAME" envDefault:"oauth"`
	ClientID     string   `env:"LLAMERO_OAUTH_CLIENT_ID,notEmpty"`
	ClientSecret string   `env:"LLAMERO_OAUTH_CLIENT_SECRET,notEmpty"`
	AuthorizeURL string   `env:"LLAMERO_OAUTH_AUTHORIZE_URL,notEmpty"`
	TokenURL     string   `env:"LLAMERO_OAUTH_TOKEN_URL,notEmpty"`
	UserInfoURL  string   `env:"LLAMERO_OAUTH_USERINFO_URL,notEmpty"`
	RedirectURL  string   `env:"LLAMERO_OAUTH_REDIRECT_URL,notEmpty"`
	Scopes       []string `env:"LLAMERO_OAUTH_SCOPES" envSeparator:"," envDefault:"openid,email,profile"`
	Audiences    []string `env:"LLAMERO_OAUTH_AUDIENCES" envSeparator:","`
}

// JWTConfig defines how internal tokens are signed.
type JWTConfig struct {
	Issuer         string        `env:"LLAMERO_JWT_ISSUER" envDefault:"llamero"`
	Audience       string        `env:"LLAMERO_JWT_AUDIENCE" envDefault:"ollama-clients"`
	PrivateKeyPath string        `env:"LLAMERO_JWT_PRIVATE_KEY_PATH,notEmpty"`
	PublicKeyPath  string        `env:"LLAMERO_JWT_PUBLIC_KEY_PATH"`
	SigningMethod  string        `env:"LLAMERO_JWT_SIGNING_METHOD" envDefault:"EdDSA"`
	TTL            time.Duration `env:"LLAMERO_JWT_TTL" envDefault:"1h"`
}

// RoleMappingConfig maps IdP group names to internal role names defined in roles.yaml.
type RoleMappingConfig struct {
	Raw    string              `env:"LLAMERO_ROLE_GROUPS"`
	Groups map[string][]string `env:"-"`
}

// LoadServer populates ServerConfig from environment variables.
func LoadServer() (*ServerConfig, error) {
	var cfg ServerConfig
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	groups, err := parseRoleGroups(cfg.Roles.Raw)
	if err != nil {
		return nil, err
	}
	cfg.Roles.Groups = groups
	return &cfg, nil
}

func parseRoleGroups(value string) (map[string][]string, error) {
	result := make(map[string][]string)
	if strings.TrimSpace(value) == "" {
		return result, nil
	}

	items := strings.Split(value, ";")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid role mapping entry %q", item)
		}

		role := strings.TrimSpace(parts[0])
		if role == "" {
			return nil, fmt.Errorf("missing role name in %q", item)
		}

		groups := strings.Split(parts[1], ",")
		for i := range groups {
			groups[i] = strings.TrimSpace(groups[i])
		}
		result[role] = dedupe(groups)
	}
	return result, nil
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
