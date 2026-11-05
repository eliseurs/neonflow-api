package logger

import (
	"io"
	"log/slog"
	"os"
)

// New devolve JSON nos ambientes publicados (Railway) e texto legível no ambiente local.
func New(production bool) *slog.Logger {
	return newWithWriter(production, os.Stdout)
}

func newWithWriter(production bool, w io.Writer) *slog.Logger {
	if production {
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
