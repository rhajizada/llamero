package logging

import (
	"log/slog"
	"os"
)

// New creates a JSON slog logger that writes to stdout with source information.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))
}
