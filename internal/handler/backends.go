package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/rhajizada/llamero/internal/models"
	backendproxy "github.com/rhajizada/llamero/internal/proxy"
	"github.com/rhajizada/llamero/internal/requestctx"
	"github.com/rhajizada/llamero/internal/workers"
)

var (
	_ models.Backend
	_ models.BackendCreateModelRequest
	_ models.BackendCopyModelRequest
	_ models.BackendPullModelRequest
	_ models.BackendPushModelRequest
	_ models.BackendDeleteModelRequest
	_ models.BackendShowModelRequest
	_ models.BackendShowModelResponse
	_ models.BackendOperationResponse
	_ models.BackendVersionResponse
	_ models.BackendTagsResponse
)

// HandleListBackends godoc
// @Summary List registered backends
// @Tags Backends
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Backend
// @Failure 500 {object} map[string]string
// @Router /api/backends [get].
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
// @Success 200 {object} models.ProcessResponse
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/backends/{backendID}/ps [get].
func (h *Handler) HandleBackendProcesses(w http.ResponseWriter, r *http.Request) {
	h.handleBackendGET(w, r, "/api/ps")
}

// HandleBackendCreate godoc
// @Summary Create a model on the specified backend
// @Tags Backends
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param backendID path string true "Backend ID"
// @Param request body models.BackendCreateModelRequest true "Create model payload"
// @Success 200 {object} models.BackendOperationResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/backends/{backendID}/create [post].
func (h *Handler) HandleBackendCreate(w http.ResponseWriter, r *http.Request) {
	h.handleBackendProxyWithBody(w, r, http.MethodPost, "/api/create", true)
}

// HandleBackendCopy godoc
// @Summary Copy a model on the specified backend
// @Tags Backends
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param backendID path string true "Backend ID"
// @Param request body models.BackendCopyModelRequest true "Copy model payload"
// @Success 200 {string} string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/backends/{backendID}/copy [post].
func (h *Handler) HandleBackendCopy(w http.ResponseWriter, r *http.Request) {
	h.handleBackendProxyWithBody(w, r, http.MethodPost, "/api/copy", true)
}

// HandleBackendPull godoc
// @Summary Pull a model on the specified backend
// @Tags Backends
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param backendID path string true "Backend ID"
// @Param request body models.BackendPullModelRequest true "Pull model payload"
// @Success 200 {string} string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/backends/{backendID}/pull [post].
func (h *Handler) HandleBackendPull(w http.ResponseWriter, r *http.Request) {
	h.handleBackendProxyWithBody(w, r, http.MethodPost, "/api/pull", true)
}

// HandleBackendPush godoc
// @Summary Push a model from the specified backend
// @Tags Backends
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param backendID path string true "Backend ID"
// @Param request body models.BackendPushModelRequest true "Push model payload"
// @Success 200 {object} models.BackendOperationResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/backends/{backendID}/push [post].
func (h *Handler) HandleBackendPush(w http.ResponseWriter, r *http.Request) {
	h.handleBackendProxyWithBody(w, r, http.MethodPost, "/api/push", true)
}

// HandleBackendDelete godoc
// @Summary Delete a model from the specified backend
// @Tags Backends
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param backendID path string true "Backend ID"
// @Param request body models.BackendDeleteModelRequest true "Delete model payload"
// @Success 200 {object} models.BackendOperationResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/backends/{backendID}/delete [delete].
func (h *Handler) HandleBackendDelete(w http.ResponseWriter, r *http.Request) {
	h.handleBackendProxyWithBody(w, r, http.MethodDelete, "/api/delete", true)
}

// HandleBackendShow godoc
// @Summary Show model details on the specified backend
// @Tags Backends
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param backendID path string true "Backend ID"
// @Param request body models.BackendShowModelRequest true "Show model payload"
// @Success 200 {object} models.BackendShowModelResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/backends/{backendID}/show [post].
func (h *Handler) HandleBackendShow(w http.ResponseWriter, r *http.Request) {
	h.handleBackendProxyWithBody(w, r, http.MethodPost, "/api/show", false)
}

