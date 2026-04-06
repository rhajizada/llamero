package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rhajizada/llamero/internal/config"
)

type Client interface {
	BuildAuthorizeURL(state string) (string, error)
	ExchangeCode(ctx context.Context, code string) (*TokenResponse, error)
	FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}

type HTTPClient struct {
	cfg        *config.ServerConfig
	httpClient *http.Client
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
}

type UserInfo struct {
	Subject string
	Email   string
	Name    string
	Groups  []string
}

func New(cfg *config.ServerConfig, httpClient *http.Client) *HTTPClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &HTTPClient{cfg: cfg, httpClient: httpClient}
}

func (c *HTTPClient) BuildAuthorizeURL(state string) (string, error) {
	u, err := url.Parse(c.cfg.OAuth.AuthorizeURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", c.cfg.OAuth.ClientID)
	q.Set("redirect_uri", c.cfg.OAuth.RedirectURL)
	q.Set("scope", strings.Join(c.cfg.OAuth.Scopes, " "))
	q.Set("state", state)
	if aud := strings.TrimSpace(c.cfg.JWT.Audience); aud != "" {
		q.Set("audience", aud)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *HTTPClient) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.cfg.OAuth.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.OAuth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.cfg.OAuth.ClientID, c.cfg.OAuth.ClientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("token endpoint returned %s", resp.Status)
	}

	var tr TokenResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&tr); decodeErr != nil {
		return nil, decodeErr
	}
	return &tr, nil
}

func (c *HTTPClient) FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.OAuth.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("userinfo endpoint returned %s", resp.Status)
	}

	var raw map[string]any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&raw); decodeErr != nil {
		return nil, decodeErr
	}

	info := &UserInfo{
		Subject: firstNonEmpty(getString(raw["sub"]), getString(raw["id"]), getString(raw["user_id"])),
		Email:   firstNonEmpty(getString(raw["email"]), getString(raw["preferred_username"])),
		Name:    getString(raw["name"]),
		Groups:  collectStrings(raw["groups"]),
	}
	if info.Subject == "" {
		return nil, errors.New("userinfo payload missing subject")
	}
	if info.Email == "" {
		info.Email = info.Subject
	}
	return info, nil
}
