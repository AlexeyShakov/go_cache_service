package redis

import (
	"context"
	"errors"
	"fmt"
	redislib "github.com/redis/go-redis/v9"
	"github.com/yourname/go_cache_service/internal/domain"
	"io"
	"net"
	"strings"
)

// MapError нормализует инфраструктурные ошибки Redis в доменные категории.
// Приоритет: ошибки контекста → транспорт → ошибки Redis-сервера → internal.
func MapError(ctx context.Context, err error) error {
	// Контекст: таймаут/отмена.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", domain.ErrTimeout, err)
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("%w: %v", domain.ErrCancelled, err)
	}

	// Транспортные ошибки — временные.
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}

	// Ошибки авторизации/прав — постоянные.
	if redislib.IsAuthError(err) || redislib.IsPermissionError(err) {
		return fmt.Errorf("%w: %v", domain.ErrPermanent, err)
	}

	// WRONGTYPE — нарушение схемы.
	if strings.HasPrefix(err.Error(), "WRONGTYPE") {
		return fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	// Временные состояния Redis.
	if redislib.IsLoadingError(err) ||
		redislib.IsTryAgainError(err) ||
		redislib.IsClusterDownError(err) ||
		redislib.IsMasterDownError(err) ||
		redislib.IsMaxClientsError(err) ||
		redislib.IsOOMError(err) ||
		redislib.IsReadOnlyError(err) {
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}

	// Не удалось классифицировать — internal.
	return fmt.Errorf("%w: %v", domain.ErrInternal, err)
}
