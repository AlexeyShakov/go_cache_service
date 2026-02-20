package redis

import (
	"errors"
	"fmt"
	redislib "github.com/redis/go-redis/v9"
	"github.com/yourname/go_cache_service/internal/domain"
	"io"
	"strings"
)

// IsRedisSideError анализирует ошибку Redis и при необходимости
// преобразует её в ошибку доменного слоя.
// Возвращает обёрнутую доменную ошибку либо nil,
// если ошибка не относится к инфраструктуре Redis.
func IsRedisSideError(err error) error {
	// Транспортные ошибки — временные сбои соединения.
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}

	// Ошибки авторизации/прав — постоянные.
	if redislib.IsAuthError(err) || redislib.IsPermissionError(err) {
		return fmt.Errorf("%w: %v", domain.ErrPermanent, err)
	}

	// WRONGTYPE — нарушение ожидаемой схемы хранения.
	if strings.HasPrefix(err.Error(), "WRONGTYPE") {
		return fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	// Временные состояния Redis (перезагрузка, кластер недоступен и т.д.).
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
