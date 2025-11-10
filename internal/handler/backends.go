package handler

import (
	"net/http"

	"github.com/rhajizada/llamero/internal/models"
)

var _ models.Backend

// ListBackends godoc
// @Summary List registered backends
// @Tags Backends
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Backend
// @Failure 500 {object} map[string]string
// @Router /admin/backends [get]
func (h *Handler) HandleListBackends(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	backends, err := h.svc.ListBackends(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list backends")
		return
	}
	writeJSON(w, http.StatusOK, backends)
}
