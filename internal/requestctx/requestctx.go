package requestctx

import "context"

type ctxKey struct{}

// Data stores request-scoped metadata surfaced to middleware.
type Data struct {
	RoutePattern string
	BackendID    string
}

// Ensure attaches request metadata storage to the context and returns the derived context.
func Ensure(ctx context.Context) context.Context {
	newCtx, _ := ensure(ctx)
	return newCtx
}

// WithRoutePattern records the resolved mux pattern inside the context.
func WithRoutePattern(ctx context.Context, pattern string) context.Context {
	ctx, data := ensure(ctx)
	data.RoutePattern = pattern
	return ctx
}

// RoutePattern retrieves the mux pattern stored in the context, if present.
func RoutePattern(ctx context.Context) (string, bool) {
	if data := dataFrom(ctx); data != nil && data.RoutePattern != "" {
		return data.RoutePattern, true
	}
	return "", false
}

// WithBackendID records the backend identifier used for proxying.
func WithBackendID(ctx context.Context, backendID string) context.Context {
	if backendID == "" {
		return ctx
	}
	ctx, data := ensure(ctx)
	data.BackendID = backendID
	return ctx
}

// BackendID retrieves the backend identifier stored in the context, if present.
func BackendID(ctx context.Context) (string, bool) {
	if data := dataFrom(ctx); data != nil && data.BackendID != "" {
		return data.BackendID, true
	}
	return "", false
}

func ensure(ctx context.Context) (context.Context, *Data) {
	if data := dataFrom(ctx); data != nil {
		return ctx, data
	}
	data := &Data{}
	return context.WithValue(ctx, ctxKey{}, data), data
}

func dataFrom(ctx context.Context) *Data {
	if ctx == nil {
		return nil
	}
	data, _ := ctx.Value(ctxKey{}).(*Data)
	return data
}
