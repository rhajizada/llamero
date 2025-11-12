package router

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/rhajizada/llamero/internal/handler"
	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/requestctx"
)

// Router wires URL paths to handler methods.
type Router struct {
	mux *http.ServeMux
}

// New builds the HTTP routing table.
func New(h *handler.Handler, authz *middleware.Authz) *Router {
	if authz == nil {
		panic("auth middleware is required")
	}
	r := &Router{
		mux: http.NewServeMux(),
	}
	r.mux.Handle("/api/docs/", httpSwagger.WrapHandler)
	r.Handle("/healthz", http.HandlerFunc(h.Health))
	r.Handle("/auth/login", http.HandlerFunc(h.Login))
	r.Handle("/auth/callback", http.HandlerFunc(h.Callback))
	r.Handle("/api/profile", http.HandlerFunc(h.Profile), authz.Require("profile:get"))
	r.Handle("/api/backends", http.HandlerFunc(h.HandleListBackends), authz.Require("backends:list"))
	r.Handle(
		"GET /api/backends/{backendID}/ps",
		http.HandlerFunc(h.HandleBackendProcesses),
		authz.Require("backends:ps"),
	)
	r.Handle(
		"POST /api/backends/{backendID}/create",
		http.HandlerFunc(h.HandleBackendCreate),
		authz.Require("backends:createModel"),
	)
	r.Handle(
		"POST /api/backends/{backendID}/copy",
		http.HandlerFunc(h.HandleBackendCopy),
		authz.Require("backends:createModel"),
	)
	r.Handle(
		"POST /api/backends/{backendID}/pull",
		http.HandlerFunc(h.HandleBackendPull),
		authz.Require("backends:pullModel"),
	)
	r.Handle(
		"POST /api/backends/{backendID}/push",
		http.HandlerFunc(h.HandleBackendPush),
		authz.Require("backends:pushModel"),
	)
	r.Handle(
		"DELETE /api/backends/{backendID}/delete",
		http.HandlerFunc(h.HandleBackendDelete),
		authz.Require("backends:deleteModel"),
	)
	r.Handle(
		"POST /api/backends/{backendID}/show",
		http.HandlerFunc(h.HandleBackendShow),
		authz.Require("backends:listModels"),
	)
	r.Handle(
		"GET /api/backends/{backendID}/version",
		http.HandlerFunc(h.HandleBackendVersion),
		authz.Require("backends:list"),
	)
	r.Handle("/api/models", http.HandlerFunc(h.HandleListModels), authz.Require("models:list"))
	r.Handle("GET /api/models/{modelID}", http.HandlerFunc(h.HandleGetModel), authz.Require("models:list"))
	r.Handle("/api/chat/completions", http.HandlerFunc(h.HandleChatCompletions), authz.Require("llm:chat"))
	r.Handle("/api/completions", http.HandlerFunc(h.HandleCompletions), authz.Require("llm:chat"))
	r.Handle("/api/embeddings", http.HandlerFunc(h.HandleEmbeddings), authz.Require("llm:embeddings"))
	return r
}

// Handle registers a route with optional middleware wrappers.
func (r *Router) Handle(path string, handler http.Handler, wrappers ...func(http.Handler) http.Handler) {
	base := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := requestctx.WithRoutePattern(req.Context(), path)
		handler.ServeHTTP(w, req.WithContext(ctx))
	})

	var wrapped http.Handler = base
	for i := len(wrappers) - 1; i >= 0; i-- {
		wrapped = wrappers[i](wrapped)
	}
	r.mux.Handle(path, wrapped)
}
