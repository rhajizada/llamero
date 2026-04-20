package requestctx_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rhajizada/llamero/internal/requestctx"
)

func TestEnsureAndAccessors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "stores route pattern and backend id",
			run: func(t *testing.T) {
				ctx := requestctx.Ensure(context.Background())
				ctx = requestctx.WithRoutePattern(ctx, "/api/models")
				ctx = requestctx.WithBackendID(ctx, "backend-1")

				route, ok := requestctx.RoutePattern(ctx)
				assert.True(t, ok)
				assert.Equal(t, "/api/models", route)

				backendID, ok := requestctx.BackendID(ctx)
				assert.True(t, ok)
				assert.Equal(t, "backend-1", backendID)
			},
		},
		{
			name: "ignores empty backend id",
			run: func(t *testing.T) {
				base := context.Background()
				ctx := requestctx.WithBackendID(base, "")
				assert.Equal(t, base, ctx)
				_, ok := requestctx.BackendID(ctx)
				assert.False(t, ok)
			},
		},
		{
			name: "returns no values from empty context",
			run: func(t *testing.T) {
				_, routeOK := requestctx.RoutePattern(context.Background())
				_, backendOK := requestctx.BackendID(context.Background())
				assert.False(t, routeOK)
				assert.False(t, backendOK)
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
