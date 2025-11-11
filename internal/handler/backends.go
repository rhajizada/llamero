package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/requestctx"
	"github.com/rhajizada/llamero/internal/service"
)

var _ models.Backend

// ListBackends godoc
// @Summary List registered backends
// @Tags Backends
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Backend
// @Failure 500 {object} map[string]string
// @Router /api/admin/backends [get]
func (h *Handler) HandleListBackends(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	backends, err := h.svc.ListBackends(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list backends")
		return
	}
	writeJSON(w, http.StatusOK, backends)
}

// HandleBackendProcesses godoc
// @Summary List running models on a backend
// @Description Forwards the request to the backend's /api/ps endpoint.
// @Tags Backends
// @Produce json
// @Security BearerAuth
// @Param backendID path string true "Backend ID"
// @Success 200 {object} models.BackendProcessList
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/backends/{backendID}/ps [get]
func (h *Handler) HandleBackendProcesses(w http.ResponseWriter, r *http.Request) {
	backendID := strings.TrimSpace(r.PathValue("backendID"))
	if backendID == "" {
		writeError(w, http.StatusNotFound, "backend not found")
		return
	}

	route, err := h.svc.LookupBackendRoute(r.Context(), backendID)
	if err != nil {
		var appErr *service.Error
		if errors.As(err, &appErr) {
			writeError(w, appErr.Code, appErr.Message)
		} else {
			writeError(w, http.StatusBadGateway, "failed to resolve backend")
		}
		return
	}

	ctx := requestctx.WithBackendID(r.Context(), backendID)
	req := r.WithContext(ctx)

	resp, err := h.proxyBackendGET(req, route, "/api/ps")
	if err != nil {
		h.logger.ErrorContext(req.Context(), "proxy backend ps", "backend_id", backendID, "err", err)
		writeError(w, http.StatusBadGateway, "backend request failed")
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	removeHopHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		h.logger.ErrorContext(req.Context(), "write backend ps response", "backend_id", backendID, "err", err)
	}
}
