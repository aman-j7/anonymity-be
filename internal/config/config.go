package config

import (
	"os"
	"time"
)

type Config struct {
	Port            string
	CleanupInterval time.Duration
	MaxIdleTime     time.Duration
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return &Config{
		Port:            port,
		CleanupInterval: 60 * time.Second,
		MaxIdleTime:     5 * time.Minute,
	}
}
