package redis

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Address  string `env:"REDIS_ADDR" env-required:"true"`
	User     string `env:"REDIS_USER"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB" env-default:"0"`
	HashKey  string `env:"REDIS_HASH_KEY" env-required:"true"`
}

func LoadRedisConfig() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
