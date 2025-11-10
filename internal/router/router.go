package router

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/rhajizada/llamero/internal/handler"
	"github.com/rhajizada/llamero/internal/middleware"
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
	if authz != nil {
		r.Handle("/auth/me", http.HandlerFunc(h.Profile), authz.Require("chat"))
		r.Handle("/admin/backends", http.HandlerFunc(h.HandleListBackends), authz.Require("admin"))
	} else {
		r.Handle("/auth/me", http.HandlerFunc(h.Profile))
	}
	return r
}

// Handle registers a route with optional middleware wrappers.
func (r *Router) Handle(path string, handler http.Handler, wrappers ...func(http.Handler) http.Handler) {
	wrapped := handler
	for i := len(wrappers) - 1; i >= 0; i-- {
		wrapped = wrappers[i](wrapped)
	}
	r.mux.Handle(path, wrapped)
}
