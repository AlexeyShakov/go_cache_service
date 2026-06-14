package logx

import (
	"context"
	"log/slog"
)

const (
	HttpID    string = "request_id"
	WorkerID  string = "worker_id"
	RefreshID string = "refresh_id"
)

// WithContext возвращает логгер, обогащённый значениями из context.Context.
//
// В частности, извлекает request_id из контекста и добавляет его в логгер.
// Если request_id отсутствует или пустой, возвращает исходный логгер без изменений.
//
// Если переданный logger равен nil, используется slog.Default().
func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	if v, ok := ctx.Value(HttpID).(string); ok && v != "" {
		logger = logger.With(HttpID, v)
	}
	if v, ok := ctx.Value(WorkerID).(string); ok && v != "" {
		logger = logger.With(WorkerID, v)
	}
	if v, ok := ctx.Value(RefreshID).(string); ok && v != "" {
		logger = logger.With(RefreshID, v)
	}
	return logger
}
