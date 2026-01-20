package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/yourname/go_cache_service/internal/infrastructure/redis"
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
		panic(err)
	}
	redisRepository, err := redis.NewRedisRepository(db, redisConfig.HashKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(redisRepository)
}
