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

// newLogger создаёт JSON-логгер для приложения.
func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// main является точкой входа приложения.
// Выполняет инициализацию зависимостей (Redis, кэш, сервис, воркер, HTTP),
// создаёт единый контекст приложения и запускает сервер и фоновый воркер.
// Корректно завершает работу по сигналам SIGINT/SIGTERM.
func main() {
	// Инициализация конфигурации и подключение к Redis.
	redisCfg, err := redis.LoadRedisConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Контекст старта используется только для подключения/проверки Redis.
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

	// Сборка доменных зависимостей: in-memory кэш и сервис.
	memCache := cache.NewInMemoryCache()
	logger := newLogger()
	svc := service.NewCacheService(repo, memCache, redisCfg.UpdateBatchLen, logger)

	// Сборка воркера периодического обновления.
	refCfg, err := service.LoadRefresherConfig()
	if err != nil {
		log.Fatal(err)
	}

	ref := service.NewRefresher(svc.Refresh, refCfg, logger)

	// Сборка HTTP-слоя.
	h := transport.NewHandlers(svc)
	e := transport.NewServer(h)

	// Единый контекст приложения управляет жизненным циклом HTTP-сервера и воркера.
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запуск воркера; при ошибке останавливаем приложение.
	go func() {
		if err := ref.Run(appCtx); err != nil {
			logger.Error("worker exited with error", "err", err)
			stop()
		}
	}()

	sc := echo.StartConfig{
		Address: ":8080",
	}

	// Запуск HTTP-сервера до отмены appCtx.
	if err := sc.Start(appCtx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	log.Println("server exited gracefully")
}
