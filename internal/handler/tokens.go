package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/service"
)

const (
	defaultPATTTL = 30 * 24 * time.Hour
	maxPATTTL     = 90 * 24 * time.Hour
	minPATTTL     = time.Hour
)

// CreatePersonalAccessTokenRequest defines the payload for PAT creation.
type CreatePersonalAccessTokenRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresIn int64    `json:"expires_in,omitempty"`
} // @name CreatePersonalAccessTokenRequest

// PersonalAccessTokenResponse documents PAT responses.
type PersonalAccessTokenResponse struct {
	models.PersonalAccessToken
	Token string `json:"token,omitempty"`
} // @name PersonalAccessTokenResponse

// HandleListTokens godoc
// @Summary List personal access tokens
// @Tags Profile
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.PersonalAccessToken
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profile/tokens [get].
func (h *Handler) HandleListTokens(w http.ResponseWriter, r *http.Request) {
	claims, userID, ok := h.extractUserContext(w, r)
	if !ok {
		return
	}
	if claims.Type == auth.TokenTypePAT {
		writeError(w, http.StatusForbidden, "personal access tokens cannot manage other tokens")
		return
	}

	tokens, err := h.svc.ListPersonalAccessTokens(r.Context(), userID)
	if err != nil {
		var appErr *service.Error
		if errors.As(err, &appErr) {
			writeError(w, appErr.Code, appErr.Message)
		} else {
			writeError(w, http.StatusInternalServerError, "failed to list tokens")
		}
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// HandleGetToken godoc
// @Summary Get personal access token metadata
// @Tags Profile
// @Produce json
// @Security BearerAuth
// @Param tokenID path string true "Token ID"
// @Success 200 {object} models.PersonalAccessToken
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profile/tokens/{tokenID} [get].
func (h *Handler) HandleGetToken(w http.ResponseWriter, r *http.Request) {
	claims, userID, ok := h.extractUserContext(w, r)
	if !ok {
		return
	}
	if claims.Type == auth.TokenTypePAT {
		writeError(w, http.StatusForbidden, "personal access tokens cannot manage other tokens")
		return
	}

	tokenID, err := uuid.Parse(strings.TrimSpace(r.PathValue("tokenID")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid token id")
		return
	}

	token, err := h.svc.GetPersonalAccessToken(r.Context(), userID, tokenID)
	if err != nil {
		var appErr *service.Error
		if errors.As(err, &appErr) {
			writeError(w, appErr.Code, appErr.Message)
		} else {
			writeError(w, http.StatusInternalServerError, "failed to load token")
		}
		return
	}

	writeJSON(w, http.StatusOK, token)
}

// HandleCreateToken godoc
// @Summary Create a personal access token
// @Tags Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body CreatePersonalAccessTokenRequest true "PAT request"
// @Success 201 {object} PersonalAccessTokenResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profile/tokens [post].
func (h *Handler) HandleCreateToken(w http.ResponseWriter, r *http.Request) {
	claims, userID, ok := h.extractUserContext(w, r)
	if !ok {
		return
	}
	if claims.Type == auth.TokenTypePAT {
		writeError(w, http.StatusForbidden, "personal access tokens cannot manage other tokens")
		return
	}

	var req CreatePersonalAccessTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	scopes := dedupeStrings(req.Scopes)
	if len(scopes) == 0 {
		writeError(w, http.StatusBadRequest, "at least one scope is required")
		return
	}
	if invalid := missingScopes(scopes, claims.Scopes); len(invalid) > 0 {
		writeError(w, http.StatusForbidden, "requested scopes exceed current permissions")
		return
	}

	expiresIn := req.ExpiresIn
	if expiresIn == 0 {
		expiresIn = int64(defaultPATTTL.Seconds())
	}
	if expiresIn < int64(minPATTTL.Seconds()) {
		writeError(w, http.StatusBadRequest, "expires_in must be at least 1 hour")
		return
	}
	if expiresIn > int64(maxPATTTL.Seconds()) {
		writeError(w, http.StatusBadRequest, "expires_in cannot exceed 90 days")
		return
	}
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	params := service.CreateTokenParams{
		UserID:    userID,
		Name:      name,
		Scopes:    scopes,
		TokenType: auth.TokenTypePAT,
		JTI:       uuid.NewString(),
		ExpiresAt: expiresAt,
	}

	tokenMeta, err := h.svc.CreatePersonalAccessToken(r.Context(), params)
	if err != nil {
		var appErr *service.Error
		if errors.As(err, &appErr) {
			writeError(w, appErr.Code, appErr.Message)
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create token")
		}
		return
	}

	externalSub := claims.ExternalSub
	if strings.TrimSpace(externalSub) == "" {
		externalSub = claims.Subject
	}

	tokenString, err := h.issuer.IssuePAT(
		userID,
		externalSub,
		claims.Email,
		claims.Role,
		scopes,
		params.JTI,
		expiresAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	resp := PersonalAccessTokenResponse{
		PersonalAccessToken: tokenMeta,
		Token:               tokenString,
	}
	writeJSON(w, http.StatusCreated, resp)
}

// HandleDeleteToken godoc
// @Summary Revoke a personal access token
// @Tags Profile
// @Security BearerAuth
// @Param tokenID path string true "Token ID"
// @Success 204 {string} string ""
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profile/tokens/{tokenID} [delete].
func (h *Handler) HandleDeleteToken(w http.ResponseWriter, r *http.Request) {
	claims, userID, ok := h.extractUserContext(w, r)
	if !ok {
		return
	}
	if claims.Type == auth.TokenTypePAT {
		writeError(w, http.StatusForbidden, "personal access tokens cannot manage other tokens")
		return
	}

	tokenID, err := uuid.Parse(strings.TrimSpace(r.PathValue("tokenID")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid token id")
		return
	}

	if err := h.svc.RevokePersonalAccessToken(r.Context(), userID, tokenID); err != nil {
		var appErr *service.Error
		if errors.As(err, &appErr) {
			writeError(w, appErr.Code, appErr.Message)
		} else {
			writeError(w, http.StatusInternalServerError, "failed to revoke token")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) extractUserContext(w http.ResponseWriter, r *http.Request) (*auth.Claims, uuid.UUID, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing authentication context")
		return nil, uuid.Nil, false
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user identifier")
		return nil, uuid.Nil, false
	}

	return claims, userID, true
}

func missingScopes(requested, allowed []string) []string {
	if len(requested) == 0 {
		return nil
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, scope := range allowed {
		allowedSet[scope] = struct{}{}
	}
	var invalid []string
	for _, scope := range requested {
		if _, ok := allowedSet[scope]; !ok {
			invalid = append(invalid, scope)
		}
	}
	return invalid
}
