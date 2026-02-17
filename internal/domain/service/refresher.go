package service

import (
	"context"
	"errors"
	"github.com/yourname/go_cache_service/internal/domain"
	"log/slog"
	"math/rand"
	"time"
)

type Worker struct {
	refreshFn    func(context.Context) error
	interval     time.Duration
	ctxTimeout   time.Duration
	logger       *slog.Logger
	jitterMaxVal int
}

// Run запускает бесконечный цикл по интервальному обновлению кэша
// В случае ошибок в рефрешере или отмены контекста цикл останавливается
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

func jitterValue(jitterMaxVal int) time.Duration {
	// random int in range [0, jitterMaxVal]
	n := rand.Intn(jitterMaxVal + 1)

	return time.Duration(n) * time.Minute
}

func NewRefresher(refresh func(ctx context.Context) error, cfg Config, logger *slog.Logger) *Worker {
	return &Worker{
		refreshFn:    refresh,
		interval:     cfg.Interval,
		ctxTimeout:   cfg.Timeout,
		logger:       logger,
		jitterMaxVal: cfg.JitterMaxVal,
	}
}
