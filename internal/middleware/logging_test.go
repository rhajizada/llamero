package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/middleware"
	"github.com/rhajizada/llamero/internal/requestctx"
)

func TestExtractParams(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/models/llama3", nil)
	req.SetPathValue("modelID", "llama3")
	tests := []struct {
		name    string
		pattern string
		want    map[string]string
		wantNil bool
	}{
		{
			name:    "extracts route params",
			pattern: "GET /api/models/{modelID}",
			want:    map[string]string{"modelID": "llama3"},
		},
		{name: "returns nil for empty pattern", pattern: "", wantNil: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			params := middleware.ExtractParams(tc.pattern, req)
			if tc.wantNil {
				assert.Nil(t, params)
				return
			}
			assert.Equal(t, tc.want, params)
		})
	}

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

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/models/llama3", nil))

	output := buf.String()
	for _, want := range []string{"status=201", "method=GET", "path=/api/models/llama3", "route=\"GET /api/models/{modelID}\"", "backend_id=backend-1"} {
		t.Run("log contains "+want, func(t *testing.T) {
			t.Parallel()
			require.NotEmpty(t, output)
			assert.Contains(t, output, want)
		})
	}
}
