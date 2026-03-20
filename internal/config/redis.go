package config

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	Ctx         = context.Background()
	RedisClient *redis.Client
)

func InitRedis() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379" 
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "", 
		DB:           0,
		PoolSize:     50,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	pong, err := RedisClient.Ping(Ctx).Result()
	if err != nil {
		log.Fatalf("❌ Failed to connect to Redis at %s: %v", addr, err)
	}

	log.Printf("✅ Redis connected successfully at %s | PING: %s", addr, pong)
}
