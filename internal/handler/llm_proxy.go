package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/rhajizada/llamero/internal/models"
)

var (
	_ models.ChatCompletionRequest
	_ models.ChatCompletionResponse
	_ models.CompletionRequest
	_ models.CompletionResponse
	_ models.EmbeddingsRequest
	_ models.EmbeddingsResponse
	_ models.ResponsesCreateRequest
	_ models.ResponsesResponse
)

const maxProxyBodyBytes int64 = 5 << 20 // 5 MiB

var errProxyBodyTooLarge = errors.New("request body too large")

// ChatCompletionProxyRequest represents the subset of LLM fields that Llamero inspects.
type ChatCompletionProxyRequest struct {
	Model string `json:"model"`
} // @name ChatCompletionProxyRequest

// EmbeddingsProxyRequest represents the subset of LLM fields that Llamero inspects.
type EmbeddingsProxyRequest struct {
	Model string `json:"model"`
} // @name EmbeddingsProxyRequest

// CompletionProxyRequest represents the subset of completion fields inspected for routing.
type CompletionProxyRequest struct {
	Model string `json:"model"`
}

// ResponsesProxyCreateRequest represents the subset of responses fields inspected for routing.
type ResponsesProxyCreateRequest struct {
	Model string `json:"model"`
}

// HandleChatCompletions godoc
// @Summary Proxy chat completions
// @Tags OpenAI
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChatCompletionRequest true "Chat completion payload"
// @Success 200 {object} models.ChatCompletionResponse
// @Failure 400 {object} map[string]string
// @Failure 413 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/chat/completions [post].
func (h *Handler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := h.readProxyPayload(r)
	if err != nil {
		h.writeProxyReadError(w, err)
		return
	}

	var payload ChatCompletionProxyRequest
	if decodeErr := json.Unmarshal(body, &payload); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(payload.Model) == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	h.forwardLLMRequest(w, r, payload.Model, body)
}

// HandleEmbeddings godoc
// @Summary Proxy embeddings
// @Tags OpenAI
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.EmbeddingsRequest true "Embeddings payload"
// @Success 200 {object} models.EmbeddingsResponse
// @Failure 400 {object} map[string]string
// @Failure 413 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/embeddings [post].
func (h *Handler) HandleEmbeddings(w http.ResponseWriter, r *http.Request) {
	body, err := h.readProxyPayload(r)
	if err != nil {
		h.writeProxyReadError(w, err)
		return
	}

	var payload EmbeddingsProxyRequest
	if decodeErr := json.Unmarshal(body, &payload); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(payload.Model) == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	h.forwardLLMRequest(w, r, payload.Model, body)
}

// HandleCompletions godoc
// @Summary Proxy legacy completions
// @Tags OpenAI
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CompletionRequest true "Completion payload"
// @Success 200 {object} models.CompletionResponse
// @Failure 400 {object} map[string]string
// @Failure 413 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/completions [post].
func (h *Handler) HandleCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := h.readProxyPayload(r)
	if err != nil {
		h.writeProxyReadError(w, err)
		return
	}

	var payload CompletionProxyRequest
	if decodeErr := json.Unmarshal(body, &payload); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(payload.Model) == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	h.forwardLLMRequest(w, r, payload.Model, body)
}

// HandleResponsesCreate godoc
// @Summary Create a model response
// @Tags OpenAI
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ResponsesCreateRequest true "Responses payload"
// @Success 200 {object} models.ResponsesResponse
// @Failure 400 {object} map[string]string
// @Failure 413 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/responses [post].
func (h *Handler) HandleResponsesCreate(w http.ResponseWriter, r *http.Request) {
	body, err := h.readProxyPayload(r)
	if err != nil {
		h.writeProxyReadError(w, err)
		return
	}

	var payload ResponsesProxyCreateRequest
	if decodeErr := json.Unmarshal(body, &payload); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(payload.Model) == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	route, err := h.svc.RouteResponsesCreate(r.Context(), payload.Model)
	if err != nil {
		h.handleRoutingError(w, err)
		return
	}
	h.forwardLLMRequestToRoute(w, r, route, body)
}
