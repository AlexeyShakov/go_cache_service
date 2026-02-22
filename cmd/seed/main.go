package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/yourname/go_cache_service/internal/infrastructure/redis"
	"github.com/yourname/go_cache_service/internal/seed"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Инициализация конфигурации и подключение к Redis.
	redisCfg, err := redis.LoadRedisConfig()
	if err != nil {
		log.Fatal(err)
	}
	// Путь до dataset можно задавать через переменную окружения.
	// По умолчанию используем стандартный путь для локального запуска.
	datasetPath := os.Getenv("SEED_DATASET_PATH")
	if datasetPath == "" {
		datasetPath = "testdata/seed.json"
	}

	raw, err := os.ReadFile(datasetPath)
	if err != nil {
		log.Fatal(fmt.Errorf("read dataset %q: %w", datasetPath, err))
	}

	var data map[string]string
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Fatal(fmt.Errorf("parse dataset %q: %w", datasetPath, err))
	}
	// Контекст старта используется только для подключения/проверки Redis.
	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStartup()

	client, err := redis.NewClient(startupCtx, redisCfg)
	if err != nil {
		log.Fatal(err)
	}
	// Единый контекст приложения управляет жизненным циклом HTTP-сервера и воркера.
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err = seed.Run(data, client, appCtx, redisCfg.HashKey)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("seed completed: %d keys written to hash=%q\n", len(data), redisCfg.HashKey)
}
