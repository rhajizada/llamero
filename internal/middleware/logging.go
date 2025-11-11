package middleware

import (
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rhajizada/llamero/internal/requestctx"
)

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

// Logging returns a middleware that emits structured request logs.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := requestctx.Ensure(r.Context())
			req := r.WithContext(ctx)

			wrapped := &wrappedWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			start := time.Now()
			next.ServeHTTP(wrapped, req)

			attrs := []slog.Attr{
				slog.Int("status", wrapped.statusCode),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Duration("duration", time.Since(start)),
			}

			if route, ok := requestctx.RoutePattern(ctx); ok && route != "" {
				attrs = append(attrs, slog.String("route", route))
				if params := extractParams(route, req); len(params) > 0 {
					attrs = append(attrs, slog.Any("params", params))
				}
			}

			if backendID, ok := requestctx.BackendID(ctx); ok {
				attrs = append(attrs, slog.String("backend_id", backendID))
			}

			logger.LogAttrs(ctx, slog.LevelInfo, "http request", attrs...)
		})
	}
}

var paramPattern = regexp.MustCompile(`\{([^}/]+)\}`)

func extractParams(pattern string, r *http.Request) map[string]string {
	if pattern == "" || r == nil {
		return nil
	}
	spaceIdx := strings.Index(pattern, " ")
	if spaceIdx >= 0 && spaceIdx < len(pattern)-1 {
		pattern = pattern[spaceIdx+1:]
	}

	matches := paramPattern.FindAllStringSubmatch(pattern, -1)
	if len(matches) == 0 {
		return nil
	}

	params := make(map[string]string, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if value := r.PathValue(name); value != "" {
			params[name] = value
		}
	}

	if len(params) == 0 {
		return nil
	}
	return params
}