// HandleBackendTags godoc
// @Summary List available models on a backend
// @Tags Backends
// @Produce json
// @Security BearerAuth
// @Param backendID path string true "Backend ID"
// @Success 200 {object} models.BackendTagsResponse
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/backends/{backendID}/tags [get].
func (h *Handler) HandleBackendTags(w http.ResponseWriter, r *http.Request) {
	h.handleBackendGET(w, r, "/api/tags")
}

// HandleBackendVersion godoc
// @Summary Retrieve Ollama version of specified backend
// @Tags Backends
// @Produce json
// @Security BearerAuth
// @Param backendID path string true "Backend ID"
// @Success 200 {object} models.BackendVersionResponse
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/backends/{backendID}/version [get].
func (h *Handler) HandleBackendVersion(w http.ResponseWriter, r *http.Request) {
	h.handleBackendGET(w, r, "/api/version")
}

func (h *Handler) handleBackendProxyWithBody(
	w http.ResponseWriter,
	r *http.Request,
	method, backendPath string,
	sync bool,
) {
	backendID := strings.TrimSpace(r.PathValue("backendID"))
	if backendID == "" {
		writeError(w, http.StatusNotFound, "backend not found")
		return
	}
	route, err := h.svc.LookupBackendRoute(r.Context(), backendID)
	if err != nil {
		writeServiceError(w, err, http.StatusBadGateway, "failed to resolve backend")
		return
	}
	body, err := h.readProxyPayload(r)
	if err != nil {
		h.writeProxyReadError(w, err)
		return
	}

	ctx := requestctx.WithBackendID(r.Context(), backendID)
	req := r.WithContext(ctx)

	resp, err := h.proxy.Forward(req, method, route.Address, backendPath, body)
	if err != nil {
		h.logger.ErrorContext(
			req.Context(),
			"proxy backend mutation",
			"backend_id",
			backendID,
			"path",
			backendPath,
			"err",
			err,
		)
		writeError(w, http.StatusBadGateway, "backend request failed")
		return
	}
	defer resp.Body.Close()

	if copyErr := backendproxy.WriteResponse(w, resp); copyErr != nil {
		h.logger.ErrorContext(req.Context(), "write backend mutation response", "backend_id", backendID, "err", copyErr)
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < 400 {
		if sync {
			h.enqueueSyncBackendByID(req.Context(), backendID)
		}
	}
}

func (h *Handler) handleBackendGET(w http.ResponseWriter, r *http.Request, backendPath string) {
	backendID := strings.TrimSpace(r.PathValue("backendID"))
	if backendID == "" {
		writeError(w, http.StatusNotFound, "backend not found")
		return
	}

	route, err := h.svc.LookupBackendRoute(r.Context(), backendID)
	if err != nil {
		writeServiceError(w, err, http.StatusBadGateway, "failed to resolve backend")
		return
	}

	ctx := requestctx.WithBackendID(r.Context(), backendID)
	req := r.WithContext(ctx)

	resp, err := h.proxy.ForwardGET(req, route.Address, backendPath)
	if err != nil {
		h.logger.ErrorContext(
			req.Context(),
			"proxy backend get",
			"backend_id",
			backendID,
			"path",
			backendPath,
			"err",
			err,
		)
		writeError(w, http.StatusBadGateway, "backend request failed")
		return
	}
	defer resp.Body.Close()

	if copyErr := backendproxy.WriteResponse(w, resp); copyErr != nil {
		h.logger.ErrorContext(req.Context(), "write backend get response", "backend_id", backendID, "err", copyErr)
	}
}

func (h *Handler) enqueueSyncBackendByID(ctx context.Context, backendID string) {
	if h.tasks == nil {
		return
	}
	task, err := workers.NewSyncBackendByIDTask(backendID)
	if err != nil {
		h.logger.ErrorContext(ctx, "create backend sync task", "backend_id", backendID, "err", err)
		return
	}
	if _, enqueueErr := h.tasks.EnqueueContext(ctx, task); enqueueErr != nil {
		h.logger.ErrorContext(ctx, "enqueue backend sync task", "backend_id", backendID, "err", enqueueErr)
	}
}
