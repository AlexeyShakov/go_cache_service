package redis

import (
	"errors"
	"fmt"
	redislib "github.com/redis/go-redis/v9"
	"github.com/yourname/go_cache_service/internal/domain"
	"io"
	"strings"
)

// IsRedisSideError проверяет, что ошибка относится к Redis. Если это так, то оборачивает в ошибку доменного слоя
func IsRedisSideError(err error) error {
	// Транспортные ошибки (соединение/чтение) — обычно временные, значит Unavailable.
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}
	if redislib.IsAuthError(err) || redislib.IsPermissionError(err) {
		return fmt.Errorf("%w: %v", domain.ErrPermanent, err)
	}
	// WRONGTYPE — типичная "ошибка схемы/ключа". Для неё нет стабильного helper, используем префикс.
	if strings.HasPrefix(err.Error(), "WRONGTYPE") {
		return fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}
	// Ретраебл Redis-состояния — временные: LOADING, TRYAGAIN, CLUSTERDOWN, и т.д.
	if redislib.IsLoadingError(err) ||
		redislib.IsTryAgainError(err) ||
		redislib.IsClusterDownError(err) ||
		redislib.IsMasterDownError(err) ||
		redislib.IsMaxClientsError(err) ||
		redislib.IsOOMError(err) ||
		redislib.IsReadOnlyError(err) {
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}
	return nil
}
