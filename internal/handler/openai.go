package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/rhajizada/llamero/internal/service"
)

const maxProxyBodyBytes int64 = 5 << 20 // 5 MiB

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"TE",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

var errProxyBodyTooLarge = errors.New("request body too large")

// ChatCompletionProxyRequest represents the subset of OpenAI fields that Llamero inspects.
type ChatCompletionProxyRequest struct {
	Model string `json:"model"`
} // @name ChatCompletionProxyRequest

// EmbeddingsProxyRequest represents the subset of OpenAI fields that Llamero inspects.
type EmbeddingsProxyRequest struct {
	Model string `json:"model"`
} // @name EmbeddingsProxyRequest

// CompletionProxyRequest represents the subset of completion fields inspected for routing.
type CompletionProxyRequest struct {
	Model string `json:"model"`
}

// HandleChatCompletions proxies OpenAI-compatible chat requests to an Ollama backend.
func (h *Handler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := h.readProxyPayload(r)
	if err != nil {
		h.writeProxyReadError(w, err)
		return
	}

	var payload ChatCompletionProxyRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(payload.Model) == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	h.forwardOpenAIRequest(w, r, payload.Model, body)
}

// HandleEmbeddings proxies OpenAI-compatible embedding requests to an Ollama backend.
func (h *Handler) HandleEmbeddings(w http.ResponseWriter, r *http.Request) {
	body, err := h.readProxyPayload(r)
	if err != nil {
		h.writeProxyReadError(w, err)
		return
	}

	var payload EmbeddingsProxyRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(payload.Model) == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	h.forwardOpenAIRequest(w, r, payload.Model, body)
}

// HandleCompletions proxies legacy OpenAI completion requests to an Ollama backend.
func (h *Handler) HandleCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := h.readProxyPayload(r)
	if err != nil {
		h.writeProxyReadError(w, err)
		return
	}

	var payload CompletionProxyRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(payload.Model) == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	h.forwardOpenAIRequest(w, r, payload.Model, body)
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

func (h *Handler) forwardOpenAIRequest(w http.ResponseWriter, r *http.Request, model string, body []byte) {
	route, err := h.svc.RouteBackend(r.Context(), model)
	if err != nil {
		h.handleRoutingError(w, err)
		return
	}

	resp, err := h.proxyToBackend(r, route, body)
	if err != nil {
		h.logger.Printf("proxy request failed (backend=%s): %v", route.ID, err)
		writeError(w, http.StatusBadGateway, "backend request failed")
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	removeHopHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		h.logger.Printf("write proxied body: %v", err)
	}
}

func (h *Handler) handleRoutingError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrNoHealthyBackends) {
		writeError(w, http.StatusServiceUnavailable, "no healthy backends available")
		return
	}
	h.logger.Printf("route backend: %v", err)
	writeError(w, http.StatusBadGateway, "failed to select backend")
}

func (h *Handler) proxyToBackend(r *http.Request, route service.BackendRoute, body []byte) (*http.Response, error) {
	target := strings.TrimRight(route.Address, "/")
	path := r.URL.Path
	if path == "" || !strings.HasPrefix(path, "/") {
		path = "/" + strings.TrimPrefix(path, "/")
	}
	target += path
	if raw := r.URL.RawQuery; raw != "" {
		target += "?" + raw
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	copyHeaders(req.Header, r.Header)
	removeHopHeaders(req.Header)
	req.Header.Del("Authorization")
	req.Header.Del("Content-Length")
	applyForwardHeaders(req, r)

	return h.client.Do(req)
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func removeHopHeaders(h http.Header) {
	for _, header := range hopHeaders {
		h.Del(header)
	}
}

func applyForwardHeaders(out *http.Request, orig *http.Request) {
	clientIP := remoteIP(orig.RemoteAddr)
	if prior := orig.Header.Get("X-Forwarded-For"); prior != "" {
		if clientIP != "" {
			clientIP = prior + ", " + clientIP
		} else {
			clientIP = prior
		}
	}
	if clientIP != "" {
		out.Header.Set("X-Forwarded-For", clientIP)
	}
	if orig.Host != "" {
		out.Header.Set("X-Forwarded-Host", orig.Host)
	}
	proto := orig.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if orig.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	out.Header.Set("X-Forwarded-Proto", proto)
	if port := forwardedPort(orig); port != "" {
		out.Header.Set("X-Forwarded-Port", port)
	}
}

func remoteIP(addr string) string {
	if addr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func forwardedPort(r *http.Request) string {
	if port := r.Header.Get("X-Forwarded-Port"); port != "" {
		return port
	}
	if r.URL != nil && r.URL.Port() != "" {
		return r.URL.Port()
	}
	if r.TLS != nil {
		return "443"
	}
	return "80"
}
