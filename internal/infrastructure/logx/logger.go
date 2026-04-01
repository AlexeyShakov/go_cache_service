package logx

import (
	"context"
	"log/slog"

	httpmiddleware "github.com/yourname/go_cache_service/internal/infrastructure/http/middleware"
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

	requestID, ok := ctx.Value(httpmiddleware.RequestIDKey).(string)
	if !ok || requestID == "" {
		return logger
	}

	return logger.With(httpmiddleware.RequestIDKey, requestID)
}
