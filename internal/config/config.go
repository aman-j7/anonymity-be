package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	CleanupInterval time.Duration
	MaxIdleTime     time.Duration

	RedisAddr  string
	ElasticURL string
}

func Load() *Config {

	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using system env")
	}

	cfg := &Config{
		Port:            os.Getenv("PORT"),
		RedisAddr:       os.Getenv("REDIS_ADDR"),
		ElasticURL:      os.Getenv("ELASTIC_URL"),
		CleanupInterval: 60 * time.Second,
		MaxIdleTime:     5 * time.Minute,
	}

	return cfg
}
