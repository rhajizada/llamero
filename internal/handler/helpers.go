package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/httpjson"
	"github.com/rhajizada/llamero/internal/middleware"
	oauthclient "github.com/rhajizada/llamero/internal/oauth"
	backendproxy "github.com/rhajizada/llamero/internal/proxy"
	"github.com/rhajizada/llamero/internal/requestctx"
	"github.com/rhajizada/llamero/internal/service"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	httpjson.Write(w, status, payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	httpjson.WriteError(w, status, message)
}

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func (h *Handler) ReadProxyPayload(r *http.Request) ([]byte, error) {
	return h.readProxyPayload(r)
}

func (h *Handler) WriteProxyReadError(w http.ResponseWriter, err error) {
	h.writeProxyReadError(w, err)
}

func (h *Handler) LoginRedirectURL(token string) string {
	return h.loginRedirectURL(token)
}

func (h *Handler) DetermineRole(info *oauthclient.UserInfo) (string, []string, error) {
	return h.determineRole(info)
}

func (h *Handler) ExtractUserContext(w http.ResponseWriter, r *http.Request) (*auth.Claims, uuid.UUID, bool) {
	return h.extractUserContext(w, r)
}

func MissingScopes(requested, allowed []string) []string {
	return missingScopes(requested, allowed)
}

func (h *Handler) readProxyPayload(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxProxyBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxProxyBodyBytes {
		return nil, errProxyBodyTooLarge
	}
	return body, nil
}

func (h *Handler) writeProxyReadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errProxyBodyTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
	default:
		writeError(w, http.StatusBadRequest, "unable to read request body")
	}
}

func (h *Handler) forwardLLMRequest(w http.ResponseWriter, r *http.Request, model string, body []byte) {
	route, err := h.svc.RouteBackend(r.Context(), model)
	if err != nil {
		h.handleRoutingError(w, err)
		return
	}
	h.forwardLLMRequestToRoute(w, r, route, body)
}

func (h *Handler) forwardLLMRequestToRoute(
	w http.ResponseWriter,
	r *http.Request,
	route service.BackendRoute,
	body []byte,
) {
	ctx := requestctx.WithBackendID(r.Context(), route.ID)
	req := r.WithContext(ctx)

	resp, err := h.proxy.ForwardLLM(req, route.Address, body)
	if err != nil {
		h.logger.ErrorContext(req.Context(), "proxy request failed", "backend_id", route.ID, "err", err)
		writeError(w, http.StatusBadGateway, "backend request failed")
		return
	}
	defer resp.Body.Close()

	if copyErr := backendproxy.WriteResponse(w, resp); copyErr != nil {
		h.logger.ErrorContext(req.Context(), "write proxied body", "err", copyErr)
	}
}

func (h *Handler) handleRoutingError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrNoHealthyBackends) {
		writeError(w, http.StatusServiceUnavailable, "no healthy backends available")
		return
	}
	h.logger.Error("route backend", "err", err)
	writeError(w, http.StatusBadGateway, "failed to select backend")
}

func writeServiceError(w http.ResponseWriter, err error, status int, fallback string) {
	var appErr *service.Error
	if errors.As(err, &appErr) {
		writeError(w, appErr.Code, appErr.Message)
		return
	}
	writeError(w, status, fallback)
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
