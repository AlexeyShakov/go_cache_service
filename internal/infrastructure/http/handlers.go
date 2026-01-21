package http

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/yourname/go_cache_service/internal/domain"
	"github.com/yourname/go_cache_service/internal/service"
)

type Handlers struct {
	svc *service.CacheService
}

func NewHandlers(svc *service.CacheService) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) Health(c *echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func (h *Handlers) GetValue(c *echo.Context) error {
	keyStr := c.Param("key")
	key := domain.Key(keyStr)

	val, err := h.svc.GetByKey(c.Request().Context(), key)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return c.String(http.StatusNotFound, "not found")
		}
		return c.String(http.StatusInternalServerError, "internal error")
	}
	return c.Blob(http.StatusOK, "application/octet-stream", []byte(val))
}
