package service

import (
	"context"
	"errors"
	"github.com/yourname/go_cache_service/internal/domain"
	"log/slog"
	"math/rand"
	"time"
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
func (r *Worker) Run(ctx context.Context) error {
	r.logger.Info("worker started",
		"interval", r.interval.String(),
		"timeout", r.ctxTimeout.String(),
	)
	// Добавляем джиттеринг, чтобы разные инстансы приложения не обновляли кэш в один момент,
	// что даст повышенную нагрузку на БД
	ticker := time.NewTicker(r.interval + jitterValue(r.jitterMaxVal))
	defer ticker.Stop()
	err := r.refresh(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrCancelled) {
			// Во время shutdown — это нормально, не ошибка.
			r.logger.Info("Воркер отменен")
			return nil
		}
		if errors.Is(err, domain.ErrPermanent) {
			r.logger.Error("Первая попытка по обновлению кэша закончилась неудачей (permanent)", "err", err)
		}
		r.logger.Warn("Первая попытка по обновлению кэша закончилась временной неудачей (transient)", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			//TODO может сделать замер времмени?
			err := r.refresh(ctx)
			if err == nil {
				r.logger.Info("Кэш обновлен")
				continue
			}
			switch {
			case errors.Is(err, domain.ErrCancelled):
				// Во время shutdown — это нормально, не ошибка.
				r.logger.Info("Воркер отменен")
				return nil

			case errors.Is(err, domain.ErrPermanent):
				r.logger.Error("Попытка обновления кэша закончилась неудачей (permanent)", "err", err)
				return err
			default:
				r.logger.Warn("Попытка обновления кэша закончилась временной неудачей (permanent)", "err", err)
			}
		}
	}
}

// refresh выполняет одну попытку обновления с ограничением по времени.
// Возвращает ошибку от refreshFn без преобразований.
func (r *Worker) refresh(ctx context.Context) error {
	start := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, r.ctxTimeout)
	defer cancel()
	err := r.refreshFn(timeoutCtx)
	if err != nil {
		return err
	}
	dur := time.Since(start)
	r.logger.Info("cache refreshed", "duration_ms", dur.Milliseconds())
	return nil
}

// jitterValue возвращает случайное смещение интервала в пределах [0..jitterMaxVal] минут.
func jitterValue(jitterMaxVal int) time.Duration {
	n := rand.Intn(jitterMaxVal + 1)

	return time.Duration(n) * time.Minute
}

// NewRefresher создаёт Worker на основе refresh-функции и конфигурации.
// refresh вызывается периодически и должен быть идемпотентным.
func NewRefresher(refresh func(ctx context.Context) error, cfg Config, logger *slog.Logger) *Worker {
	return &Worker{
		refreshFn:    refresh,
		interval:     cfg.Interval,
		ctxTimeout:   cfg.Timeout,
		logger:       logger,
		jitterMaxVal: cfg.JitterMaxVal,
	}
}
