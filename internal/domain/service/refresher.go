package service

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/yourname/go_cache_service/internal/domain"
	"github.com/yourname/go_cache_service/internal/infrastructure/logx"
)

// Worker периодически обновляет состояние кэша, вызывая refreshFn.
// Применяет jitter к интервалу, чтобы избежать синхронных обновлений между инстансами.
// Останавливается при отмене контекста.
type Worker struct {
	refreshFn    func(context.Context) error
	interval     time.Duration
	ctxTimeout   time.Duration
	logger       *slog.Logger
	jitterMaxVal int
}

// Run запускает цикл обновления с интервалом и jitter.
// Каждая попытка обновления выполняется с таймаутом ctxTimeout.
// При domain.ErrCancelled воркер завершает работу без ошибки,
// при domain.ErrPermanent — завершает работу с ошибкой.
func (w *Worker) Run(ctx context.Context) error {
	extCtx := operationCtx(ctx, []string{logx.WorkerID, logx.RefreshID})
	logger := logx.WithContext(extCtx, w.logger)
	logger.Info("worker started",
		"interval", w.interval.String(),
		"timeout", w.ctxTimeout.String(),
	)
	// Добавляем джиттеринг, чтобы разные инстансы приложения не обновляли кэш в один момент,
	// что даст повышенную нагрузку на БД
	jVal := jitterValue(w.jitterMaxVal)
	logger.Info("Значение джиттеринга", "value", jVal.Minutes())
	ticker := time.NewTicker(w.interval + jVal)
	defer ticker.Stop()
	err := w.refresh(extCtx)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCancelled):
			logger.Info("Воркер отменен")
			return nil
		case errors.Is(err, domain.ErrPermanent):
			logger.Error("Первая попытка обновления кэша закончилась неудачей (permanent)", "err", err)
		default:
			logger.Warn("Первая попытка обновления кэша закончилась временной неудачей (transient)", "err", err)
		}
	}
	for {
		refreshCtx := operationCtx(extCtx, []string{logx.RefreshID})
		logger := logx.WithContext(refreshCtx, w.logger)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := w.refresh(refreshCtx)
			if err == nil {
				continue
			}
			switch {
			case errors.Is(err, domain.ErrCancelled):
				// Во время shutdown — это нормально, не ошибка.
				logger.Info("Воркер отменен")
				return nil

			case errors.Is(err, domain.ErrPermanent):
				logger.Error("Попытка обновления кэша закончилась неудачей (permanent)", "err", err)
				return err
			default:
				logger.Warn("Попытка обновления кэша закончилась временной неудачей (permanent)", "err", err)
			}
		}
	}
}

// refresh выполняет одну попытку обновления с ограничением по времени.
// Возвращает ошибку от refreshFn без преобразований.
func (w *Worker) refresh(ctx context.Context) error {
	start := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, w.ctxTimeout)
	defer cancel()
	err := w.refreshFn(timeoutCtx)
	if err != nil {
		return err
	}
	dur := time.Since(start)
	logger := logx.WithContext(ctx, w.logger)
	logger.Info("cache refreshed", "duration_ms", dur.Milliseconds())
	return nil
}

// operationCtx добавляет в контекст информацию об айди воркера или айди для итерации рефреша
func operationCtx(ctx context.Context, actions []string) context.Context {
	for _, action := range actions {
		switch action {
		case logx.WorkerID:
			ctx = context.WithValue(ctx, logx.WorkerID, uuid.NewString())

		case logx.RefreshID:
			ctx = context.WithValue(ctx, logx.RefreshID, uuid.NewString())
		}
	}
	return ctx
}

// jitterValue возвращает случайное смещение интервала в пределах [0..jitterMaxVal] минут.
func jitterValue(jitterMaxVal int) time.Duration {
	n := rand.Intn(jitterMaxVal + 1)

	return time.Duration(n) * time.Minute
}

// NewRefresher создаёт Worker на основе refresh-функции и конфигурации.
// refresh вызывается периодически и должен быть идемпотентным.
func NewRefresher(refresh func(ctx context.Context) error, cfg RefreshConfig, logger *slog.Logger) *Worker {
	return &Worker{
		refreshFn:    refresh,
		interval:     cfg.Interval,
		ctxTimeout:   cfg.Timeout,
		logger:       logger,
		jitterMaxVal: cfg.JitterMaxVal,
	}
}
