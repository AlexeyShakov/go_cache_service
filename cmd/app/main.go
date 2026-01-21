package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/yourname/go_cache_service/internal/cache"
	"github.com/yourname/go_cache_service/internal/infrastructure/redis"
	"github.com/yourname/go_cache_service/internal/service"
	"github.com/yourname/go_cache_service/internal/worker"
	"log"
	"time"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("no .env file found")
	}

	redisConfig, err := redis.LoadRedisConfig()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := redis.NewClient(ctx, redisConfig)
	if err != nil {
		log.Fatal(err)
	}
	redisRepository, err := redis.NewRedisRepository(db, redisConfig.HashKey)
	if err != nil {
		log.Fatal(err)
	}
	inMemoryCache := cache.NewInMemoryCache()
	cacheService := service.NewCacheService(redisRepository, inMemoryCache)
	refresherConfig, err := worker.LoadRefresherConfig()
	if err != nil {
		log.Fatal(err)
	}
	cacheRefresher := worker.NewRefresher(cacheService, refresherConfig)
	appCtx, cancel := context.WithCancel(context.Background())

	go cacheRefresher.Run(appCtx)
	fmt.Println(redisRepository)
}
