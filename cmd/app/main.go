package main

import (
	"context"
	"errors"
	"github.com/yourname/go_cache_service/internal/domain/service"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/yourname/go_cache_service/internal/cache"
	transport "github.com/yourname/go_cache_service/internal/infrastructure/http"
	"github.com/yourname/go_cache_service/internal/infrastructure/redis"
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func main() {
	// --- build repo (Redis) ---
	redisCfg, err := redis.LoadRedisConfig()
	if err != nil {
		log.Fatal(err)
	}

	// short startup context just for connecting/ping
	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStartup()

	client, err := redis.NewClient(startupCtx, redisCfg)
	if err != nil {
		log.Fatal(err)
	}

	repo, err := redis.NewRedisRepository(client, redisCfg.HashKey)
	if err != nil {
		log.Fatal(err)
	}

	// --- build cache + service ---
	memCache := cache.NewInMemoryCache()
	svc := service.NewCacheService(repo, memCache, redisCfg.UpdateBatchLen)

	// --- build worker ---
	refCfg, err := service.LoadRefresherConfig()
	if err != nil {
		log.Fatal(err)
	}
	logger := newLogger()
	ref := service.NewRefresher(svc.Refresh, refCfg, logger)

	// --- build HTTP ---
	h := transport.NewHandlers(svc)
	e := transport.NewServer(h)

	// --- one app context for the whole process (worker + server) ---
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// start worker (stops when appCtx is canceled)
	go func() {
		if err := ref.Run(appCtx); err != nil {
			logger.Error("worker exited with error", "err", err)
			stop()
		}
	}()

	sc := echo.StartConfig{
		Address: ":8080",
	}

	if err := sc.Start(appCtx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	log.Println("server exited gracefully")
}
