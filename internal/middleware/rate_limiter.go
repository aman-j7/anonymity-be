package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type ActionLimit struct {
	Limit  int
	Window time.Duration
}

type RateLimiter struct {
	redis  *redis.Client
	limits map[string]ActionLimit
}


func NewRateLimiter(redisClient *redis.Client, limits map[string]ActionLimit) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		limits: limits,
	}
}

func (r *RateLimiter) IsLimited(ctx context.Context, roomID, playerID, action string) (bool, error) {
	limitConfig, exists := r.limits[action]

	if !exists {
		return false, nil
	}

	key := fmt.Sprintf("rate_limit:%s:%s:%s", roomID, playerID, action)

	count, err := r.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		err = r.redis.Expire(ctx, key, limitConfig.Window).Err()
		if err != nil {
			return false, err
		}
	}

	if count > int64(limitConfig.Limit) {
		return true, nil
	}

	return false, nil
}


func HTTPRateLimiter(rl *RateLimiter, action string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			playerID := r.RemoteAddr
			roomID := r.URL.Query().Get("room")

			limited, err := rl.IsLimited(context.Background(), roomID, playerID, action)
			if err != nil {
				http.Error(w, "Rate limiter error", http.StatusInternalServerError)
				return
			}
			if limited {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}

			next(w, r)
		}
	}
}