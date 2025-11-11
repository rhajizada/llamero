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
	r := &Router{
		mux: http.NewServeMux(),
	}
	r.mux.Handle("/api/docs/", httpSwagger.WrapHandler)
	r.Handle("/healthz", http.HandlerFunc(h.Health))
	r.Handle("/auth/login", http.HandlerFunc(h.Login))
	r.Handle("/auth/callback", http.HandlerFunc(h.Callback))
	if authz == nil {
		panic("auth middleware is required")
	}

	r.Handle("/api/users/me", http.HandlerFunc(h.Profile), authz.Require("chat"))
	r.Handle("/api/admin/backends", http.HandlerFunc(h.HandleListBackends), authz.Require("admin"))
	r.Handle("GET /api/backends/{backendID}/ps", http.HandlerFunc(h.HandleBackendProcesses), authz.Require("admin"))
	r.Handle("/api/chat/completions", http.HandlerFunc(h.HandleChatCompletions), authz.Require("chat"))
	r.Handle("/api/completions", http.HandlerFunc(h.HandleCompletions), authz.Require("generate"))
	r.Handle("/api/embeddings", http.HandlerFunc(h.HandleEmbeddings), authz.Require("embed"))
	r.Handle("/api/models", http.HandlerFunc(h.HandleListModels), authz.Require("models:read"))
	r.Handle("GET /api/models/{model}", http.HandlerFunc(h.HandleGetModel), authz.Require("models:read"))
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
