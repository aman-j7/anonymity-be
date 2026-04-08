package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	RedisAddr        string
	ElasticURL       string
	MaxRoomCount     int
	RoomBatchSize    int
	OpenRouterApiKey string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	parseInt := func(key string) int {
		valStr := os.Getenv(key)
		if valStr == "" {
			log.Fatalf("Environment variable %s is required", key)
		}
		val, err := strconv.Atoi(valStr)
		if err != nil {
			log.Fatalf("Invalid integer value for %s: %v", key, err)
		}
		return val
	}

	return &Config{
		Port:             mustGetEnv("PORT"),
		RedisAddr:        mustGetEnv("REDIS_ADDR"),
		ElasticURL:       mustGetEnv("ELASTIC_URL"),
		OpenRouterApiKey: mustGetEnv("OPEN_ROUTER_API_KEY"),
		MaxRoomCount:     parseInt("MAX_ROOM_COUNT"),
		RoomBatchSize:    parseInt("ROOM_BATCH_SIZE"),
	}
}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Environment variable %s is required", key)
	}
	return val
}