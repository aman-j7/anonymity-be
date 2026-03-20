package infra

import (
	"anonymity/internal/config"
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func initRedis(cfg *config.Config) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		DB:           0,
		PoolSize:     50,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	Redis = client
	log.Println("✅ Redis initialized")
}
