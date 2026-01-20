package redis

import (
	"os"
	"strconv"
)

type Config struct {
	Address  string
	User     string
	Password string
	DB       int
	HashKey  string
}

func LoadRedisConfig() (Config, error) {
	address := os.Getenv("REDIS_ADDR")
	user := os.Getenv("REDIS_USER")
	passw := os.Getenv("REDIS_PASSWORD")
	hashKey := os.Getenv("REDIS_HASH_KEY")
	db := 0
	if dbEnv := os.Getenv("REDIS_DB"); dbEnv != "" {
		parsed, err := strconv.Atoi(dbEnv)
		if err != nil {
			return Config{}, err
		}
		db = parsed
	}
	var cfg = Config{
		Address:  address,
		User:     user,
		Password: passw,
		HashKey:  hashKey,
		DB:       db,
	}
	return cfg, nil
}
