package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/handler"
	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/requestctx"
	"github.com/rhajizada/llamero/internal/router"
)

func TestRouterHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "builds full router",
			run: func(t *testing.T) {
				h := handler.NewTestHandler(&config.ServerConfig{}, nil)
				authz := middleware.NewAuthz(nil, nil)
				r := router.New(h, authz)
				require.NotNil(t, r)

				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
				r.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "stores route pattern and wraps middleware",
			run: func(t *testing.T) {
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

				assert.Equal(t, http.StatusAccepted, rec.Code)
				assert.Equal(t, "yes", rec.Header().Get("X-Wrapped"))
				assert.Equal(t, "GET /api/test/{id}", route)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}
