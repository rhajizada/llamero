package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/service"
)

var _ models.User

// Health reports a basic status.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Login kicks off the OAuth authorization code flow.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	state := h.state.Issue()
	authURL, err := h.buildAuthorizeURL(state)
	if err != nil {
		h.logger.Printf("build auth url: %v", err)
		writeError(w, http.StatusInternalServerError, "configuration error")
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback exchanges the authorization code, fetches user info, and issues a JWT.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if errStr := r.FormValue("error"); errStr != "" {
		writeError(w, http.StatusBadRequest, errStr)
		return
	}

	code := r.FormValue("code")
	state := r.FormValue("state")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing authorization code")
		return
	}
	if ok := h.state.Consume(state); !ok {
		writeError(w, http.StatusBadRequest, "invalid state parameter")
		return
	}

	tokenResp, err := h.exchangeCode(ctx, code)
	if err != nil {
		h.logger.Printf("exchange code: %v", err)
		writeError(w, http.StatusBadGateway, "token exchange failed")
		return
	}
	if tokenResp.AccessToken == "" {
		writeError(w, http.StatusBadGateway, "provider did not return access token")
		return
	}

	user, err := h.fetchUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		h.logger.Printf("fetch userinfo: %v", err)
		writeError(w, http.StatusBadGateway, "user info request failed")
		return
	}

	role, scopes, err := h.determineRole(user)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	upserted, err := h.svc.UpsertUser(ctx, repository.UpsertUserParams{
		Sub:         user.Subject,
		Provider:    h.cfg.OAuth.ProviderName,
		Email:       user.Email,
		DisplayName: nullableString(user.Name),
		Role:        role,
		Scopes:      scopes,
		Groups:      user.Groups,
		LastLoginAt: timePtr(time.Now()),
	})
	if err != nil {
		h.logger.Printf("upsert user: %v", err)
		var appErr *service.Error
		if errors.As(err, &appErr) {
			writeError(w, appErr.Code, appErr.Message)
		} else {
			writeError(w, http.StatusInternalServerError, "failed to persist user")
		}
		return
	}

	token, err := h.issuer.Issue(upserted.ID, user.Subject, user.Email, role, scopes)
	if err != nil {
		h.logger.Printf("issue token: %v", err)
		writeError(w, http.StatusInternalServerError, "token issuance failed")
		return
	}

	resp := map[string]any{
		"token":       token,
		"token_type":  "Bearer",
		"expires_in":  int(h.cfg.JWT.TTL.Seconds()),
		"role":        role,
		"scopes":      scopes,
		"provider":    h.cfg.OAuth.ProviderName,
		"issued_at":   time.Now().UTC().Format(time.RFC3339),
		"redirect_to": h.cfg.ExternalURL,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) buildAuthorizeURL(state string) (string, error) {
	u, err := url.Parse(h.cfg.OAuth.AuthorizeURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", h.cfg.OAuth.ClientID)
	q.Set("redirect_uri", h.cfg.OAuth.RedirectURL)
	q.Set("scope", strings.Join(h.cfg.OAuth.Scopes, " "))
	q.Set("state", state)
	if len(h.cfg.OAuth.Audiences) > 0 {
		q.Set("audience", strings.Join(h.cfg.OAuth.Audiences, " "))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (h *Handler) exchangeCode(ctx context.Context, code string) (*tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", h.cfg.OAuth.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.cfg.OAuth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(h.cfg.OAuth.ClientID, h.cfg.OAuth.ClientSecret)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token endpoint returned %s", resp.Status)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func (h *Handler) fetchUserInfo(ctx context.Context, accessToken string) (*userInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.cfg.OAuth.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("userinfo endpoint returned %s", resp.Status)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	info := &userInfo{
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

func (h *Handler) determineRole(info *userInfo) (string, []string, error) {
	groups := dedupeStrings(info.Groups)
	if len(groups) == 0 {
		return "", nil, fmt.Errorf("user %s is not in any authorized groups", info.Subject)
	}

	role, scopes, ok := h.roles.Resolve(groups)
	if !ok || len(scopes) == 0 {
		return "", nil, fmt.Errorf("user %s is not in any authorized groups", info.Subject)
	}
	return role, scopes, nil
}

// Profile godoc
// @Summary Get authenticated user profile
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/users/me [get]
func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing authentication context")
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid user identifier")
		return
	}

	user, err := h.svc.GetUser(r.Context(), userID)
	if err != nil {
		var appErr *service.Error
		if errors.As(err, &appErr) {
			writeError(w, appErr.Code, appErr.Message)
		} else {
			writeError(w, http.StatusInternalServerError, "failed to load profile")
		}
		return
	}

	writeJSON(w, http.StatusOK, user)
}
