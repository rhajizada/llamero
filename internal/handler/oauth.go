package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/models"
	oauthclient "github.com/rhajizada/llamero/internal/oauth"
	"github.com/rhajizada/llamero/internal/repository"
	"github.com/rhajizada/llamero/internal/xslices"
)

var _ models.User

const maxOAuthCallbackFormBytes int64 = 64 << 10

func ParseOAuthCallbackForm(w http.ResponseWriter, r *http.Request) error {
	return parseOAuthCallbackForm(w, r)
}

// Health reports a basic status.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Login kicks off the OAuth authorization code flow.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	state := h.state.Issue()
	authURL, err := h.oauth.BuildAuthorizeURL(state)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "build auth url", "err", err)
		writeError(w, http.StatusInternalServerError, "configuration error")
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback exchanges the authorization code, fetches user info, and issues a JWT.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := parseOAuthCallbackForm(w, r); err != nil {
		if errors.As(err, new(*http.MaxBytesError)) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid form payload")
		}
		return
	}

	if errStr := r.Form.Get("error"); errStr != "" {
		writeError(w, http.StatusBadRequest, errStr)
		return
	}

	code := r.Form.Get("code")
	state := r.Form.Get("state")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing authorization code")
		return
	}
	if ok := h.state.Consume(state); !ok {
		writeError(w, http.StatusBadRequest, "invalid state parameter")
		return
	}

	tokenResp, err := h.oauth.ExchangeCode(ctx, code)
	if err != nil {
		h.logger.ErrorContext(ctx, "exchange code", "err", err)
		writeError(w, http.StatusBadGateway, "token exchange failed")
		return
	}
	if tokenResp.AccessToken == "" {
		writeError(w, http.StatusBadGateway, "provider did not return access token")
		return
	}

	user, err := h.oauth.FetchUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		h.logger.ErrorContext(ctx, "fetch user info", "err", err)
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
		LastLoginAt: new(time.Now()),
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "upsert user", "err", err)
		writeServiceError(w, err, http.StatusInternalServerError, "failed to persist user")
		return
	}

	token, err := h.issuer.Issue(upserted.ID, user.Subject, user.Email, role, scopes)
	if err != nil {
		h.logger.ErrorContext(ctx, "issue token", "err", err)
		writeError(w, http.StatusInternalServerError, "token issuance failed")
		return
	}

	redirectTarget := h.loginRedirectURL(token)
	http.Redirect(w, r, redirectTarget, http.StatusFound)
}

func parseOAuthCallbackForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxOAuthCallbackFormBytes)
	return r.ParseForm()
}

// Profile godoc
// @Summary Get authenticated user profile
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profile [get].
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
		writeServiceError(w, err, http.StatusInternalServerError, "failed to load profile")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) loginRedirectURL(token string) string {
	base := strings.TrimRight(h.cfg.ExternalURL, "/")
	if base == "" {
		base = "/"
	}

	params := url.Values{}
	params.Set("token", token)
	params.Set("expires_in", strconv.Itoa(int(h.cfg.JWT.TTL.Seconds())))

	return fmt.Sprintf("%s/login#%s", base, params.Encode())
}

func (h *Handler) determineRole(info *oauthclient.UserInfo) (string, []string, error) {
	groups := xslices.UniqueTrimmedStrings(info.Groups)
	if len(groups) == 0 {
		return "", nil, fmt.Errorf("user %s is not in any authorized groups", info.Subject)
	}

	role, scopes, ok := h.roles.Resolve(groups)
	if !ok || len(scopes) == 0 {
		return "", nil, fmt.Errorf("user %s is not in any authorized groups", info.Subject)
	}
	return role, scopes, nil
}
