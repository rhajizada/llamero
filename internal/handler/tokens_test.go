package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rhajizada/llamero/internal/handler"
)

func TestTokenHelpers(t *testing.T) {
	t.Parallel()

	h := &handler.Handler{}
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "extract user context rejects missing auth context",
			run: func(t *testing.T) {
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
				_, _, ok := h.ExtractUserContext(rec, req)
				assert.False(t, ok)
			},
		},
		{
			name: "missing scopes returns nil when no scopes requested",
			run: func(t *testing.T) {
				assert.Nil(t, handler.MissingScopes(nil, []string{"a"}))
			},
		},
		{
			name: "missing scopes returns values not granted",
			run: func(t *testing.T) {
				assert.Equal(t, []string{"c"}, handler.MissingScopes([]string{"a", "c"}, []string{"a", "b"}))
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
