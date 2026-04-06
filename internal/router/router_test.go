package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/handler"
	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/requestctx"
	"github.com/rhajizada/llamero/internal/router"
)

func TestNew(t *testing.T) {
	t.Parallel()

	h := handler.NewTestHandler(&config.ServerConfig{}, nil)
	authz := middleware.NewAuthz(nil, nil)
	r := router.New(h, authz)
	if r == nil {
		t.Fatal("expected router")
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestHandleWrapsRoutePatternAndMiddleware(t *testing.T) {
	t.Parallel()

	r := router.NewMux()
	var route string
	r.Handle("GET /api/test/{id}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		route, _ = requestctx.RoutePattern(req.Context())
		w.WriteHeader(http.StatusAccepted)
	}), func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Wrapped", "yes")
			next.ServeHTTP(w, req)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test/123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if rec.Header().Get("X-Wrapped") != "yes" {
		t.Fatal("expected wrapper to run")
	}
	if route != "GET /api/test/{id}" {
		t.Fatalf("unexpected stored route pattern: %q", route)
	}
}
