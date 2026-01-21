package worker

import (
	"context"
	"github.com/yourname/go_cache_service/internal/service"
	"log"
	"time"
)

type Worker struct {
	service    *service.CacheService
	interval   time.Duration
	ctxTimeout time.Duration
}

func (r *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	r.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.refresh(ctx)
		}
	}
}

func (r *Worker) refresh(ctx context.Context) {
	timeoutCtx, cancel := context.WithTimeout(ctx, r.ctxTimeout)
	err := r.service.Refresh(timeoutCtx)
	cancel()
	if err != nil {
		log.Println(err)
	}
}

func NewRefresher(service *service.CacheService, cfg Config) *Worker {
	return &Worker{
		service:    service,
		interval:   cfg.Interval,
		ctxTimeout: cfg.Timeout,
	}
}
