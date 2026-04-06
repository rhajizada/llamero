package requestctx_test

import (
	"context"
	"testing"

	"github.com/rhajizada/llamero/internal/requestctx"
)

func TestEnsureAndAccessors(t *testing.T) {
	t.Parallel()

	ctx := requestctx.Ensure(context.Background())
	ctx = requestctx.WithRoutePattern(ctx, "/api/models")
	ctx = requestctx.WithBackendID(ctx, "backend-1")

	route, ok := requestctx.RoutePattern(ctx)
	if !ok || route != "/api/models" {
		t.Fatalf("unexpected route pattern: %q %v", route, ok)
	}

	backendID, ok := requestctx.BackendID(ctx)
	if !ok || backendID != "backend-1" {
		t.Fatalf("unexpected backend id: %q %v", backendID, ok)
	}
}

func TestWithBackendIDIgnoresEmptyValue(t *testing.T) {
	t.Parallel()

	base := context.Background()
	ctx := requestctx.WithBackendID(base, "")
	if ctx != base {
		t.Fatal("expected empty backend id to keep original context")
	}
	if _, ok := requestctx.BackendID(ctx); ok {
		t.Fatal("did not expect backend id in context")
	}
}

func TestNilContextLookups(t *testing.T) {
	t.Parallel()

	if _, ok := requestctx.RoutePattern(context.Background()); ok {
		t.Fatal("did not expect route pattern from nil context")
	}
	if _, ok := requestctx.BackendID(context.Background()); ok {
		t.Fatal("did not expect backend id from nil context")
	}
}
