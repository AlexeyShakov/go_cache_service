package http

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// NewServer создаёт и настраивает HTTP-сервер на базе Echo.
// Регистрирует middleware и маршруты, связывая их с обработчиками.
func NewServer(h *Handlers) *echo.Echo {
	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/health", h.Health)
	e.GET("/cache/:key", h.GetValue)

	return e
}
