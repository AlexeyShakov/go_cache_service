package httpmiddleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// RequestIDKey — ключ, по которому request_id хранится в context.Context.
const RequestIDKey string = "request_id"

// RequestID извлекает X-Request-Id из входящего запроса.
// Если заголовок не передан клиентом, генерирует новый.
// Затем сохраняет request_id в context.Context и прокидывает его дальше по цепочке.
// Также дублирует request_id в заголовок ответа, чтобы клиент мог его видеть.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()

			requestID := req.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = uuid.NewString()
			}

			// Кладём request_id в context.Context текущего HTTP-запроса.
			ctx := context.WithValue(req.Context(), RequestIDKey, requestID)

			// Подменяем request внутри echo.Context, чтобы дальше все обработчики
			// и сервисы работали уже с новым context.
			c.SetRequest(req.WithContext(ctx))

			// Возвращаем request_id в ответе.
			c.Response().Header().Set("X-Request-Id", requestID)

			return next(c)
		}
	}
}
