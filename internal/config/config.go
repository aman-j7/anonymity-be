package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string

	RedisAddr     string
	ElasticURL    string
	MaxRoomCount  int
	RoomBatchSize int
}

func Load() *Config {

	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using system env")
	}

	cfg := &Config{
		Port:       os.Getenv("PORT"),
		RedisAddr:  os.Getenv("REDIS_ADDR"),
		ElasticURL: os.Getenv("ELASTIC_URL"),
	}

	return cfg
}
