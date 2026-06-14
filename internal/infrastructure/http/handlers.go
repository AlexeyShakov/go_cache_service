// Package http содержит HTTP-обработчики.
package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/yourname/go_cache_service/internal/domain"
	"github.com/yourname/go_cache_service/internal/domain/service"
	"github.com/yourname/go_cache_service/internal/infrastructure/logx"
)

// Handlers объединяет HTTP-обработчики сервиса кэша.
// Не содержит бизнес-логики и делегирует выполнение CacheService.
type Handlers struct {
	svc    *service.CacheService
	logger *slog.Logger
}

// Health возвращает статус доступности сервиса.
// Используется для health-check и readiness probe.
func (h *Handlers) Health(c *echo.Context) error {
	logger := logx.WithContext(c.Request().Context(), h.logger)
	logger.Info("health check called")

	return c.String(http.StatusOK, "ok")
}

// GetValue обрабатывает запрос получения значения по ключу.
// Контекст запроса передаётся в сервис.
// Доменные ошибки маппятся в соответствующие HTTP-коды.
func (h *Handlers) GetValue(c *echo.Context) error {
	logger := logx.WithContext(c.Request().Context(), h.logger)

	keyStr := c.Param("key")
	logger.Info("get value request started", "key", keyStr)

	val, err := h.svc.GetByKey(c.Request().Context(), keyStr)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			logger.Warn("value not found", "key", keyStr)
			return c.String(http.StatusNotFound, "not found")
		}
		logger.Error("failed to get value", "key", keyStr, "err", err)
		return c.String(http.StatusInternalServerError, "internal error")
	}

	logger.Info("value returned successfully", "key", keyStr)
	return c.Blob(http.StatusOK, "application/octet-stream", []byte(val))
}

// NewHandlers создаёт набор HTTP-обработчиков поверх переданного сервиса.
func NewHandlers(svc *service.CacheService, logger *slog.Logger) *Handlers {
	return &Handlers{
		svc:    svc,
		logger: logger,
	}
}
