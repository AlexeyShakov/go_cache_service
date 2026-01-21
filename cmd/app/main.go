package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"

	"github.com/yourname/go_cache_service/internal/cache"
	transport "github.com/yourname/go_cache_service/internal/infrastructure/http"
	"github.com/yourname/go_cache_service/internal/infrastructure/redis"
	"github.com/yourname/go_cache_service/internal/service"
	"github.com/yourname/go_cache_service/internal/worker"
)

func main() {
	// Load .env (optional for local dev)
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}

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
	svc := service.NewCacheService(repo, memCache)

	// --- build worker ---
	refCfg, err := worker.LoadRefresherConfig()
	if err != nil {
		log.Fatal(err)
	}
	ref := worker.NewRefresher(svc, refCfg)

	// --- build HTTP ---
	h := transport.NewHandlers(svc)
	e := transport.NewServer(h)

	// --- one app context for the whole process (worker + server) ---
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// start worker (stops when appCtx is canceled)
	go ref.Run(appCtx)

	sc := echo.StartConfig{
		Address: ":8080",
	}

	if err := sc.Start(appCtx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	log.Println("server exited gracefully")
}
