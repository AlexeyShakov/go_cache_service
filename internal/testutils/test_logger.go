package testutils

import (
	"bytes"
	"log/slog"
)

// InitLogger инициализирует объект логгера.
func InitLogger(out *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}
