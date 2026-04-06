package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/requestctx"
)

func TestExtractParams(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/models/llama3", nil)
	req.SetPathValue("modelID", "llama3")
	params := middleware.ExtractParams("GET /api/models/{modelID}", req)
	if params["modelID"] != "llama3" {
		t.Fatalf("unexpected params: %#v", params)
	}
	if got := middleware.ExtractParams("", req); got != nil {
		t.Fatalf("expected nil params for empty pattern, got %#v", got)
	}
}

func TestLoggingMiddlewareAndWrappedWriter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	mw := middleware.Logging(logger)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := requestctx.WithRoutePattern(r.Context(), "GET /api/models/{modelID}")
		ctx = requestctx.WithBackendID(ctx, "backend-1")
		r = r.WithContext(ctx)
		r.SetPathValue("modelID", "llama3")
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/models/llama3", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	output := buf.String()
	for _, want := range []string{"status=201", "method=GET", "path=/api/models/llama3", "route=\"GET /api/models/{modelID}\"", "backend_id=backend-1"} {
		t.Run(want, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(output, want) {
				t.Fatalf("expected log output to contain %q, got %s", want, output)
			}
		})
	}
}
