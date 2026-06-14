package http

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	httpmiddleware "github.com/yourname/go_cache_service/internal/infrastructure/http/middleware"
)

// NewServer создаёт и настраивает HTTP-сервер на базе Echo.
// Регистрирует middleware и маршруты, связывая их с обработчиками.
func NewServer(h *Handlers) *echo.Echo {
	e := echo.New()

	// ВАЖНО: сначала кладём request_id в context
	e.Use(httpmiddleware.RequestID())

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/health", h.Health)
	e.GET("/cache/:key", h.GetValue)

	return e
}
