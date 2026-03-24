// Package http содержит HTTP-обработчики,
package http

import (
	"errors"
	"github.com/yourname/go_cache_service/internal/domain/service"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/yourname/go_cache_service/internal/domain"
)

// Handlers объединяет HTTP-обработчики сервиса кэша.
// Не содержит бизнес-логики и делегирует выполнение CacheService.
type Handlers struct {
	svc *service.CacheService
}

// Health возвращает статус доступности сервиса.
// Используется для health-check и readiness probe.
func (h *Handlers) Health(c *echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

// GetValue обрабатывает запрос получения значения по ключу.
// Контекст запроса передаётся в сервис.
// Доменные ошибки маппятся в соответствующие HTTP-коды.
func (h *Handlers) GetValue(c *echo.Context) error {
	keyStr := c.Param("key")

	val, err := h.svc.GetByKey(c.Request().Context(), keyStr)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return c.String(http.StatusNotFound, "not found")
		}
		return c.String(http.StatusInternalServerError, "internal error")
	}
	return c.Blob(http.StatusOK, "application/octet-stream", []byte(val))
}

// NewHandlers создаёт набор HTTP-обработчиков поверх переданного сервиса.
func NewHandlers(svc *service.CacheService) *Handlers {
	return &Handlers{svc: svc}
}
