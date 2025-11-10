package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/service"
)

var _ models.Model
var _ models.ModelList

// HandleListModels godoc
// @Summary List available models
// @Tags Models
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.ModelList
// @Failure 500 {object} map[string]string
// @Router /v1/models [get]
func (h *Handler) HandleListModels(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListModels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list models")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// HandleGetModel godoc
// @Summary Get model metadata
// @Tags Models
// @Produce json
// @Security BearerAuth
// @Param model path string true "Model ID"
// @Success 200 {object} models.Model
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/models/{model} [get]
func (h *Handler) HandleGetModel(w http.ResponseWriter, r *http.Request) {
	modelID := strings.TrimSpace(r.PathValue("model"))
	if modelID == "" {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}
	model, err := h.svc.GetModel(r.Context(), modelID)
	if err != nil {
		var appErr *service.Error
		if errors.As(err, &appErr) {
			writeError(w, appErr.Code, appErr.Message)
		} else {
			writeError(w, http.StatusInternalServerError, "failed to load model")
		}
		return
	}
	writeJSON(w, http.StatusOK, model)
}
